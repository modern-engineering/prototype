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
// This rung compiles solution and import clauses, extern and var
// declarations, default declarations (folded into the records they
// modify, provenance kept in each binding's Source), and deploy
// statements whose parameters are literals or symbol references.
// Every parsed construct beyond it (provision, sections, output
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

	packages map[string]*regPackage   // import path -> registration
	imports  map[string]importBinding // reference name -> import
	symbols  map[string]*symbol       // flat namespace: name -> first declaration

	verbDefaults map[string]*defaultBody            // statement verb -> its one default
	typeDefaults map[*catalogueElement]*defaultBody // element -> its one default

	records []image.Record
}

// A defaultBody is one collected default declaration: the position
// anchoring duplicate reports, the source layer its bindings carry
// into records, and a per-element cache of the prevalidated bindings
// it contributes.
type defaultBody struct {
	pos    token.Position
	body   *ast.Body
	source string

	folds map[*catalogueElement][]image.Binding
}

// A symbol is one row of the solution's flat namespace. Instance names,
// var symbols, and extern symbols share the one namespace (D-08's link
// model extended), so a name resolves to at most one declaration
// solution-wide.
type symbol struct {
	class string         // classVar, classExtern, or classInstance
	pos   token.Position // the declaring name, anchor of duplicate reports

	// Var symbols carry their literal twice: canonical for the image,
	// and as the text a referencing slot validates.
	value *image.Value
	text  string

	// Extern symbols carry their declared symbol type and its taint.
	typeRef   image.Ref
	sensitive bool
}

// The symbol classes of the flat namespace.
const (
	classVar      = "var"
	classExtern   = "extern"
	classInstance = "instance"
)

// A regPackage is one validated catalogue package registration.
type regPackage struct {
	path, name string
	elements   map[string]*catalogueElement
}

// A catalogueElement is one validated catalogue registration: a
// component together with its dry-extracted parameter schema, or a
// symbol type. A component's schema is extracted at most once, at the
// element's first reference or, for unreferenced elements, when the
// catalogue section is pinned.
type catalogueElement struct {
	pkgPath string
	pkgName string
	name    string
	kind    string // image.KindComponent or image.KindSymbol

	// Components:
	desc   *application.Descriptor
	dried  bool
	schema []image.ParamSchema
	keys   map[string]bool

	// Symbol types:
	symbol *SymbolType
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
		cfg:          cfg,
		packages:     make(map[string]*regPackage, len(cfg.Catalogue)),
		imports:      make(map[string]importBinding),
		symbols:      make(map[string]*symbol),
		verbDefaults: make(map[string]*defaultBody),
		typeDefaults: make(map[*catalogueElement]*defaultBody),
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
			ce := &catalogueElement{pkgPath: pkg.Path, pkgName: pkg.Name}
			switch el := el.(type) {
			case *appElement:
				ce.kind, ce.name, ce.desc = image.KindComponent, el.name, el.desc
			case *symbolElement:
				ce.kind, ce.name, ce.symbol = image.KindSymbol, el.name, el.typ
			default:
				return nil, fmt.Errorf("catalogue: package %q: element %d is nil", pkg.Path, j)
			}
			if !gotoken.IsIdentifier(ce.name) {
				return nil, fmt.Errorf("catalogue: package %q: element name %q is not a valid Go identifier", pkg.Path, ce.name)
			}
			if _, ok := rp.elements[ce.name]; ok {
				return nil, fmt.Errorf("catalogue: package %q: duplicate element %s", pkg.Path, ce.name)
			}
			switch ce.kind {
			case image.KindComponent:
				if ce.desc == nil {
					return nil, fmt.Errorf("catalogue: package %q: element %s has a nil descriptor", pkg.Path, ce.name)
				}
				if ce.desc.Make == nil {
					return nil, fmt.Errorf("catalogue: package %q: element %s: descriptor has no Make factory", pkg.Path, ce.name)
				}
			case image.KindSymbol:
				if ce.symbol == nil {
					return nil, fmt.Errorf("catalogue: package %q: element %s has a nil symbol type", pkg.Path, ce.name)
				}
			}
			rp.elements[ce.name] = ce
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

// check links the declarations in two phases. collect first enters
// every declared name of every unit into the solution's flat symbol
// namespace, so a statement may reference a symbol declared later or
// in another unit (cross-unit forward references are legal from the
// symbols rung on); the second phase then enforces this rung's
// language subset and resolves, dry-instantiates, and binds the
// statements in unit order. Diagnostics accumulate; only a panic in
// user code stops the walk.
func (ln *linker) check() {
	ln.collect()
	if ln.internal != nil {
		return
	}
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
			case *ast.ExternDecl, *ast.VarDecl, *ast.DefaultDecl:
				// Collected already; defaults fold at the records.
			case *ast.ProvisionDecl:
				ln.errorf(d.Pos(), "provision declarations not yet supported by this compiler rung")
			}
		}
	}
}

// collect walks every unit's declarations and enters each declared
// name — instance names, var symbols, and extern symbols — into the
// flat namespace, in unit-then-statement order. Provision names join
// at the rung that gives the statements meaning. Defaults collect in
// a second pass: their bodies may reference any symbol, so they wait
// for the namespace to be complete.
func (ln *linker) collect() {
	for _, f := range ln.files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.DeployDecl:
				for _, spec := range d.Specs {
					ln.declare(spec.Name, &symbol{class: classInstance, pos: spec.Name.NamePos})
				}
			case *ast.ExternDecl:
				for _, spec := range d.Specs {
					ln.extern(spec)
				}
			case *ast.VarDecl:
				for _, spec := range d.Specs {
					ln.varSymbol(spec)
				}
			}
		}
	}
	for _, f := range ln.files {
		for _, decl := range f.Decls {
			if d, ok := decl.(*ast.DefaultDecl); ok {
				ln.collectDefault(d)
				if ln.internal != nil {
					return
				}
			}
		}
	}
}

// collectDefault enters one default declaration. The target — a
// component type or a statement verb — dedups solution-wide: linking
// is an order-insensitive union, so a second default for the same
// target is a fault wherever it lives, never a nearer-wins layer.
// Section bodies stay a later rung. A type-scoped body validates
// eagerly against its element, records or none, so a broken default
// cannot hide behind an undeployed element; a verb-scoped body is
// element-dependent and validates at fold, so only its context-free
// references resolve here.
func (ln *linker) collectDefault(d *ast.DefaultDecl) {
	switch t := d.Target.(type) {
	case *ast.Ident: // the statement verbs: deploy, provision
		if first, ok := ln.verbDefaults[t.Name]; ok {
			ln.errorf(d.Keyword, "duplicate default for %s (first declared at %s)", t.Name, first.pos)
			return
		}
		source := image.SourceDefaultDeploy
		if t.Name != "deploy" {
			// Provision records arrive at a later rung; until they do
			// this default never folds, and its source value is that
			// rung's to choose.
			source = ""
		}
		ln.verbDefaults[t.Name] = &defaultBody{pos: d.Keyword, body: d.Body, source: source}
		ln.checkDefaultBody(d.Body, true)

	case *ast.TypeRef:
		elem := ln.resolve(t)
		if elem != nil && elem.kind != image.KindComponent {
			ln.errorf(t.Pos(), "cannot default %s: a symbol type takes no parameters", refString(t))
			elem = nil
		}
		if elem == nil {
			ln.checkDefaultBody(d.Body, true)
			return
		}
		if first, ok := ln.typeDefaults[elem]; ok {
			ln.errorf(d.Keyword, "duplicate default for %s (first declared at %s)", refString(t), first.pos)
			return
		}
		db := &defaultBody{pos: d.Keyword, body: d.Body, source: image.SourceDefaultType}
		ln.typeDefaults[elem] = db
		ln.checkDefaultBody(d.Body, false)
		if !ln.dry(elem, t) {
			return
		}
		ln.defaultFolds(db, elem, t, true)
	}
}

// checkDefaultBody screens a default body at collection: sections are
// a later rung, and — for bodies that will not validate against an
// element right away (checkRefs) — symbol references resolve now,
// context-free, so a dangling name surfaces exactly once even if no
// record ever folds the default.
func (ln *linker) checkDefaultBody(body *ast.Body, checkRefs bool) {
	for _, item := range body.Items {
		switch it := item.(type) {
		case *ast.Section:
			ln.errorf(it.Name.NamePos, "%s sections not yet supported by this compiler rung", it.Name.Name)
		case *ast.Param:
			if !checkRefs {
				continue
			}
			if ref, isRef := it.Value.(*ast.RefExpr); isRef {
				ln.resolveRef(ref)
			}
		}
	}
}

// extern declares one extern symbol: a value the deploying site must
// bind, of a declared symbol type. A type that fails to resolve still
// declares the name — the fault is the type's, already diagnosed, and
// the use sites need no echo of it.
func (ln *linker) extern(spec *ast.ExternSpec) {
	sym := &symbol{class: classExtern, pos: spec.Name.NamePos}
	if elem := ln.resolveSymbolType(spec.Type); elem != nil {
		sym.typeRef = image.Ref{Package: elem.pkgPath, Name: elem.name}
		sym.sensitive = elem.symbol.Sensitive
	}
	ln.declare(spec.Name, sym)
}

// resolveSymbolType resolves an extern's type reference and checks
// that it names a symbol-type element; nil (with the fault diagnosed)
// otherwise.
func (ln *linker) resolveSymbolType(t *ast.TypeRef) *catalogueElement {
	elem := ln.resolve(t)
	if elem == nil {
		return nil
	}
	if elem.kind != image.KindSymbol {
		ln.errorf(t.Pos(), "element %s is a %s, not a symbol type", refString(t), elem.kind)
		return nil
	}
	return elem
}

// varSymbol declares one var symbol, carrying its literal both in
// canonical image form and as the text a referencing slot validates.
func (ln *linker) varSymbol(spec *ast.VarSpec) {
	val, text, ok := literalValue(spec.Value)
	if !ok {
		// Unreachable past a clean parse: the grammar admits literals
		// only, and syntax gates linking.
		return
	}
	ln.declare(spec.Name, &symbol{class: classVar, pos: spec.Name.NamePos, value: val, text: text})
}

// declare enters one named declaration into the flat namespace. The
// first declaration owns the name; any second one, whatever the
// classes involved, is a duplicate symbol diagnosed at the later
// declaration, carrying the owning position.
func (ln *linker) declare(name *ast.Ident, sym *symbol) {
	if first, ok := ln.symbols[name.Name]; ok {
		ln.errorf(name.NamePos, "duplicate symbol %s (first declared at %s)", name.Name, first.pos)
		return
	}
	ln.symbols[name.Name] = sym
}

// deploy checks one deploy spec end to end; a fully clean spec appends a
// record. The spec's name was declared by collect: the spec owns the
// name iff the declaring position is its own, and a duplicate (already
// diagnosed there) must not append a second record for it.
func (ln *linker) deploy(spec *ast.DeploySpec) {
	sym := ln.symbols[spec.Name.Name]
	ok := sym != nil && sym.pos == spec.Name.NamePos
	elem := ln.resolve(spec.Type)
	if elem != nil && elem.kind != image.KindComponent {
		ln.errorf(spec.Type.Pos(), "cannot deploy %s: element is a symbol type, not a component", refString(spec.Type))
		elem = nil
	}
	if elem == nil {
		ok = false
	} else if !ln.dry(elem, spec.Type) {
		return
	}
	bindings, bindOK := ln.bind(elem, spec)
	if ln.internal != nil {
		return
	}
	if elem != nil {
		bindings = ln.fold(elem, spec, bindings)
		if ln.internal != nil {
			return
		}
	}
	if !ok || !bindOK {
		return
	}
	ln.records = append(ln.records, image.Record{
		Verb:    image.VerbDeploy,
		Element: image.Ref{Package: elem.pkgPath, Name: elem.name},
		Name:    spec.Name.Name,
		Params:  bindings,
	})
}

// fold merges the default layers under a statement's own bindings.
// The nearest layer wins a key: the instance over the element-scoped
// default over the verb-scoped one; the catalogue's slot defaults are
// no layer at all — a binding exists iff some SDL statement set it,
// and unset slots stay with the pinned schema. Source keeps each
// surviving binding's provenance (image.Equal masks it).
func (ln *linker) fold(elem *catalogueElement, spec *ast.DeploySpec, instance []image.Binding) []image.Binding {
	verb, typed := ln.verbDefaults["deploy"], ln.typeDefaults[elem]
	if verb == nil && typed == nil {
		return instance
	}
	merged := make(map[string]image.Binding)
	if verb != nil {
		for _, b := range ln.defaultFolds(verb, elem, spec.Type, false) {
			merged[b.Key] = b
		}
		if ln.internal != nil {
			return nil
		}
	}
	if typed != nil {
		for _, b := range ln.defaultFolds(typed, elem, spec.Type, true) {
			merged[b.Key] = b
		}
	}
	for _, b := range instance {
		merged[b.Key] = b
	}
	bindings := make([]image.Binding, 0, len(merged))
	for _, key := range slices.Sorted(maps.Keys(merged)) {
		bindings = append(bindings, merged[key])
	}
	return bindings
}

// defaultFolds returns d's bindings as they apply to elem, building
// and caching them on first use: one validation per element, however
// many records fold the default, so a fault in a default body
// surfaces once.
func (ln *linker) defaultFolds(d *defaultBody, elem *catalogueElement, at *ast.TypeRef, strict bool) []image.Binding {
	if d.folds == nil {
		d.folds = make(map[*catalogueElement][]image.Binding)
	}
	if bs, ok := d.folds[elem]; ok {
		return bs
	}
	bs := ln.buildDefault(d, elem, at, strict)
	d.folds[elem] = bs
	return bs
}

// buildDefault validates a default body against one element and
// returns the bindings it contributes, unsorted (fold merges by key).
// strict — the element-scoped path — holds the body to the element's
// schema exactly as an instance body: unknown parameters are faults
// and references resolve loudly. The verb-scoped path skips
// parameters the element does not declare (a verb-wide default is a
// broad brush over heterogeneous elements) and resolves references
// quietly, their context-free faults already diagnosed at collection.
func (ln *linker) buildDefault(d *defaultBody, elem *catalogueElement, at *ast.TypeRef, strict bool) []image.Binding {
	sf := &surface{ln: ln, elem: elem, at: at}
	var bindings []image.Binding
	for _, item := range d.body.Items {
		it, ok := item.(*ast.Param)
		if !ok {
			continue // sections were diagnosed at collection
		}
		var b image.Binding
		var bound bool
		switch ref, isRef := it.Value.(*ast.RefExpr); {
		case strict:
			b, bound = ln.bindParam(sf, it, d.source)
		case !elem.keys[it.Key.Name]:
			// Not this element's parameter; the default passes it by.
		case isRef:
			if sym := ln.lookupValueSymbol(ref); sym != nil {
				b, bound = ln.bindSymbol(sf, it.Key, ref, sym, d.source)
			}
		default:
			b, bound = ln.bindParam(sf, it, d.source)
		}
		if ln.internal != nil {
			return nil
		}
		if bound {
			bindings = append(bindings, b)
		}
	}
	return bindings
}

// lookupValueSymbol returns the var or extern symbol a reference
// names, or nil quietly: the context-free faults (dangling, dotted,
// instance-valued) were diagnosed when the reference was collected.
func (ln *linker) lookupValueSymbol(ref *ast.RefExpr) *symbol {
	if ref.Sel != nil {
		return nil
	}
	sym := ln.symbols[ref.X.Name]
	if sym == nil || sym.class == classInstance {
		return nil
	}
	return sym
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

// bind checks a spec's body and binds its parameters, sorted by key.
// Literal parameters validate through a throwaway service's own
// flag.Value.Set — a fresh Make per statement, the very surface the
// running instance will parse with, so no state leaks between
// statements or into the pinned schema; the canonical bound value is
// owned by the SDL literal kind, and the flag's own String() is never
// read back. Symbol references resolve against the flat namespace and
// are recorded as references — records carry references, never
// inlined symbol values (A-10).
//
// With elem nil (an unresolved reference) the body is still walked —
// symbol references resolve and section diagnostics fire — but keys
// and values go unchecked.
func (ln *linker) bind(elem *catalogueElement, spec *ast.DeploySpec) (bindings []image.Binding, ok bool) {
	ok = true
	if spec.Body == nil {
		return nil, true
	}
	sf := &surface{ln: ln, elem: elem, at: spec.Type}
	for _, item := range spec.Body.Items {
		switch it := item.(type) {
		case *ast.Section:
			ln.errorf(it.Name.NamePos, "%s sections not yet supported by this compiler rung", it.Name.Name)
			ok = false

		case *ast.Param:
			b, bound := ln.bindParam(sf, it, image.SourceInstance)
			if ln.internal != nil {
				return nil, false
			}
			if !bound {
				ok = false
				continue
			}
			bindings = append(bindings, b)
		}
	}
	slices.SortFunc(bindings, func(a, b image.Binding) int {
		return strings.Compare(a.Key, b.Key)
	})
	return bindings, ok
}

// bindParam binds one body parameter with the given provenance: a
// literal validates through its slot and lands as a canonical value, a
// symbol reference resolves and lands as a reference. It reports
// whether a binding was produced; a false return has recorded its
// diagnostic (or aborted through ln.internal), except for the
// *ast.BadValue the parse phase already reported.
func (ln *linker) bindParam(sf *surface, it *ast.Param, source string) (image.Binding, bool) {
	if ref, isRef := it.Value.(*ast.RefExpr); isRef {
		sym := ln.resolveRef(ref)
		if sym == nil {
			return image.Binding{}, false
		}
		return ln.bindSymbol(sf, it.Key, ref, sym, source)
	}
	val, text, isLiteral := literalValue(it.Value)
	if !isLiteral || sf.elem == nil {
		return image.Binding{}, false
	}
	f := sf.slot(it.Key)
	if f == nil {
		return image.Binding{}, false
	}
	panicked, err := setFlag(f.Value, text)
	if panicked != nil {
		ln.internal = &internalError{
			pos: it.Value.Pos(),
			msg: fmt.Sprintf("element %s: parameter %s: Set panicked: %v", refString(sf.at), it.Key.Name, panicked),
		}
		return image.Binding{}, false
	}
	if err != nil {
		ln.errorf(it.Value.Pos(), "invalid value for parameter %s: %v", it.Key.Name, err)
		return image.Binding{}, false
	}
	return image.Binding{Key: it.Key.Name, Value: val, Source: source}, true
}

// resolveRef resolves a value-position reference against the flat
// namespace to a value-carrying symbol; nil (with the fault diagnosed)
// otherwise.
func (ln *linker) resolveRef(ref *ast.RefExpr) *symbol {
	if ref.Sel != nil {
		ln.errorf(ref.Pos(), "output reference %s not yet supported by this compiler rung: provision outputs arrive at a later rung", refExprString(ref))
		return nil
	}
	sym, ok := ln.symbols[ref.X.Name]
	if !ok {
		ln.errorf(ref.Pos(), "undefined symbol %s", ref.X.Name)
		return nil
	}
	if sym.class == classInstance {
		ln.errorf(ref.Pos(), "instance %s has no value", ref.X.Name)
		return nil
	}
	return sym
}

// bindSymbol binds one resolved var or extern reference. A var's
// literal validates through the slot exactly as an inline literal
// would — the symbol table is compile-bound — but the binding records
// the reference, never the value. An extern has no value to validate
// at compile time (the site binds it at reification, against the same
// slot, at stage (c)); the reference alone is recorded, and a
// sensitive symbol type taints the binding.
func (ln *linker) bindSymbol(sf *surface, key *ast.Ident, ref *ast.RefExpr, sym *symbol, source string) (image.Binding, bool) {
	if sf.elem == nil {
		return image.Binding{}, false
	}
	b := image.Binding{
		Key:       key.Name,
		Ref:       &image.SymbolRef{Symbol: ref.X.Name},
		Source:    source,
		Sensitive: sym.sensitive,
	}
	switch sym.class {
	case classVar:
		f := sf.slot(key)
		if f == nil {
			return image.Binding{}, false
		}
		panicked, err := setFlag(f.Value, sym.text)
		if panicked != nil {
			ln.internal = &internalError{
				pos: ref.Pos(),
				msg: fmt.Sprintf("element %s: parameter %s: Set panicked: %v", refString(sf.at), key.Name, panicked),
			}
			return image.Binding{}, false
		}
		if err != nil {
			ln.errorf(ref.Pos(), "invalid value for parameter %s: var %s: %v", key.Name, ref.X.Name, err)
			return image.Binding{}, false
		}
	case classExtern:
		if !sf.known(key) {
			return image.Binding{}, false
		}
	}
	return b, true
}

// A surface is one element's throwaway binding surface: a fresh dry
// instance whose flag.Values validate the body's texts, made at most
// once per statement so its parameters share state the way one running
// instance's would — and none of it leaks across statements or into
// the pinned schema.
type surface struct {
	ln   *linker
	elem *catalogueElement // nil when the element reference did not resolve
	at   *ast.TypeRef      // the referencing statement, for panic attribution
	fs   *flag.FlagSet
}

// known reports whether the element declares the parameter, diagnosing
// an unknown key.
func (s *surface) known(key *ast.Ident) bool {
	if s.elem.keys[key.Name] {
		return true
	}
	s.ln.errorf(key.NamePos, "unknown parameter %s: element %s has no such parameter", key.Name, refString(s.at))
	return false
}

// slot returns the parameter's flag on the throwaway surface, dry-made
// on first use. Nil means the parameter cannot be bound: an unknown
// key (diagnosed here) or an abort through ln.internal.
func (s *surface) slot(key *ast.Ident) *flag.Flag {
	if !s.known(key) {
		return nil
	}
	if s.fs == nil {
		fs, panicked := dryFlags(s.elem.desc)
		if panicked != nil {
			s.ln.internal = &internalError{
				pos: s.at.Pos(),
				msg: fmt.Sprintf("element %s: Make panicked: %v", refString(s.at), panicked),
			}
			return nil
		}
		s.fs = fs
	}
	f := lookupFlag(s.fs, key.Name)
	if f == nil {
		// The schema knows the key but a fresh instance does not: the
		// factory broke the dry-instantiation invariant.
		s.ln.internal = &internalError{
			pos: s.at.Pos(),
			msg: fmt.Sprintf("element %s: Make broke the dry-instantiation invariant: fresh instance lacks parameter %s", refString(s.at), key.Name),
		}
		return nil
	}
	return f
}

// emit pins the catalogue and assembles the canonical image: every
// registered package — referenced or not, since the registration is
// what this compilation was checked against — sorted by path, elements
// by name, symbols by name, records in unit-then-statement order.
// Pinning a component not yet dried instantiates it here; without a
// referencing statement a panic is reported positionless.
func (ln *linker) emit() (*image.Image, *internalError) {
	catalogue := make([]image.Package, 0, len(ln.packages))
	for _, path := range slices.Sorted(maps.Keys(ln.packages)) {
		pkg := ln.packages[path]
		elements := make([]image.ElementSchema, 0, len(pkg.elements))
		for _, name := range slices.Sorted(maps.Keys(pkg.elements)) {
			elem := pkg.elements[name]
			es := image.ElementSchema{Name: elem.name, Kind: elem.kind}
			switch elem.kind {
			case image.KindComponent:
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
				es.Doc, es.Params = elem.desc.Doc, elem.schema
			case image.KindSymbol:
				es.Doc, es.Sensitive = elem.symbol.Doc, elem.symbol.Sensitive
			}
			elements = append(elements, es)
		}
		catalogue = append(catalogue, image.Package{
			Path:     pkg.path,
			Name:     pkg.name,
			Elements: elements,
		})
	}

	symbols := []image.SymbolDef{}
	for _, name := range slices.Sorted(maps.Keys(ln.symbols)) {
		sym := ln.symbols[name]
		switch sym.class {
		case classVar:
			symbols = append(symbols, image.SymbolDef{Name: name, Class: image.ClassVar, Value: sym.value})
		case classExtern:
			typeRef := sym.typeRef
			symbols = append(symbols, image.SymbolDef{Name: name, Class: image.ClassExtern, Type: &typeRef})
		}
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
		Symbols:    symbols,
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
