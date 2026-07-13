// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Fieldcase-gen generates deployment artifacts from a compiled solution
// image: the stage-2 half of the fieldcase validation ground. It is the
// image's first scenario consumer — everything it emits derives from the
// image and a site file, never from the solution's SDL source.
//
// Usage:
//
//	fieldcase-gen -image <image.json> -site <file.site> -mode host -o <dir>
//	fieldcase-gen -image <image.json> -site <file.site> -mode k8s  -o <dir>
//
// Host mode emits a single-process host program (package main) that
// bakes the image's records and links the catalogue packages the
// image's element refs name; the site file is validated here but read
// again by the host at startup, so one generated host serves every
// site of the solution and no site value — in particular no secret —
// is baked into source. K8s mode emits one static YAML per deploy
// record (ConfigMap, optional Secret, Deployment): there the site's
// values ARE baked, split per instance into a non-sensitive ConfigMap
// fragment and a sensitive Secret fragment.
//
// In both modes the site must bind every extern the solution's
// instances reach (the stage-(c) MUST-bind rule): a gap is an error
// quoting the symbol and its type.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/modern-engineering/prototype/fieldcase/host"
	"github.com/modern-engineering/prototype/fieldcase/host/site"
	"github.com/modern-engineering/prototype/solution/image"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("fieldcase-gen", flag.ContinueOnError)
	imagePath := fs.String("image", "", "compiled solution image (JSON) to generate from")
	sitePath := fs.String("site", "", "site file binding the image's externs")
	mode := fs.String("mode", "", "artifact to generate: host or k8s")
	outDir := fs.String("o", "", "output directory (created if absent)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *imagePath == "" || *sitePath == "" || *outDir == "" || (*mode != "host" && *mode != "k8s") {
		fmt.Fprintln(os.Stderr, "fieldcase-gen: -image, -site, -o, and -mode {host,k8s} are all required")
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
		return 2
	}

	f, err := os.Open(*imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
		return 1
	}
	img, err := image.Decode(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
		return 1
	}

	m, err := analyze(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %s: %v\n", *imagePath, err)
		return 1
	}

	st, err := site.Load(*sitePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
		return 1
	}
	if faults := m.cfg.CheckSite(st); len(faults) > 0 {
		for _, fault := range faults {
			fmt.Fprintf(os.Stderr, "fieldcase-gen: %s: %s\n", *sitePath, fault)
		}
		return 1
	}

	// The MUST-bind gate at generation time: the artifact set serves
	// every instance, so the closure is computed over all of them.
	all := make([]string, len(m.cfg.Instances))
	for i, inst := range m.cfg.Instances {
		all[i] = inst.Name
	}
	needs := m.cfg.Needs(all)
	if missing := m.cfg.UnboundExterns(st, needs); len(missing) > 0 {
		for _, sym := range missing {
			fmt.Fprintf(os.Stderr, "fieldcase-gen: unbound extern %s (%s): the site must bind extern.%s\n",
				sym.Name, sym.Type, sym.Name)
		}
		return 1
	}
	// Sites must also cover every driver output the run consumes, or
	// the emitted artifacts would fail their first startup.
	var gaps []string
	for _, p := range m.cfg.Provisions {
		section := st.Driver(p.Name)
		for _, out := range sortedNames(needs.Outputs[p.Name]) {
			if _, ok := section[out]; !ok {
				gaps = append(gaps, fmt.Sprintf("driver.%s.%s (%s %s)", p.Name, out, p.Kind, p.Type))
			}
		}
	}
	if len(gaps) > 0 {
		for _, g := range gaps {
			fmt.Fprintf(os.Stderr, "fieldcase-gen: site does not configure consumed output %s\n", g)
		}
		return 1
	}

	if err := os.MkdirAll(*outDir, 0o777); err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
		return 1
	}
	var files map[string][]byte
	switch *mode {
	case "host":
		files, err = generateHost(m)
	case "k8s":
		files, err = generateK8s(m, st)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
		return 1
	}
	for _, name := range sortedFileNames(files) {
		path := filepath.Join(*outDir, name)
		if err := os.WriteFile(path, files[name], 0o666); err != nil {
			fmt.Fprintf(os.Stderr, "fieldcase-gen: %v\n", err)
			return 1
		}
	}
	fmt.Fprintf(os.Stderr, "fieldcase-gen: wrote %d file(s) to %s\n", len(files), *outDir)
	return 0
}

// A model is the analyzed image: the host-shaped config (analysis only;
// no descriptors, no drivers) plus profile-specific details emission
// needs.
type model struct {
	img *image.Image
	cfg *host.Config

	pkgName map[string]string // import path -> package name
	elems   map[image.Ref]*image.ElementSchema
	instRef map[string]image.Ref // deploy instance -> element ref
	deploys map[string]*image.Record
}

// analyze folds the image into the model, translating records and
// symbols into the host's baked-config shape.
func analyze(img *image.Image) (*model, error) {
	m := &model{
		img:     img,
		pkgName: make(map[string]string),
		elems:   make(map[image.Ref]*image.ElementSchema),
		instRef: make(map[string]image.Ref),
		deploys: make(map[string]*image.Record),
	}
	for _, pkg := range img.Catalogue {
		m.pkgName[pkg.Path] = pkg.Name
		for i := range pkg.Elements {
			m.elems[image.Ref{Package: pkg.Path, Name: pkg.Elements[i].Name}] = &pkg.Elements[i]
		}
	}

	cfg := &host.Config{Solution: img.Solution}
	for _, sym := range img.Symbols {
		hs := host.Symbol{Name: sym.Name, Class: sym.Class}
		switch sym.Class {
		case image.ClassExtern:
			if sym.Type == nil {
				return nil, fmt.Errorf("extern %s has no type reference", sym.Name)
			}
			hs.Type = m.display(*sym.Type)
			if el := m.elems[*sym.Type]; el != nil {
				hs.Sensitive = el.Sensitive
			}
		case image.ClassVar:
			if sym.Value == nil {
				return nil, fmt.Errorf("var %s has no value", sym.Name)
			}
			v, err := canonical(sym.Value)
			if err != nil {
				return nil, fmt.Errorf("var %s: %v", sym.Name, err)
			}
			hs.Default = v
		default:
			return nil, fmt.Errorf("symbol %s has unknown class %q", sym.Name, sym.Class)
		}
		cfg.Symbols = append(cfg.Symbols, hs)
	}

	for i := range img.Records {
		rec := &img.Records[i]
		params, err := bindings(rec.Params)
		if err != nil {
			return nil, fmt.Errorf("record %s: %v", rec.Name, err)
		}
		switch rec.Verb {
		case image.VerbDeploy:
			cfg.Instances = append(cfg.Instances, host.Instance{Name: rec.Name, Params: params})
			m.instRef[rec.Name] = rec.Element
			m.deploys[rec.Name] = rec
		case image.VerbProvision:
			el := m.elems[rec.Element]
			if el == nil {
				return nil, fmt.Errorf("record %s: element %s.%s is not in the catalogue", rec.Name, rec.Element.Package, rec.Element.Name)
			}
			p := host.Provision{
				Name:   rec.Name,
				Type:   m.display(rec.Element),
				Kind:   rec.Kind,
				Params: params,
			}
			for _, out := range el.Outputs {
				p.Outputs = append(p.Outputs, host.Output{Name: out.Name, Sensitive: out.Sensitive})
			}
			cfg.Provisions = append(cfg.Provisions, p)
		default:
			return nil, fmt.Errorf("record %s has unknown verb %q", rec.Name, rec.Verb)
		}
	}
	m.cfg = cfg
	return m, nil
}

// display renders an element reference as pkgname.Element for
// messages and generated identifiers.
func (m *model) display(ref image.Ref) string {
	return m.pkgName[ref.Package] + "." + ref.Name
}

// bindings translates image bindings to the host's canonical-string
// form.
func bindings(bs []image.Binding) ([]host.Binding, error) {
	out := make([]host.Binding, 0, len(bs))
	for _, b := range bs {
		hb := host.Binding{Key: b.Key, Sensitive: b.Sensitive, Source: b.Source}
		switch {
		case b.Value != nil:
			v, err := canonical(b.Value)
			if err != nil {
				return nil, fmt.Errorf("param %s: %v", b.Key, err)
			}
			hb.Kind, hb.Value = host.BindLiteral, v
		case b.Ref != nil && b.Ref.Output == "":
			hb.Kind, hb.Value = host.BindSymbol, b.Ref.Symbol
		case b.Ref != nil:
			hb.Kind, hb.Instance, hb.Output = host.BindOutput, b.Ref.Symbol, b.Ref.Output
		default:
			return nil, fmt.Errorf("param %s binds neither value nor reference", b.Key)
		}
		out = append(out, hb)
	}
	return out, nil
}

// canonical renders an image value as the string the flag surface
// consumes: exactly the spelling flag.Value.Set will parse back.
func canonical(v *image.Value) (string, error) {
	switch v.Kind {
	case image.KindString:
		return v.Str, nil
	case image.KindInt:
		return strconv.FormatInt(v.Int, 10), nil
	case image.KindBool:
		return strconv.FormatBool(v.Bool), nil
	case image.KindDuration:
		return v.Dur.String(), nil
	case image.KindToken:
		// Tokens live in the opaque compartments, outside the flag
		// surface.
		return v.Tok, nil
	}
	return "", fmt.Errorf("unknown value kind %q", v.Kind)
}

func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func sortedFileNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
