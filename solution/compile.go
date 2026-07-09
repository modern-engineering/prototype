// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"errors"
	"flag"
	"fmt"
	gotoken "go/token"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution/image"
)

// MainCompile is the compiler's back half: the generated main hands it
// the embedded units and the live catalogue, and it links them into the
// desired-state image, written to cfg.Output as canonical JSON.
//
// The return value is the process exit code:
//
//	0  the image was emitted
//	1  solution diagnostics; positioned "file:line:col: message" lines
//	   on cfg.Stderr, one per line, sorted
//	2  usage or internal errors: a malformed config or catalogue
//	   registration ("compile: message" lines), or a panicking element
//	   factory (positioned at the referencing statement when one exists)
//
// This rung compiles the I1 language subset — solution and import
// clauses plus deploy statements with literal parameters. Every parsed
// construct beyond it (extern, var, default, provision, sections, symbol
// references) is reported as a positioned diagnostic rather than
// silently dropped.
func MainCompile(cfg CompileConfig) int {
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	ln, err := newLinker(cfg)
	if err != nil {
		printf(stderr, "compile: %v\n", err)
		return 2
	}

	// Syntax gates linking: past this point every surviving AST is
	// structurally sound.
	if !ln.parse() {
		ln.report(stderr)
		return 1
	}

	ln.link()
	ln.check()
	if ln.internal != nil {
		// Print what the solution got told so far, then the abort.
		ln.report(stderr)
		printf(stderr, "%v\n", ln.internal)
		return 2
	}
	if len(ln.diags) > 0 {
		ln.report(stderr)
		return 1
	}

	img, ierr := ln.emit()
	if ierr != nil {
		printf(stderr, "%v\n", ierr)
		return 2
	}
	if err := img.Encode(output); err != nil {
		printf(stderr, "compile: %v\n", err)
		return 2
	}
	return 0
}

// A linker carries one compilation through its phases: parse the units,
// link their headers, check the declarations, emit the image. Faults in
// the solution's own text accumulate in diags (exit 1, collect-all);
// faults in the machine-supplied inputs or in user Go code abort through
// newLinker errors and internal (exit 2).
type linker struct {
	cfg CompileConfig

	files []*ast.File // parsed units, parallel to cfg.Units

	diags    scanner.ErrorList
	internal *internalError

	packages  map[string]*regPackage    // import path -> registration
	imports   map[string]importBinding  // reference name -> import
	instances map[string]token.Position // instance name -> first definition
	records   []image.Record
}

// A regPackage is one validated catalogue package registration.
type regPackage struct {
	path, name string
	elements   map[string]*catalogueElement
}

// A catalogueElement is one validated App registration together with its
// dry-extracted parameter schema. The schema is extracted at most once,
// at the element's first reference or, for unreferenced elements, when
// the catalogue section is pinned.
type catalogueElement struct {
	pkgPath string
	pkgName string
	name    string
	desc    *application.Descriptor

	dried  bool
	schema []image.ParamSchema
	keys   map[string]bool
}

// An importBinding is one entry of the union import table.
type importBinding struct {
	path string
	pos  token.Position // the binding import spec, for conflict reports
}

// An internalError aborts compilation with exit code 2. When the fault
// is attributable to a source statement, pos is valid and the report is
// positioned like a diagnostic; otherwise the line carries the
// "compile:" prefix.
type internalError struct {
	pos token.Position
	msg string
}

func (e *internalError) String() string {
	if e.pos.IsValid() {
		return e.pos.String() + ": " + e.msg
	}
	return "compile: " + e.msg
}

// newLinker validates the machine-supplied inputs — the config shape and
// the catalogue registrations, whose faults are usage errors rather than
// solution diagnostics — and indexes the catalogue for resolution.
func newLinker(cfg CompileConfig) (*linker, error) {
	if cfg.Solution == "" {
		return nil, errors.New("config: empty solution name")
	}
	if len(cfg.Units) == 0 {
		return nil, errors.New("config: no units to compile")
	}
	unitNames := make(map[string]bool, len(cfg.Units))
	for i, u := range cfg.Units {
		if u.Name == "" {
			return nil, fmt.Errorf("config: unit %d has an empty name", i)
		}
		if unitNames[u.Name] {
			return nil, fmt.Errorf("config: duplicate unit name %q", u.Name)
		}
		unitNames[u.Name] = true
	}

	ln := &linker{
		cfg:       cfg,
		packages:  make(map[string]*regPackage, len(cfg.Catalogue)),
		imports:   make(map[string]importBinding),
		instances: make(map[string]token.Position),
	}
	for i, pkg := range cfg.Catalogue {
		if pkg.Path == "" {
			return nil, fmt.Errorf("catalogue: package %d has an empty import path", i)
		}
		if !gotoken.IsIdentifier(pkg.Name) {
			return nil, fmt.Errorf("catalogue: package %q: name %q is not a valid Go identifier", pkg.Path, pkg.Name)
		}
		if _, ok := ln.packages[pkg.Path]; ok {
			return nil, fmt.Errorf("catalogue: package %q registered twice", pkg.Path)
		}
		rp := &regPackage{
			path:     pkg.Path,
			name:     pkg.Name,
			elements: make(map[string]*catalogueElement, len(pkg.Elements)),
		}
		for j, el := range pkg.Elements {
			app, ok := el.(*appElement)
			if !ok {
				return nil, fmt.Errorf("catalogue: package %q: element %d is nil", pkg.Path, j)
			}
			if !gotoken.IsIdentifier(app.name) {
				return nil, fmt.Errorf("catalogue: package %q: element name %q is not a valid Go identifier", pkg.Path, app.name)
			}
			if _, ok := rp.elements[app.name]; ok {
				return nil, fmt.Errorf("catalogue: package %q: duplicate element %s", pkg.Path, app.name)
			}
			if app.desc == nil {
				return nil, fmt.Errorf("catalogue: package %q: element %s has a nil descriptor", pkg.Path, app.name)
			}
			if app.desc.Make == nil {
				return nil, fmt.Errorf("catalogue: package %q: element %s: descriptor has no Make factory", pkg.Path, app.name)
			}
			rp.elements[app.name] = &catalogueElement{
				pkgPath: pkg.Path,
				pkgName: pkg.Name,
				name:    app.name,
				desc:    app.desc,
			}
		}
		ln.packages[pkg.Path] = rp
	}
	return ln, nil
}

// errorf records one positioned diagnostic.
func (ln *linker) errorf(pos token.Position, format string, args ...any) {
	ln.diags.Add(pos, fmt.Sprintf(format, args...))
}

// report prints the collected diagnostics, one "file:line:col: message"
// line each, in canonical position order.
func (ln *linker) report(w io.Writer) {
	ln.diags.Sort()
	for _, e := range ln.diags {
		printf(w, "%v\n", e)
	}
}

// parse parses every unit, aggregating all units' errors before failing
// so one broken file does not hide another's faults. It reports whether
// the phase was clean.
func (ln *linker) parse() bool {
	ln.files = make([]*ast.File, len(ln.cfg.Units))
	for i, u := range ln.cfg.Units {
		f, err := parser.ParseFile(u.Name, []byte(u.Source))
		ln.files[i] = f
		if err != nil {
			var list scanner.ErrorList
			if errors.As(err, &list) {
				ln.diags = append(ln.diags, list...)
			} else {
				ln.diags.Add(token.Position{Filename: u.Name}, err.Error())
			}
		}
	}
	return len(ln.diags) == 0
}

// link verifies the shared solution clause and unions the units' import
// tables into one reference-name table (D-08's link model: peer units,
// one namespace).
func (ln *linker) link() {
	for _, f := range ln.files {
		if f.Solution != nil && f.Solution.Name.Name != ln.cfg.Solution {
			ln.errorf(f.Solution.Name.NamePos,
				"solution mismatch: unit declares %s, want %s", f.Solution.Name.Name, ln.cfg.Solution)
		}
		for _, decl := range f.Imports {
			for _, spec := range decl.Specs {
				ln.addImport(spec)
			}
		}
	}
}

// addImport binds one import spec's reference name — its alias or, for
// unaliased specs, the registered package name of its path — into the
// union table. Without an alias the reference name lives in the imported
// package itself, which only the catalogue can supply, so an unaliased
// import of an unregistered path is reported at the spec; an aliased one
// is diagnosed at its first use instead. Rebinding a name to a different
// path is an error at the second spec.
func (ln *linker) addImport(spec *ast.ImportSpec) {
	path := spec.Path.Value
	var name string
	switch {
	case spec.Alias != nil:
		name = spec.Alias.Name
	default:
		pkg, ok := ln.packages[path]
		if !ok {
			ln.errorf(spec.Pos(), "import %q: package not registered in the compile catalogue", path)
			return
		}
		name = pkg.name
	}
	if prev, ok := ln.imports[name]; ok {
		if prev.path != path {
			ln.errorf(spec.Pos(), "import name %s already bound to %q (first imported at %s)", name, prev.path, prev.pos)
		}
		return
	}
	ln.imports[name] = importBinding{path: path, pos: spec.Pos()}
}

// check walks the declarations of every unit in order: it enforces this
// rung's language subset and resolves, checks, and records the deploy
// statements. Diagnostics accumulate; only a panic in user code stops
// the walk.
func (ln *linker) check() {
	for _, f := range ln.files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.DeployDecl:
				for _, spec := range d.Specs {
					ln.deploy(spec)
					if ln.internal != nil {
						return
					}
				}
			case *ast.ExternDecl:
				ln.errorf(d.Pos(), "extern declarations not yet supported by this compiler rung")
			case *ast.VarDecl:
				ln.errorf(d.Pos(), "var declarations not yet supported by this compiler rung")
			case *ast.DefaultDecl:
				ln.errorf(d.Pos(), "default declarations not yet supported by this compiler rung")
			case *ast.ProvisionDecl:
				ln.errorf(d.Pos(), "provision declarations not yet supported by this compiler rung")
			}
		}
	}
}

// deploy checks one deploy spec end to end; a fully clean spec appends a
// record.
func (ln *linker) deploy(spec *ast.DeploySpec) {
	ok := ln.declare(spec.Name)
	elem := ln.resolve(spec.Type)
	if elem == nil {
		ok = false
	} else if !ln.dry(elem, spec.Type) {
		return
	}
	bindings, bindOK := ln.bind(elem, spec)
	if !ok || !bindOK || ln.internal != nil {
		return
	}
	ln.records = append(ln.records, image.Record{
		Verb:    image.VerbDeploy,
		Element: image.Ref{Package: elem.pkgPath, Name: elem.name},
		Name:    spec.Name.Name,
		Params:  bindings,
	})
}

// declare enters an instance name into the solution's flat namespace,
// reporting a duplicate symbol at its second definition.
func (ln *linker) declare(name *ast.Ident) bool {
	if first, ok := ln.instances[name.Name]; ok {
		ln.errorf(name.NamePos, "duplicate symbol %s (first deployed at %s)", name.Name, first)
		return false
	}
	ln.instances[name.Name] = name.NamePos
	return true
}

// resolve maps a type reference through the union import table and the
// registered catalogue to its element; it reports and returns nil when
// any link of the chain is missing.
func (ln *linker) resolve(t *ast.TypeRef) *catalogueElement {
	if t.Pkg == nil {
		// The parser tolerates the bare legacy form in importless
		// units; at compile it is unresolvable all the same.
		ln.errorf(t.Pos(),
			"unqualified type reference %s: element references must be package-qualified through an import (e.g. pkg.%s)",
			t.Name.Name, t.Name.Name)
		return nil
	}
	imp, ok := ln.imports[t.Pkg.Name]
	if !ok {
		ln.errorf(t.Pos(), "package %s is not imported", t.Pkg.Name)
		return nil
	}
	pkg, ok := ln.packages[imp.path]
	if !ok {
		ln.errorf(t.Pos(), "package %q (imported as %s) is not registered in the compile catalogue", imp.path, t.Pkg.Name)
		return nil
	}
	elem, ok := pkg.elements[t.Name.Name]
	if !ok {
		ln.errorf(t.Pos(), "unknown element %s in package %q", t.Name.Name, pkg.path)
		return nil
	}
	return elem
}

// dry extracts elem's parameter schema by dry instantiation, at most
// once; a panicking factory becomes an internal fault attributed to the
// referencing statement. It reports whether the schema is usable.
func (ln *linker) dry(elem *catalogueElement, at *ast.TypeRef) bool {
	if elem.dried {
		return true
	}
	fs, panicked := dryFlags(elem.desc)
	if panicked != nil {
		ln.internal = &internalError{
			pos: at.Pos(),
			msg: fmt.Sprintf("element %s: Make panicked: %v", refString(at), panicked),
		}
		return false
	}
	elem.schema = paramSchemas(fs)
	elem.keys = make(map[string]bool, len(elem.schema))
	for _, p := range elem.schema {
		elem.keys[p.Name] = true
	}
	elem.dried = true
	return true
}

// bind checks a spec's body and binds its literal parameters, sorted by
// key. Validation runs each literal through a throwaway service's own
// flag.Value.Set — a fresh Make per statement, the very surface the
// running instance will parse with, so no state leaks between statements
// or into the pinned schema. The canonical bound value is owned by the
// SDL literal kind; the flag's own String() is never read back.
//
// With elem nil (an unresolved reference) the body is still walked for
// subset diagnostics, but keys and values go unchecked.
func (ln *linker) bind(elem *catalogueElement, spec *ast.DeploySpec) (bindings []image.Binding, ok bool) {
	ok = true
	if spec.Body == nil {
		return nil, true
	}
	var throwaway *flag.FlagSet // lazily dry-made, one per statement
	for _, item := range spec.Body.Items {
		switch it := item.(type) {
		case *ast.Section:
			ln.errorf(it.Name.NamePos, "%s sections not yet supported by this compiler rung", it.Name.Name)
			ok = false

		case *ast.Param:
			val, text, isLiteral := literalValue(it.Value)
			if !isLiteral {
				if ref, isRef := it.Value.(*ast.RefExpr); isRef {
					ln.errorf(ref.Pos(), "symbol reference %s not yet supported by this compiler rung", refExprString(ref))
				}
				// Anything else is an *ast.BadValue the parse phase
				// already reported.
				ok = false
				continue
			}
			if elem == nil {
				continue
			}
			if !elem.keys[it.Key.Name] {
				ln.errorf(it.Key.NamePos, "unknown parameter %s: element %s has no such parameter", it.Key.Name, refString(spec.Type))
				ok = false
				continue
			}
			if throwaway == nil {
				fs, panicked := dryFlags(elem.desc)
				if panicked != nil {
					ln.internal = &internalError{
						pos: spec.Type.Pos(),
						msg: fmt.Sprintf("element %s: Make panicked: %v", refString(spec.Type), panicked),
					}
					return nil, false
				}
				throwaway = fs
			}
			f := lookupFlag(throwaway, it.Key.Name)
			if f == nil {
				// The schema knows the key but a fresh instance does
				// not: the factory broke the dry-instantiation
				// invariant.
				ln.internal = &internalError{
					pos: spec.Type.Pos(),
					msg: fmt.Sprintf("element %s: Make broke the dry-instantiation invariant: fresh instance lacks parameter %s", refString(spec.Type), it.Key.Name),
				}
				return nil, false
			}
			panicked, err := setFlag(f.Value, text)
			if panicked != nil {
				ln.internal = &internalError{
					pos: it.Value.Pos(),
					msg: fmt.Sprintf("element %s: parameter %s: Set panicked: %v", refString(spec.Type), it.Key.Name, panicked),
				}
				return nil, false
			}
			if err != nil {
				ln.errorf(it.Value.Pos(), "invalid value for parameter %s: %v", it.Key.Name, err)
				ok = false
				continue
			}
			bindings = append(bindings, image.Binding{
				Key:    it.Key.Name,
				Value:  val,
				Source: image.SourceInstance,
			})
		}
	}
	slices.SortFunc(bindings, func(a, b image.Binding) int {
		return strings.Compare(a.Key, b.Key)
	})
	return bindings, ok
}

// emit pins the catalogue and assembles the canonical image: every
// registered package — referenced or not, since the registration is
// what this compilation was checked against — sorted by path, elements
// by name, records in unit-then-statement order. Pinning an element not
// yet dried instantiates it here; without a referencing statement a
// panic is reported positionless.
func (ln *linker) emit() (*image.Image, *internalError) {
	catalogue := make([]image.Package, 0, len(ln.packages))
	for _, path := range slices.Sorted(maps.Keys(ln.packages)) {
		pkg := ln.packages[path]
		elements := make([]image.ElementSchema, 0, len(pkg.elements))
		for _, name := range slices.Sorted(maps.Keys(pkg.elements)) {
			elem := pkg.elements[name]
			if !elem.dried {
				fs, panicked := dryFlags(elem.desc)
				if panicked != nil {
					return nil, &internalError{
						msg: fmt.Sprintf("element %s.%s: Make panicked: %v", elem.pkgName, elem.name, panicked),
					}
				}
				elem.schema = paramSchemas(fs)
				elem.dried = true
			}
			elements = append(elements, image.ElementSchema{
				Name:   elem.name,
				Kind:   image.KindComponent,
				Doc:    elem.desc.Doc,
				Params: elem.schema,
			})
		}
		catalogue = append(catalogue, image.Package{
			Path:     pkg.path,
			Name:     pkg.name,
			Elements: elements,
		})
	}

	records := ln.records
	if records == nil {
		records = []image.Record{}
	}
	generation := ln.cfg.Generation
	if generation == 0 {
		generation = 1
	}
	return &image.Image{
		Format:     image.Format,
		Solution:   ln.cfg.Solution,
		Generation: generation,
		Catalogue:  catalogue,
		Records:    records,
	}, nil
}

// dryFlags performs one recover-guarded dry instantiation: Make a fresh
// service, take its flag surface, discard the runner. A nil flag set is
// a flagless service. panicked carries any panic — from Make itself,
// from Flags, or from a Make that returned no service.
func dryFlags(d *application.Descriptor) (fs *flag.FlagSet, panicked any) {
	defer func() {
		if p := recover(); p != nil {
			fs, panicked = nil, p
		}
	}()
	return d.Make().Flags(), nil
}

// paramSchemas reads a dry flag surface into the pinned parameter
// schema, in flag.FlagSet.VisitAll order (lexicographic). Default pins
// the flag's DefValue verbatim as a schema fact; Boolean marks flags
// settable without a value.
func paramSchemas(fs *flag.FlagSet) []image.ParamSchema {
	if fs == nil {
		return nil
	}
	var params []image.ParamSchema
	fs.VisitAll(func(f *flag.Flag) {
		boolean := false
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
			boolean = bf.IsBoolFlag()
		}
		params = append(params, image.ParamSchema{
			Name:    f.Name,
			Usage:   f.Usage,
			Default: f.DefValue,
			Boolean: boolean,
		})
	})
	return params
}

// lookupFlag finds a declared flag on a throwaway surface; nil for a
// nil set.
func lookupFlag(fs *flag.FlagSet, name string) *flag.Flag {
	if fs == nil {
		return nil
	}
	return fs.Lookup(name)
}

// setFlag runs one validation Set, guarding against a panicking
// flag.Value implementation the same way dry instantiation is guarded.
func setFlag(v flag.Value, text string) (panicked any, err error) {
	defer func() {
		if p := recover(); p != nil {
			panicked, err = p, nil
		}
	}()
	return nil, v.Set(text)
}

// printf writes one diagnostic line. The write is best effort: a
// diagnostic writer that fails has nowhere better to hear about it, and
// the exit code already carries the verdict, so the write error is
// deliberately discarded — here, once, rather than at every call site.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// literalValue converts an SDL literal into its canonical image value
// and the string handed to flag.Value.Set for validation — the same
// string a host binds at run time. Canonical form is owned by the
// literal kind: strings bind as written, ints as normalized decimal
// digits, durations as the parsed time.Duration, bools as true or
// false.
func literalValue(v ast.Value) (val *image.Value, text string, ok bool) {
	switch lit := v.(type) {
	case *ast.StringLit:
		return image.String(lit.Value), lit.Value, true
	case *ast.IntLit:
		return image.Int(lit.Value), strconv.FormatInt(lit.Value, 10), true
	case *ast.DurationLit:
		return image.Duration(lit.Value), lit.Value.String(), true
	case *ast.BoolLit:
		return image.Bool(lit.Value), strconv.FormatBool(lit.Value), true
	}
	return nil, "", false
}

// refString renders a type reference as the unit wrote it.
func refString(t *ast.TypeRef) string {
	if t.Pkg != nil {
		return t.Pkg.Name + "." + t.Name.Name
	}
	return t.Name.Name
}

// refExprString renders a value-position reference as the unit wrote it.
func refExprString(r *ast.RefExpr) string {
	if r.Sel != nil {
		return r.X.Name + "." + r.Sel.Name
	}
	return r.X.Name
}
