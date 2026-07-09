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
// MainCompile compiles solution and import clauses, extern and var
// declarations, default declarations (folded into the records they
// modify, provenance kept in each binding's Source), and deploy and
// provision statements: parameters bind literals, symbol references,
// and provision-output references (a.b) — reference edges form a DAG,
// and cycles among provision outputs are link errors — while the on
// and metadata sections carry the statement's other compartments,
// outside the namespace and the DAG.
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
// into records, a per-element cache of the prevalidated parameter
// bindings it contributes, and its section compartments — validated
// once at collection, since sections are element-independent.
type defaultBody struct {
	pos    token.Position
	body   *ast.Body
	source string

	folds map[*catalogueElement][]image.Binding

	on, metadata []image.Binding
}

// A compartments holds one statement's three binding compartments,
// one per audience: params for the application (typed by the pinned
// schema), on for the environment controller (typed by the platform's
// profile), and metadata for nobody (carried through untouched).
type compartments struct {
	params, on, metadata []image.Binding
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

	// Instance symbols carry their statement's element, resolved
	// quietly at collection so output references can consult the
	// type's scheme wherever the statement lives; nil when the
	// reference does not resolve (the statement itself diagnoses
	// that, loudly, once).
	elem *catalogueElement
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
// component or provision type together with its dry-extracted
// parameter schema, or a symbol type. A parameter schema is extracted
// at most once, at the element's first reference or, for unreferenced
// elements, when the catalogue section is pinned.
type catalogueElement struct {
	pkgPath string
	pkgName string
	name    string
	kind    string // image.KindComponent, image.KindProvision, or image.KindSymbol

	// Components and provision types:
	dried  bool
	schema []image.ParamSchema
	keys   map[string]bool

	// Components:
	desc *application.Descriptor

	// Provision types:
	prov    *ProvisionType
	outputs map[string]*Output // by name; validated at registration

	// Symbol types:
	symbol *SymbolType
}

// String renders the element's registered identity, package name
// qualified — the spelling diagnostics fall back to when the unit's
// own spelling is out of reach.
func (ce *catalogueElement) String() string { return ce.pkgName + "." + ce.name }

// noun names the element's kind the way diagnostics speak about it.
func (ce *catalogueElement) noun() string {
	switch ce.kind {
	case image.KindProvision:
		return "provision type"
	case image.KindSymbol:
		return "symbol type"
	}
	return ce.kind
}

// factory names the element's dry-instantiation entry point for panic
// attribution: a component's Make, a provision type's Params.
func (ce *catalogueElement) factory() string {
	if ce.kind == image.KindProvision {
		return "Params"
	}
	return "Make"
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
			case *provisionElement:
				ce.kind, ce.name, ce.prov = image.KindProvision, el.name, el.typ
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
			case image.KindProvision:
				if err := checkProvisionType(ce); err != nil {
					return nil, fmt.Errorf("catalogue: package %q: element %s %v", pkg.Path, ce.name, err)
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

// checkProvisionType validates one provision-type registration and
// indexes its output scheme. Kinds are the author's explicit
// declaration (A-11): registering none is a fault here, never a
// default the compiler supplies.
func checkProvisionType(ce *catalogueElement) error {
	if ce.prov == nil {
		return errors.New("has a nil provision type")
	}
	if ce.prov.Kinds == 0 {
		return errors.New("registers no provision kinds (declare Slice, Attach, or both)")
	}
	if ce.prov.Kinds&^(Slice|Attach) != 0 {
		return fmt.Errorf("registers unknown provision kinds %#b", uint8(ce.prov.Kinds))
	}
	ce.outputs = make(map[string]*Output, len(ce.prov.Outputs))
	for i := range ce.prov.Outputs {
		out := &ce.prov.Outputs[i]
		if !gotoken.IsIdentifier(out.Name) {
			return fmt.Errorf("output name %q is not a valid Go identifier", out.Name)
		}
		if _, ok := ce.outputs[out.Name]; ok {
			return fmt.Errorf("declares output %s twice", out.Name)
		}
		ce.outputs[out.Name] = out
	}
	return nil
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
			case *ast.ProvisionDecl:
				for _, spec := range d.Specs {
					ln.provision(spec)
					if ln.internal != nil {
						return
					}
				}
			case *ast.ExternDecl, *ast.VarDecl, *ast.DefaultDecl:
				// Collected already; defaults fold at the records.
			}
		}
	}
	ln.checkCycles()
}

// collect walks every unit's declarations and enters each declared
// name — instance names, var symbols, and extern symbols — into the
// flat namespace, in unit-then-statement order. Instance symbols take
// their statement's element quietly, so output references resolve
// against the type's scheme wherever the statement lives; the loud
// resolution diagnostics stay with the statement check. Defaults
// collect in a second pass: their bodies may reference any symbol, so
// they wait for the namespace to be complete.
func (ln *linker) collect() {
	for _, f := range ln.files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.DeployDecl:
				for _, spec := range d.Specs {
					ln.declare(spec.Name, &symbol{class: classInstance, pos: spec.Name.NamePos, elem: ln.lookupElement(spec.Type)})
				}
			case *ast.ProvisionDecl:
				for _, spec := range d.Specs {
					ln.declare(spec.Name, &symbol{class: classInstance, pos: spec.Name.NamePos, elem: ln.lookupElement(spec.Type)})
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

// collectDefault enters one default declaration. The target — an
// element type or a statement verb — dedups solution-wide: linking
// is an order-insensitive union, so a second default for the same
// target is a fault wherever it lives, never a nearer-wins layer.
// A type-scoped body validates eagerly against its element, records
// or none, so a broken default cannot hide behind an undeployed
// element; a verb-scoped body is element-dependent and validates at
// fold, so only its context-free references resolve here. Section
// compartments are element-independent either way and collect once,
// here.
func (ln *linker) collectDefault(d *ast.DefaultDecl) {
	switch t := d.Target.(type) {
	case *ast.Ident: // the statement verbs: deploy, provision
		if first, ok := ln.verbDefaults[t.Name]; ok {
			ln.errorf(d.Keyword, "duplicate default for %s (first declared at %s)", t.Name, first.pos)
			return
		}
		source := image.SourceDefaultDeploy
		if t.Name == "provision" {
			source = image.SourceDefaultProvision
		}
		db := &defaultBody{pos: d.Keyword, body: d.Body, source: source}
		ln.verbDefaults[t.Name] = db
		db.on, db.metadata, _ = ln.sections(d.Body, source)
		ln.checkDefaultRefs(d.Body)

	case *ast.TypeRef:
		elem := ln.resolve(t)
		if elem != nil && elem.kind == image.KindSymbol {
			ln.errorf(t.Pos(), "cannot default %s: a symbol type takes no parameters", refString(t))
			elem = nil
		}
		if elem == nil {
			ln.sections(d.Body, image.SourceDefaultType)
			ln.checkDefaultRefs(d.Body)
			return
		}
		if first, ok := ln.typeDefaults[elem]; ok {
			ln.errorf(d.Keyword, "duplicate default for %s (first declared at %s)", refString(t), first.pos)
			return
		}
		db := &defaultBody{pos: d.Keyword, body: d.Body, source: image.SourceDefaultType}
		ln.typeDefaults[elem] = db
		db.on, db.metadata, _ = ln.sections(d.Body, image.SourceDefaultType)
		if !ln.dry(elem, t) {
			return
		}
		ln.defaultFolds(db, elem, t, true)
	}
}

// checkDefaultRefs screens a default body's parameter references at
// collection, for bodies that will not validate against an element
// right away: references resolve now, context-free, so a dangling
// name surfaces exactly once even if no record ever folds the
// default. Sections are collectDefault's business.
func (ln *linker) checkDefaultRefs(body *ast.Body) {
	for _, item := range body.Items {
		if it, isParam := item.(*ast.Param); isParam {
			if ref, isRef := it.Value.(*ast.RefExpr); isRef {
				ln.checkRef(ref)
			}
		}
	}
}

// checkRef resolves a value-position reference for its diagnostics
// alone, dispatching on its shape: bare references name value
// symbols, dotted ones name provision outputs.
func (ln *linker) checkRef(ref *ast.RefExpr) {
	if ref.Sel != nil {
		ln.resolveOutput(ref)
	} else {
		ln.resolveRef(ref)
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
		ln.errorf(t.Pos(), "element %s is a %s, not a symbol type", refString(t), elem.noun())
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
		ln.errorf(spec.Type.Pos(), "cannot deploy %s: element is a %s, not a component", refString(spec.Type), elem.noun())
		elem = nil
	}
	if elem == nil {
		ok = false
	} else if !ln.dry(elem, spec.Type) {
		return
	}
	inst, bindOK := ln.bind(elem, spec.Type, spec.Body)
	if ln.internal != nil {
		return
	}
	if elem != nil {
		inst = ln.fold(elem, image.VerbDeploy, spec.Type, inst)
		if ln.internal != nil {
			return
		}
	}
	if !ok || !bindOK {
		return
	}
	ln.records = append(ln.records, image.Record{
		Verb:     image.VerbDeploy,
		Element:  image.Ref{Package: elem.pkgPath, Name: elem.name},
		Name:     spec.Name.Name,
		Params:   inst.params,
		On:       inst.on,
		Metadata: inst.metadata,
	})
}

// provision checks one provision spec end to end, deploy's twin with
// the kind dimension added: the statement carries which access kind it
// holds — slice or attach — and the record always says so explicitly,
// the compiler resolving an omitted kind word while the type registers
// exactly one.
func (ln *linker) provision(spec *ast.ProvisionSpec) {
	sym := ln.symbols[spec.Name.Name]
	ok := sym != nil && sym.pos == spec.Name.NamePos
	elem := ln.resolve(spec.Type)
	if elem != nil && elem.kind != image.KindProvision {
		ln.errorf(spec.Type.Pos(), "cannot provision %s: element is a %s, not a provision type", refString(spec.Type), elem.noun())
		elem = nil
	}
	kind, kindOK := ln.provisionKind(spec, elem)
	if elem == nil {
		ok = false
	} else if !ln.dry(elem, spec.Type) {
		return
	}
	inst, bindOK := ln.bind(elem, spec.Type, spec.Body)
	if ln.internal != nil {
		return
	}
	if elem != nil {
		inst = ln.fold(elem, image.VerbProvision, spec.Type, inst)
		if ln.internal != nil {
			return
		}
	}
	if !ok || !bindOK || !kindOK {
		return
	}
	ln.records = append(ln.records, image.Record{
		Verb:     image.VerbProvision,
		Kind:     kind,
		Element:  image.Ref{Package: elem.pkgPath, Name: elem.name},
		Name:     spec.Name.Name,
		Params:   inst.params,
		On:       inst.on,
		Metadata: inst.metadata,
	})
}

// provisionKind resolves a spec's provision kind against its type's
// registration: a written kind word must name a kind the type
// registers, and an omitted word resolves only while the type
// registers exactly one (CP-A). With elem nil the word is checked for
// being a kind at all; registration cannot be consulted, and the
// element fault is already diagnosed.
func (ln *linker) provisionKind(spec *ast.ProvisionSpec, elem *catalogueElement) (string, bool) {
	if spec.Kind == nil {
		if elem == nil {
			return "", false
		}
		switch elem.prov.Kinds {
		case Slice:
			return image.KindSlice, true
		case Attach:
			return image.KindAttach, true
		}
		ln.errorf(spec.Type.Pos(), "missing provision kind: type %s registers both slice and attach", refString(spec.Type))
		return "", false
	}
	word := spec.Kind.Name
	var kind Kinds
	switch word {
	case image.KindSlice:
		kind = Slice
	case image.KindAttach:
		kind = Attach
	default:
		ln.errorf(spec.Kind.NamePos, "unknown provision kind %s: kinds are slice and attach", word)
		return "", false
	}
	if elem == nil {
		return "", false
	}
	if elem.prov.Kinds&kind == 0 {
		ln.errorf(spec.Kind.NamePos, "type %s does not register %s", refString(spec.Type), word)
		return "", false
	}
	return word, true
}

// fold merges the default layers under a statement's own bindings,
// per compartment. The nearest layer wins a key: the instance over
// the element-scoped default over the verb-scoped one, each verb
// folding only its own default; the catalogue's slot defaults are no
// layer at all — a binding exists iff some SDL statement set it, and
// unset slots stay with the pinned schema. Source keeps each
// surviving binding's provenance (image.Equal masks it).
func (ln *linker) fold(elem *catalogueElement, verb string, at *ast.TypeRef, inst compartments) compartments {
	verbDef, typed := ln.verbDefaults[verb], ln.typeDefaults[elem]
	if verbDef == nil && typed == nil {
		return inst
	}
	var verbParams, typedParams []image.Binding
	if verbDef != nil {
		verbParams = ln.defaultFolds(verbDef, elem, at, false)
		if ln.internal != nil {
			return compartments{}
		}
	}
	if typed != nil {
		typedParams = ln.defaultFolds(typed, elem, at, true)
	}
	return compartments{
		params:   mergeCompartment(verbParams, typedParams, inst.params),
		on:       mergeCompartment(verbDef.onLayer(), typed.onLayer(), inst.on),
		metadata: mergeCompartment(verbDef.metadataLayer(), typed.metadataLayer(), inst.metadata),
	}
}

// onLayer and metadataLayer read a default's section compartments,
// tolerating the absent default a fold reaches for.
func (d *defaultBody) onLayer() []image.Binding {
	if d == nil {
		return nil
	}
	return d.on
}

func (d *defaultBody) metadataLayer() []image.Binding {
	if d == nil {
		return nil
	}
	return d.metadata
}

// mergeCompartment folds one compartment's layers, nearest layer
// last: a later layer wins a key, and the result is key-sorted.
func mergeCompartment(layers ...[]image.Binding) []image.Binding {
	merged := make(map[string]image.Binding)
	for _, layer := range layers {
		for _, b := range layer {
			merged[b.Key] = b
		}
	}
	if len(merged) == 0 {
		return nil
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
			continue // sections fold from their own compartments (collectDefault)
		}
		var b image.Binding
		var bound bool
		switch ref, isRef := it.Value.(*ast.RefExpr); {
		case strict:
			b, bound = ln.bindParam(sf, it, d.source)
		case !elem.keys[it.Key.Name]:
			// Not this element's parameter; the default passes it by.
		case isRef && ref.Sel != nil:
			if out := ln.lookupOutput(ref); out != nil {
				b, bound = ln.bindOutput(sf, it.Key, ref, out, d.source)
			}
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

// lookupValueSymbol returns the var or extern symbol a bare reference
// names, or nil quietly: the context-free faults (dangling,
// instance-valued) were diagnosed when the reference was collected.
func (ln *linker) lookupValueSymbol(ref *ast.RefExpr) *symbol {
	sym := ln.symbols[ref.X.Name]
	if sym == nil || sym.class == classInstance {
		return nil
	}
	return sym
}

// lookupOutput returns the provision output a dotted reference names,
// or nil quietly: the context-free faults were diagnosed when the
// reference was collected.
func (ln *linker) lookupOutput(ref *ast.RefExpr) *Output {
	sym := ln.symbols[ref.X.Name]
	if sym == nil || sym.class != classInstance || sym.elem == nil || sym.elem.kind != image.KindProvision {
		return nil
	}
	return sym.elem.outputs[ref.Sel.Name]
}

// lookupElement resolves a type reference quietly: collect uses it to
// seed instance symbols with their elements before any statement is
// checked; the loud diagnosis of a missing link belongs to [resolve],
// at the statement itself.
func (ln *linker) lookupElement(t *ast.TypeRef) *catalogueElement {
	if t.Pkg == nil {
		return nil
	}
	imp, ok := ln.imports[t.Pkg.Name]
	if !ok {
		return nil
	}
	pkg, ok := ln.packages[imp.path]
	if !ok {
		return nil
	}
	return pkg.elements[t.Name.Name]
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
	fs, panicked := elemFlags(elem)
	if panicked != nil {
		ln.internal = &internalError{
			pos: at.Pos(),
			msg: fmt.Sprintf("element %s: %s panicked: %v", refString(at), elem.factory(), panicked),
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

// bind checks a statement body and binds its compartments, each
// key-sorted. Literal parameters validate through a throwaway
// instance's own flag.Value.Set — a fresh dry surface per statement,
// the very one the running instance will parse with, so no state
// leaks between statements or into the pinned schema; the canonical
// bound value is owned by the SDL literal kind, and the flag's own
// String() is never read back. References resolve against the flat
// namespace — bare to var and extern symbols, dotted to provision
// outputs — and are recorded as references: records carry references,
// never inlined values (A-10). Sections bind by their own compartment
// rules, outside the element's schema.
//
// With elem nil (an unresolved reference) the body is still walked —
// references resolve and sections validate — but parameter keys and
// values go unchecked.
func (ln *linker) bind(elem *catalogueElement, at *ast.TypeRef, body *ast.Body) (inst compartments, ok bool) {
	ok = true
	if body == nil {
		return compartments{}, true
	}
	sf := &surface{ln: ln, elem: elem, at: at}
	for _, item := range body.Items {
		it, isParam := item.(*ast.Param)
		if !isParam {
			continue // sections bind below
		}
		b, bound := ln.bindParam(sf, it, image.SourceInstance)
		if ln.internal != nil {
			return compartments{}, false
		}
		if !bound {
			ok = false
			continue
		}
		inst.params = append(inst.params, b)
	}
	slices.SortFunc(inst.params, func(a, b image.Binding) int {
		return strings.Compare(a.Key, b.Key)
	})
	var sectionsOK bool
	inst.on, inst.metadata, sectionsOK = ln.sections(body, image.SourceInstance)
	return inst, ok && sectionsOK
}

// sections validates and binds a body's section compartments with the
// given provenance. Two sections exist, on and metadata, at most once
// each per body; the grammar keeps their names outside the parameter
// namespace, so future sections can never collide with catalogue slot
// names.
func (ln *linker) sections(body *ast.Body, source string) (on, metadata []image.Binding, ok bool) {
	ok = true
	if body == nil {
		return nil, nil, true
	}
	seen := make(map[string]token.Position, 2)
	for _, item := range body.Items {
		sec, isSection := item.(*ast.Section)
		if !isSection {
			continue
		}
		name := sec.Name.Name
		if name != "on" && name != "metadata" {
			ln.errorf(sec.Name.NamePos, "unknown section %s: sections are on and metadata", name)
			ok = false
			continue
		}
		if first, dup := seen[name]; dup {
			ln.errorf(sec.Name.NamePos, "duplicate %s section (first declared at %s)", name, first)
			ok = false
			continue
		}
		seen[name] = sec.Name.NamePos
		bindings, bound := ln.sectionParams(sec, source)
		if !bound {
			ok = false
		}
		if name == "on" {
			on = bindings
		} else {
			metadata = bindings
		}
	}
	return on, metadata, ok
}

// sectionParams validates one section body — parameters only, no
// nesting — and binds its values by the section's rules. Bindings
// come back key-sorted.
func (ln *linker) sectionParams(sec *ast.Section, source string) (bindings []image.Binding, ok bool) {
	ok = true
	for _, item := range sec.Body.Items {
		switch it := item.(type) {
		case *ast.Section:
			ln.errorf(it.Name.NamePos, "%s sections take parameters only, not nested sections", sec.Name.Name)
			ok = false
		case *ast.Param:
			b, bound := ln.sectionParam(sec.Name.Name, it, source)
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

// sectionParam binds one section parameter. metadata values are
// string literals only — the compartment is constrained to portable
// keys and values. on values are literals or opaque profile tokens: a
// bare identifier in an on body names a point in the platform's
// deployment profile (working key set: location), never a solution
// symbol, so the compartment stays outside the flat namespace and the
// binding DAG; checking tokens against a profile is a recorded door.
func (ln *linker) sectionParam(section string, it *ast.Param, source string) (image.Binding, bool) {
	if _, isBad := it.Value.(*ast.BadValue); isBad {
		return image.Binding{}, false // the parse already reported it
	}
	if section == "metadata" {
		lit, isString := it.Value.(*ast.StringLit)
		if !isString {
			ln.errorf(it.Value.Pos(), "metadata values must be string literals")
			return image.Binding{}, false
		}
		return image.Binding{Key: it.Key.Name, Value: image.String(lit.Value), Source: source}, true
	}
	if ref, isRef := it.Value.(*ast.RefExpr); isRef {
		if ref.Sel != nil {
			ln.errorf(ref.Pos(), "on values are literals or profile tokens, not output references")
			return image.Binding{}, false
		}
		return image.Binding{Key: it.Key.Name, Value: image.Token(ref.X.Name), Source: source}, true
	}
	val, _, isLiteral := literalValue(it.Value)
	if !isLiteral {
		// Unreachable past a clean parse: every remaining value node
		// is a literal.
		return image.Binding{}, false
	}
	return image.Binding{Key: it.Key.Name, Value: val, Source: source}, true
}

// bindParam binds one body parameter with the given provenance: a
// literal validates through its slot and lands as a canonical value, a
// reference resolves — bare against the value symbols, dotted against
// the provision output schemes — and lands as a reference. It reports
// whether a binding was produced; a false return has recorded its
// diagnostic (or aborted through ln.internal), except for the
// *ast.BadValue the parse phase already reported.
func (ln *linker) bindParam(sf *surface, it *ast.Param, source string) (image.Binding, bool) {
	if ref, isRef := it.Value.(*ast.RefExpr); isRef {
		if ref.Sel != nil {
			out := ln.resolveOutput(ref)
			if out == nil {
				return image.Binding{}, false
			}
			return ln.bindOutput(sf, it.Key, ref, out, source)
		}
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

// resolveRef resolves a bare value-position reference against the
// flat namespace to a value-carrying symbol; nil (with the fault
// diagnosed) otherwise.
func (ln *linker) resolveRef(ref *ast.RefExpr) *symbol {
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

// resolveOutput resolves a dotted value-position reference a.b to the
// output b in the scheme of provision instance a's type; nil (with the
// fault diagnosed) otherwise. The scheme is registered on the type
// (A-11), so the resolution cannot tell a slice from an attachment —
// by design, since migrating between them must not touch downstream
// wiring.
func (ln *linker) resolveOutput(ref *ast.RefExpr) *Output {
	sym, ok := ln.symbols[ref.X.Name]
	if !ok {
		ln.errorf(ref.Pos(), "undefined symbol %s", ref.X.Name)
		return nil
	}
	if sym.class != classInstance {
		ln.errorf(ref.Pos(), "symbol %s is a %s, not a provision instance", ref.X.Name, sym.class)
		return nil
	}
	if sym.elem == nil {
		// The instance's own statement failed to resolve its element
		// and diagnosed that loudly; the use site has nothing to add.
		return nil
	}
	if sym.elem.kind != image.KindProvision {
		ln.errorf(ref.Pos(), "instance %s has no outputs: only provision instances emit outputs", ref.X.Name)
		return nil
	}
	out, ok := sym.elem.outputs[ref.Sel.Name]
	if !ok {
		ln.errorf(ref.Sel.NamePos, "unknown output %s: provision type %s declares no such output", ref.Sel.Name, sym.elem)
		return nil
	}
	return out
}

// bindOutput binds one resolved provision-output reference. Like an
// extern, an output has no value to validate at compile time — the
// deployment environment resolves it at reconcile time, stage (d) —
// so only the key is checked, the reference alone is recorded, and a
// sensitive output taints the binding (A-10).
func (ln *linker) bindOutput(sf *surface, key *ast.Ident, ref *ast.RefExpr, out *Output, source string) (image.Binding, bool) {
	if sf.elem == nil || !sf.known(key) {
		return image.Binding{}, false
	}
	return image.Binding{
		Key:       key.Name,
		Ref:       &image.SymbolRef{Symbol: ref.X.Name, Output: ref.Sel.Name},
		Source:    source,
		Sensitive: out.Sensitive,
	}, true
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
		fs, panicked := elemFlags(s.elem)
		if panicked != nil {
			s.ln.internal = &internalError{
				pos: s.at.Pos(),
				msg: fmt.Sprintf("element %s: %s panicked: %v", refString(s.at), s.elem.factory(), panicked),
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
			msg: fmt.Sprintf("element %s: %s broke the dry-instantiation invariant: fresh instance lacks parameter %s", refString(s.at), s.elem.factory(), key.Name),
		}
		return nil
	}
	return f
}

// checkCycles rejects reference cycles among the records. The image's
// reference edges form the binding DAG (A-10): a binding referencing
// output a.b makes its record depend on instance a, while var and
// extern symbols have no dependencies of their own — so only provision
// instances can close a cycle, including the length-1 cycle of an
// instance referencing its own output. on and metadata compartments
// stay outside the DAG. Records whose statements failed their checks
// appended nothing and are simply absent; their faults are already
// diagnosed.
func (ln *linker) checkCycles() {
	edges := make(map[string][]string, len(ln.records))
	for _, rec := range ln.records {
		var deps []string
		for _, b := range rec.Params {
			if b.Ref != nil && b.Ref.Output != "" {
				deps = append(deps, b.Ref.Symbol)
			}
		}
		edges[rec.Name] = deps
	}

	// Depth-first walk in record order, so reports are deterministic.
	// One cycle is reported per walk: the first back edge found names
	// the whole loop, positioned at the instance that closes it.
	const unvisited, walking, done = 0, 1, 2
	state := make(map[string]int, len(edges))
	var stack []string
	var visit func(name string) bool
	visit = func(name string) bool {
		state[name] = walking
		stack = append(stack, name)
		for _, dep := range edges[name] {
			switch state[dep] {
			case unvisited:
				if visit(dep) {
					return true
				}
			case walking:
				cycle := append(slices.Clone(stack[slices.Index(stack, dep):]), dep)
				ln.errorf(ln.symbols[dep].pos, "provision reference cycle: %s", strings.Join(cycle, " -> "))
				return true
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = done
		return false
	}
	for _, rec := range ln.records {
		if state[rec.Name] != unvisited {
			continue
		}
		if visit(rec.Name) {
			// The stack holds the cycle and whatever led into it; one
			// report covers it all, so mark it settled and move on.
			for _, name := range stack {
				state[name] = done
			}
			stack = stack[:0]
		}
	}
}

// emit pins the catalogue and assembles the canonical image: every
// registered package — referenced or not, since the registration is
// what this compilation was checked against — sorted by path, elements
// by name, symbols by name, records in unit-then-statement order.
// Pinning a component or provision type not yet dried instantiates it
// here; without a referencing statement a panic is reported
// positionless.
func (ln *linker) emit() (*image.Image, *internalError) {
	catalogue := make([]image.Package, 0, len(ln.packages))
	for _, path := range slices.Sorted(maps.Keys(ln.packages)) {
		pkg := ln.packages[path]
		elements := make([]image.ElementSchema, 0, len(pkg.elements))
		for _, name := range slices.Sorted(maps.Keys(pkg.elements)) {
			elem := pkg.elements[name]
			es := image.ElementSchema{Name: elem.name, Kind: elem.kind}
			switch elem.kind {
			case image.KindComponent, image.KindProvision:
				if !elem.dried {
					fs, panicked := elemFlags(elem)
					if panicked != nil {
						return nil, &internalError{
							msg: fmt.Sprintf("element %s: %s panicked: %v", elem, elem.factory(), panicked),
						}
					}
					elem.schema = paramSchemas(fs)
					elem.dried = true
				}
				es.Params = elem.schema
				if elem.kind == image.KindComponent {
					es.Doc = elem.desc.Doc
				} else {
					es.Doc = elem.prov.Doc
					es.Outputs = outputSchemas(elem.prov.Outputs)
					es.Kinds = kindStrings(elem.prov.Kinds)
				}
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

// elemFlags performs one recover-guarded dry instantiation of an
// element's parameter surface. A component Makes a fresh service and
// takes its flag surface, discarding the runner; a provision type
// declares its Params on a fresh set, uniform with the descriptor
// path. A nil flag set is a parameterless element. panicked carries
// any panic out of the user code involved — the factory itself, or a
// Make that returned no service.
func elemFlags(elem *catalogueElement) (fs *flag.FlagSet, panicked any) {
	defer func() {
		if p := recover(); p != nil {
			fs, panicked = nil, p
		}
	}()
	if elem.kind == image.KindProvision {
		if elem.prov.Params == nil {
			return nil, nil
		}
		fs = flag.NewFlagSet(elem.name, flag.ContinueOnError)
		elem.prov.Params(fs)
		return fs, nil
	}
	return elem.desc.Make().Flags(), nil
}

// outputSchemas pins a provision type's output scheme sorted by name,
// the canonical order shared with parameter schemas; registration
// order carries no meaning the image would need to keep.
func outputSchemas(outputs []Output) []image.OutputSchema {
	if len(outputs) == 0 {
		return nil
	}
	schemas := make([]image.OutputSchema, 0, len(outputs))
	for _, out := range outputs {
		schemas = append(schemas, image.OutputSchema{Name: out.Name, Sensitive: out.Sensitive})
	}
	slices.SortFunc(schemas, func(a, b image.OutputSchema) int {
		return strings.Compare(a.Name, b.Name)
	})
	return schemas
}

// kindStrings renders a registered kind set in the image's canonical
// order, slice before attach.
func kindStrings(k Kinds) []string {
	var kinds []string
	if k&Slice != 0 {
		kinds = append(kinds, image.KindSlice)
	}
	if k&Attach != 0 {
		kinds = append(kinds, image.KindAttach)
	}
	return kinds
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
