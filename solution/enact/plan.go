// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package enact

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/image"
)

// A Plan is one image's enactment, resolved and ordered: the externs
// a site must bind, the provision steps in dependency order, and the
// deploy steps after them. [Load] is the one producer. The plan is
// data — it invokes nothing — and its steps point into the live
// catalogue it was loaded against, so it never leaves the process
// that loaded it.
type Plan struct {
	// Solution and Generation identify the desired state the plan
	// enacts, copied from the image header.
	Solution   string
	Generation int64

	// Externs are the late-bound symbols a deploying site must
	// supply, in the image's symbol-table order. Which values the
	// site actually supplies is deliberately not checked here: the
	// plan is site-agnostic, and gating unbound externs is the wet
	// half's first act.
	Externs []Extern

	// Provisions are the provision steps in the order the PROVISION
	// phase must run them: every step after the steps whose outputs
	// it references, ties broken by image record order.
	Provisions []Step

	// Deploys are the deploy steps in image record order, all of
	// them after every provision: reconcile-time outputs feed
	// application parameters, never the reverse.
	Deploys []Step
}

// An Extern is one late-bound symbol the deploying site must bind
// before the wet half runs (A-10's site-configuration moment).
type Extern struct {
	// Name is the symbol's solution-wide name, the key a site binds.
	Name string

	// Type references the extern's symbol-type element in the
	// image's pinned catalogue.
	Type image.Ref

	// Sensitive carries the symbol type's taint as the image pinned
	// it — the same authority that tainted the record bindings — so
	// a consumer's redaction can never disagree with the bindings'
	// own marks.
	Sensitive bool
}

// A Step is one image record joined with its resolved live element:
// the desired state plus the code the loading process links to
// realize it. Exactly one payload arm is non-nil, matching the
// record's verb. A step is data; making and running the element is
// the plan consumer's business.
type Step struct {
	// Record is the image record, verbatim.
	Record image.Record

	// Component is a deploy record's live descriptor: the wet half
	// Makes a fresh service from it and binds Record.Params into the
	// service's own flags. Nil for provision steps.
	Component *application.Descriptor

	// Provision is a provision record's live type: the wet half
	// Makes a fresh provisioner from it, binds Record.Params into
	// the provisioner's flags, and runs the driver against the
	// type's declared outputs. Nil for deploy steps.
	Provision *solution.ProvisionType
}

// Load plans img against the live catalogue cat: every record
// resolved to its live element and validated, provisions ordered
// along their output references, externs listed. Load is pure and
// deterministic — the same image and catalogue always plan the same
// way — and cheap enough to run once per process and phase, which is
// how a plan crosses binaries: it does not; each process loads its
// own. On any fault Load returns a nil plan and an error joining
// every fault found, one line each.
func Load(img *image.Image, cat []solution.Package) (*Plan, error) {
	if img == nil {
		return nil, errors.New("enact: nil image")
	}
	ld := newLoader(img, cat)
	externs := ld.externs()
	provisions, deploys := ld.steps()
	if len(ld.faults) == 0 {
		// Ordering assumes the reference closure the validation just
		// established; on a faulted image it would only pile
		// confusion onto already-reported faults.
		provisions = ld.order(provisions)
	}
	if len(ld.faults) > 0 {
		return nil, errors.Join(ld.faults...)
	}
	return &Plan{
		Solution:   img.Solution,
		Generation: img.Generation,
		Externs:    externs,
		Provisions: provisions,
		Deploys:    deploys,
	}, nil
}

// A loader is one Load in progress: the image, the indexed live
// catalogue, and the faults found so far.
type loader struct {
	img *image.Image

	// live indexes the catalogue registrations by element ref;
	// pinned indexes the image's own catalogue section the same way.
	live   map[image.Ref]solution.Registration
	pinned map[image.Ref]image.ElementSchema

	// names maps package import paths to package names for rendering
	// refs in faults, the live catalogue's spelling winning over the
	// image's pin.
	names map[string]string

	// symbols indexes the image's symbol table; provisions indexes
	// the image's provision records by instance name.
	symbols    map[string]image.SymbolDef
	provisions map[string]*image.Record

	faults []error
	seen   map[string]bool // fault lines already recorded
}

func newLoader(img *image.Image, cat []solution.Package) *loader {
	ld := &loader{
		img:        img,
		live:       make(map[image.Ref]solution.Registration),
		pinned:     make(map[image.Ref]image.ElementSchema),
		names:      make(map[string]string),
		symbols:    make(map[string]image.SymbolDef, len(img.Symbols)),
		provisions: make(map[string]*image.Record),
		seen:       make(map[string]bool),
	}
	for _, pkg := range img.Catalogue {
		ld.names[pkg.Path] = pkg.Name
		for _, es := range pkg.Elements {
			ld.pinned[image.Ref{Package: pkg.Path, Name: es.Name}] = es
		}
	}
	for _, pkg := range cat {
		ld.names[pkg.Path] = pkg.Name
		for _, el := range pkg.Elements {
			reg := solution.Unpack(el)
			ld.live[image.Ref{Package: pkg.Path, Name: reg.Name}] = reg
		}
	}
	for _, def := range img.Symbols {
		ld.symbols[def.Name] = def
	}
	for i := range img.Records {
		rec := &img.Records[i]
		if rec.Verb == image.VerbProvision {
			ld.provisions[rec.Name] = rec
		}
	}
	return ld
}

// fault records one fault line. Lines deduplicate: the image has no
// source positions to tell two sites of the same fault apart, so a
// second identical line would carry no new information.
func (ld *loader) fault(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if ld.seen[msg] {
		return
	}
	ld.seen[msg] = true
	ld.faults = append(ld.faults, errors.New(msg))
}

// display renders an element ref the way faults speak about
// elements: package name, dot, element name. The live catalogue's
// package spelling wins; the image's pin covers packages the process
// does not link; a package known to neither falls back to its
// quoted import path.
func (ld *loader) display(ref image.Ref) string {
	if name, ok := ld.names[ref.Package]; ok {
		return name + "." + ref.Name
	}
	return strconv.Quote(ref.Package) + "." + ref.Name
}

// externs lists the image's extern symbols with the type identity
// and sensitivity the image pinned for them. Vars stay out: their
// values ride the symbol table itself, so a site owes them nothing.
func (ld *loader) externs() []Extern {
	var externs []Extern
	for _, def := range ld.img.Symbols {
		if def.Class != image.ClassExtern {
			continue
		}
		if def.Type == nil {
			ld.fault("extern %s declares no type", def.Name)
			continue
		}
		es, ok := ld.pinned[*def.Type]
		if !ok {
			ld.fault("extern %s declares type %s but the image pins no such element", def.Name, ld.display(*def.Type))
			continue
		}
		if es.Kind != image.KindSymbol {
			ld.fault("extern %s declares type %s but the image pins a %s, not a symbol type", def.Name, ld.display(*def.Type), es.Kind)
			continue
		}
		externs = append(externs, Extern{Name: def.Name, Type: *def.Type, Sensitive: es.Sensitive})
	}
	return externs
}

// steps resolves every record into a step, splitting the phases:
// provision steps still in image order — order settles them later —
// and deploy steps in the image order they keep.
func (ld *loader) steps() (provisions, deploys []Step) {
	names := make(map[string]bool, len(ld.img.Records))
	for _, rec := range ld.img.Records {
		if names[rec.Name] {
			ld.fault("image declares instance %s twice", rec.Name)
			continue
		}
		names[rec.Name] = true
		switch rec.Verb {
		case image.VerbDeploy:
			if step, ok := ld.deploy(rec); ok {
				deploys = append(deploys, step)
			}
		case image.VerbProvision:
			if step, ok := ld.provision(rec); ok {
				provisions = append(provisions, step)
			}
		default:
			ld.fault("record %s declares unknown verb %q", rec.Name, rec.Verb)
		}
	}
	return provisions, deploys
}

// deploy resolves one deploy record against the live catalogue.
func (ld *loader) deploy(rec image.Record) (Step, bool) {
	elem := ld.display(rec.Element)
	reg, ok := ld.live[rec.Element]
	if !ok {
		ld.fault("image references %s but the catalogue registers no such element", elem)
		return Step{}, false
	}
	if reg.App == nil {
		ld.fault("cannot deploy %s: %s is %s, not a component", rec.Name, elem, noun(reg))
		return Step{}, false
	}
	return Step{Record: rec, Component: reg.App}, true
}

// provision resolves one provision record against the live catalogue
// and holds the record's kind to what enactment implements: attach
// only, and only where the live type still registers it. Slice
// records are refused wholesale — the slice lifecycle (create,
// mutate, destroy, prune) is unbuilt — even though they compile:
// citizenship in the image is dry, running is not.
func (ld *loader) provision(rec image.Record) (Step, bool) {
	elem := ld.display(rec.Element)
	reg, ok := ld.live[rec.Element]
	if !ok {
		ld.fault("image references %s but the catalogue registers no such element", elem)
		return Step{}, false
	}
	if reg.Provision == nil {
		ld.fault("cannot provision %s: %s is %s, not a provision type", rec.Name, elem, noun(reg))
		return Step{}, false
	}
	switch rec.Kind {
	case image.KindSlice:
		ld.fault("slice provisioning is not implemented: %s provisions %s as a slice (attach only)", rec.Name, elem)
	case image.KindAttach:
		if reg.Provision.Kinds&solution.Attach == 0 {
			ld.fault("image provisions %s as attach but %s does not register attach", rec.Name, elem)
		}
	default:
		ld.fault("record %s declares unknown provision kind %q", rec.Name, rec.Kind)
	}
	return Step{Record: rec, Provision: reg.Provision}, true
}

// noun names a registration's kind the way faults speak about it,
// article included.
func noun(reg solution.Registration) string {
	switch {
	case reg.App != nil:
		return "a component"
	case reg.Provision != nil:
		return "a provision type"
	case reg.Symbol != nil:
		return "a symbol type"
	}
	return "an empty registration"
}

// order arranges the provision steps so every step follows the steps
// whose outputs it references: Kahn's algorithm over the
// output-reference edges, extracting at each turn the earliest ready
// step in image record order — so independent steps keep the
// image's order, and a freed step runs as early as its position
// allows. The linker guarantees the edges acyclic (the binding DAG);
// meeting a cycle anyway means a hand-crafted image, and is a fault.
func (ld *loader) order(steps []Step) []Step {
	// deps[i] names the provision instances step i waits on. The
	// deployment, extensions, and metadata compartments stay outside
	// the binding DAG, so only Params contribute edges.
	deps := make([]map[string]bool, len(steps))
	for i, step := range steps {
		set := make(map[string]bool)
		for _, b := range step.Record.Params {
			if b.Ref != nil && b.Ref.Output != "" {
				set[b.Ref.Symbol] = true
			}
		}
		deps[i] = set
	}

	ordered := make([]Step, 0, len(steps))
	done := make(map[string]bool, len(steps))
	placed := make([]bool, len(steps))
	for len(ordered) < len(steps) {
		next := -1
		for i := range steps {
			if placed[i] {
				continue
			}
			ready := true
			for dep := range deps[i] {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				next = i
				break
			}
		}
		if next < 0 {
			var stuck []string
			for i, step := range steps {
				if !placed[i] {
					stuck = append(stuck, step.Record.Name)
				}
			}
			ld.fault("provision reference cycle: no order settles %s", strings.Join(stuck, ", "))
			return steps
		}
		placed[next] = true
		done[steps[next].Record.Name] = true
		ordered = append(ordered, steps[next])
	}
	return ordered
}
