// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The composer's half of the package: what a program building [Image]
// values by struct literal calls before encoding, so hand-composed
// images meet the contracts compiled images meet by construction.

package image

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Canonicalize sorts the image in place into the canonical element
// orders the package comment prescribes and the compiler emits by
// construction: catalogue packages by path, elements by name,
// parameter and output schemas by name, provision kinds slice before
// attach, symbols by name, build settings by key, and every binding
// compartment — params, deployment, each extension stanza, metadata —
// by key. The catalogue, symbol, and record sections themselves are
// lifted from nil to empty, the spelling compiled images carry.
//
// Records are deliberately not sorted: their order is statement order,
// meaning the image must keep, and a composer's record order is its
// statement order. Unit digests keep that same order.
//
// Canonicalize is idempotent, and its sorts are stable, so even an
// image [Image.Validate] would refuse (a duplicate binding key, say)
// canonicalizes deterministically.
func (img *Image) Canonicalize() {
	if img.Catalogue == nil {
		img.Catalogue = []Package{}
	}
	if img.Symbols == nil {
		img.Symbols = []SymbolDef{}
	}
	if img.Records == nil {
		img.Records = []Record{}
	}

	slices.SortStableFunc(img.Catalogue, func(a, b Package) int { return strings.Compare(a.Path, b.Path) })
	for i := range img.Catalogue {
		pkg := &img.Catalogue[i]
		slices.SortStableFunc(pkg.Elements, func(a, b ElementSchema) int { return strings.Compare(a.Name, b.Name) })
		for j := range pkg.Elements {
			el := &pkg.Elements[j]
			slices.SortStableFunc(el.Params, func(a, b ParamSchema) int { return strings.Compare(a.Name, b.Name) })
			slices.SortStableFunc(el.Outputs, func(a, b OutputSchema) int { return strings.Compare(a.Name, b.Name) })
			slices.SortStableFunc(el.Kinds, func(a, b string) int { return cmp.Compare(kindRank(a), kindRank(b)) })
		}
	}

	slices.SortStableFunc(img.Symbols, func(a, b SymbolDef) int { return strings.Compare(a.Name, b.Name) })
	if img.Build != nil {
		slices.SortStableFunc(img.Build.Settings, func(a, b Setting) int { return strings.Compare(a.Key, b.Key) })
	}

	for i := range img.Records {
		rec := &img.Records[i]
		sortBindings(rec.Params)
		sortBindings(rec.Deployment)
		for _, stanza := range rec.Extensions {
			sortBindings(stanza)
		}
		sortBindings(rec.Metadata)
	}
}

// sortBindings orders one compartment canonically, by key. Extension
// stanzas need no map-level companion: they encode through
// encoding/json, which emits map keys sorted.
func sortBindings(bindings []Binding) {
	slices.SortStableFunc(bindings, func(a, b Binding) int { return strings.Compare(a.Key, b.Key) })
}

// kindRank orders a provision kind list canonically: slice before
// attach, the order compilation pins registered kinds in ([ElementSchema].Kinds),
// with anything unknown after, in its given order.
func kindRank(kind string) int {
	switch kind {
	case KindSlice:
		return 0
	case KindAttach:
		return 1
	}
	return 2
}

// Validate judges the image's self-consistency, dry: the header is
// well-formed ([Format], a solution name, a positive generation); the
// kind, verb, class, and source vocabularies hold; names are unique
// where the schema keys by them (packages by path, elements and their
// schemas per package, scheme qualifiers across the catalogue, symbols,
// instances, binding keys per compartment); every binding carries
// exactly one payload arm — a literal of a known kind, or a reference
// the image itself resolves: a symbol reference into the symbol table,
// an output reference onto a provision record whose pinned element
// declares that output. Extern symbols declare a type the image pins
// as a symbol type and vars pin a value; opaque tokens stay in the
// deployment and extension compartments; metadata stays strings; and
// the provision-output references close without a cycle. Faults report
// all at once, one line each, joined; nil means the image is a
// consistent document.
//
// Validation is deliberately image-internal. Whether the pinned
// catalogue tells the truth — elements a linked binary registers,
// parameters their flag surfaces accept, provision kinds their types
// still carry — is semantic validation against a live catalogue, and
// stays the solution/enact package's job at the moment of enactment.
// Canonical order is [Image.Canonicalize]'s business, not judged here;
// the [Build] block is provenance, not judged at all; and names the
// SDL notation could not spell are fine here — the text frontend's
// limits, which sdl echo enforces when rendering, are not the
// schema's.
func (img *Image) Validate() error {
	if img == nil {
		return errors.New("image: nil image")
	}
	v := &validator{
		img:        img,
		pinned:     make(map[Ref]ElementSchema),
		names:      make(map[string]string, len(img.Catalogue)),
		symbols:    make(map[string]bool, len(img.Symbols)),
		provisions: make(map[string]Ref),
	}
	v.header()
	v.catalogue()
	v.symbolTable()
	v.records()
	v.cycles()
	return errors.Join(v.faults...)
}

// A validator is one Validate in progress: the image, the indexes its
// checks resolve through, and the faults found so far.
type validator struct {
	img *Image

	// pinned indexes the image's catalogue section by element ref;
	// names maps package paths to package names for rendering refs in
	// faults.
	pinned map[Ref]ElementSchema
	names  map[string]string

	// symbols holds the symbol table's names; provisions maps each
	// provision record's instance name to its element ref, so output
	// references resolve against the pinned output scheme.
	symbols    map[string]bool
	provisions map[string]Ref

	faults []error
}

// faultf records one fault line.
func (v *validator) faultf(format string, args ...any) {
	v.faults = append(v.faults, fmt.Errorf("image: "+format, args...))
}

// display renders an element ref the way faults speak about elements:
// package name, dot, element name, falling back to the quoted import
// path for a package the image does not pin (or pins nameless).
func (v *validator) display(ref Ref) string {
	if name, ok := v.names[ref.Package]; ok && name != "" {
		return name + "." + ref.Name
	}
	return strconv.Quote(ref.Package) + "." + ref.Name
}

func (v *validator) header() {
	if v.img.Format != Format {
		v.faultf("format %q is not %q", v.img.Format, Format)
	}
	if v.img.Solution == "" {
		v.faultf("no solution name")
	}
	if v.img.Generation < 1 {
		v.faultf("generation %d is not positive", v.img.Generation)
	}
}

// catalogue validates the pinned catalogue section and builds the
// pinned index the symbol and record checks resolve through. On
// duplicate pins the first wins, mirroring how the duplicate is the
// fault, not everything after it.
func (v *validator) catalogue() {
	qualifiers := make(map[string]Ref)
	for _, pkg := range v.img.Catalogue {
		if pkg.Path == "" {
			v.faultf("catalogue pins a package with an empty path")
		}
		if pkg.Name == "" {
			v.faultf("package %q has no name", pkg.Path)
		}
		if _, ok := v.names[pkg.Path]; ok {
			v.faultf("catalogue pins package %q twice", pkg.Path)
			continue
		}
		v.names[pkg.Path] = pkg.Name

		for _, es := range pkg.Elements {
			ref := Ref{Package: pkg.Path, Name: es.Name}
			if es.Name == "" {
				v.faultf("package %q pins an element with an empty name", pkg.Path)
			}
			if _, ok := v.pinned[ref]; ok {
				v.faultf("package %q pins element %s twice", pkg.Path, es.Name)
				continue
			}
			v.pinned[ref] = es
			v.element(ref, es, qualifiers)
		}
	}
}

// element validates one pinned element schema.
func (v *validator) element(ref Ref, es ElementSchema, qualifiers map[string]Ref) {
	elem := v.display(ref)
	switch es.Kind {
	case KindComponent, KindProvision, KindSymbol, KindScheme:
	default:
		v.faultf("element %s: unknown kind %q", elem, es.Kind)
	}
	switch {
	case es.Kind == KindScheme && es.Qualifier == "":
		v.faultf("scheme element %s pins no qualifier", elem)
	case es.Kind != KindScheme && es.Qualifier != "":
		v.faultf("element %s pins qualifier %q but is a %s, not a scheme", elem, es.Qualifier, es.Kind)
	case es.Qualifier != "":
		if first, ok := qualifiers[es.Qualifier]; ok {
			v.faultf("scheme qualifier %q pinned by both %s and %s", es.Qualifier, v.display(first), elem)
		} else {
			qualifiers[es.Qualifier] = ref
		}
	}

	params := make(map[string]bool, len(es.Params))
	for _, p := range es.Params {
		if params[p.Name] {
			v.faultf("element %s pins parameter %s twice", elem, p.Name)
		}
		params[p.Name] = true
	}
	outputs := make(map[string]bool, len(es.Outputs))
	for _, out := range es.Outputs {
		if outputs[out.Name] {
			v.faultf("element %s pins output %s twice", elem, out.Name)
		}
		outputs[out.Name] = true
	}
	for _, kind := range es.Kinds {
		if kind != KindSlice && kind != KindAttach {
			v.faultf("element %s registers unknown provision kind %q", elem, kind)
		}
	}
}

// symbolTable validates the symbol table: unique names, known classes,
// and the class-specific payloads — an extern declares a pinned symbol
// type and no value, a var pins a literal value and no type.
func (v *validator) symbolTable() {
	for _, def := range v.img.Symbols {
		if def.Name == "" {
			v.faultf("a symbol declares no name")
		}
		if v.symbols[def.Name] {
			v.faultf("symbol %s declared twice", def.Name)
			continue
		}
		v.symbols[def.Name] = true

		switch def.Class {
		case ClassExtern:
			if def.Value != nil {
				v.faultf("extern %s pins a value, which only vars carry", def.Name)
			}
			if def.Type == nil {
				v.faultf("extern %s declares no type", def.Name)
				continue
			}
			es, ok := v.pinned[*def.Type]
			if !ok {
				v.faultf("extern %s declares type %s but the image pins no such element", def.Name, v.display(*def.Type))
			} else if es.Kind != KindSymbol {
				v.faultf("extern %s declares type %s but the image pins a %s, not a symbol type", def.Name, v.display(*def.Type), es.Kind)
			}
		case ClassVar:
			if def.Type != nil {
				v.faultf("var %s declares a type, which only externs carry", def.Name)
			}
			if def.Value == nil {
				v.faultf("var %s pins no value", def.Name)
				continue
			}
			v.literal(fmt.Sprintf("var %s", def.Name), def.Value, false)
		default:
			v.faultf("symbol %s: unknown class %q", def.Name, def.Class)
		}
	}
}

// records validates every record. The provision index fills in a
// pre-pass so output references resolve regardless of record order —
// the order carries meaning, not scope.
func (v *validator) records() {
	for _, rec := range v.img.Records {
		if rec.Verb == VerbProvision && rec.Name != "" {
			if _, ok := v.provisions[rec.Name]; !ok {
				v.provisions[rec.Name] = rec.Element
			}
		}
	}

	names := make(map[string]bool, len(v.img.Records))
	for i, rec := range v.img.Records {
		if rec.Name == "" {
			v.faultf("record %d declares no instance name", i)
		} else if names[rec.Name] {
			v.faultf("instance %s declared twice", rec.Name)
		}
		names[rec.Name] = true
		v.record(rec)
	}
}

// record validates one record: the verb and provision-kind
// vocabularies, the element resolving to a pin whose kind matches the
// verb, and the four binding compartments.
func (v *validator) record(rec Record) {
	switch rec.Verb {
	case VerbDeploy:
		if rec.Kind != "" {
			v.faultf("deploy record %s carries provision kind %q", rec.Name, rec.Kind)
		}
		if es, ok := v.resolve(rec); ok && es.Kind != KindComponent {
			v.faultf("record %s deploys %s, which the image pins as a %s, not a component", rec.Name, v.display(rec.Element), es.Kind)
		}
	case VerbProvision:
		if rec.Kind != KindSlice && rec.Kind != KindAttach {
			v.faultf("record %s declares unknown provision kind %q", rec.Name, rec.Kind)
		}
		if es, ok := v.resolve(rec); ok && es.Kind != KindProvision {
			v.faultf("record %s provisions %s, which the image pins as a %s, not a provision type", rec.Name, v.display(rec.Element), es.Kind)
		}
	default:
		v.faultf("record %s declares unknown verb %q", rec.Name, rec.Verb)
	}

	v.params(rec)
	v.opaque(rec.Name, "deployment", "top-level field", "top-level", rec.Deployment, false)
	for _, q := range slices.Sorted(maps.Keys(rec.Extensions)) {
		if q == "" {
			v.faultf("record %s declares a with-stanza with no qualifier", rec.Name)
		}
		v.opaque(rec.Name, "with "+q, "with "+q+" parameter", "with-stanza", rec.Extensions[q], false)
	}
	v.opaque(rec.Name, "metadata", "metadata parameter", "metadata", rec.Metadata, true)
}

// resolve looks a record's element up in the pinned catalogue,
// faulting when the pin is missing.
func (v *validator) resolve(rec Record) (ElementSchema, bool) {
	es, ok := v.pinned[rec.Element]
	if !ok {
		v.faultf("record %s references %s but the image pins no such element", rec.Name, v.display(rec.Element))
	}
	return es, ok
}

// params validates the params compartment: unique keys, known sources,
// exactly one payload arm per binding, literal kinds known and never
// opaque, references resolving image-internally.
func (v *validator) params(rec Record) {
	keys := make(map[string]bool, len(rec.Params))
	for _, b := range rec.Params {
		if keys[b.Key] {
			v.faultf("record %s binds %s twice in params", rec.Name, b.Key)
		}
		keys[b.Key] = true
		noun := fmt.Sprintf("record %s: binding %s", rec.Name, b.Key)
		v.source(noun, b.Source)
		switch {
		case b.Value != nil && b.Ref != nil:
			v.faultf("%s carries both a value and a reference", noun)
		case b.Value == nil && b.Ref == nil:
			v.faultf("%s carries neither value nor reference", noun)
		case b.Value != nil:
			v.literal(noun, b.Value, false)
		default:
			v.ref(noun, b.Ref)
		}
	}
}

// opaque validates one opaque compartment — deployment, an extension
// stanza, or metadata: unique keys, known sources, literal values only
// (these compartments never resolve symbols), tokens welcome except
// where stringOnly says metadata's portable strings rule.
func (v *validator) opaque(recName, compartment, noun, what string, bindings []Binding, stringOnly bool) {
	keys := make(map[string]bool, len(bindings))
	for _, b := range bindings {
		if keys[b.Key] {
			v.faultf("record %s binds %s twice in %s", recName, b.Key, compartment)
		}
		keys[b.Key] = true
		bound := fmt.Sprintf("record %s: %s %s", recName, noun, b.Key)
		v.source(bound, b.Source)
		if b.Ref != nil {
			v.faultf("%s carries a symbol reference; %s values never resolve symbols", bound, what)
			continue
		}
		if b.Value == nil {
			v.faultf("%s carries no value", bound)
			continue
		}
		if !v.literal(bound, b.Value, true) {
			continue
		}
		if stringOnly && b.Value.Kind != KindString {
			v.faultf("%s: metadata values are strings, not %q", bound, b.Value.Kind)
		}
	}
}

// literal judges one literal payload: the kind must be known, and an
// opaque token must be welcome — tokens belong to the deployment and
// extension compartments, never to values the solution's namespace
// resolves. Reports whether the kind was known at all.
func (v *validator) literal(noun string, val *Value, tokensOK bool) bool {
	switch val.Kind {
	case KindString, KindInt, KindBool, KindDuration:
		return true
	case KindToken:
		if !tokensOK {
			v.faultf("%s pins opaque token %q; tokens belong to the deployment and extension compartments", noun, val.Tok)
		}
		return true
	}
	v.faultf("%s: unknown value kind %q", noun, val.Kind)
	return false
}

// source holds one binding's provenance to the source vocabulary.
func (v *validator) source(noun, source string) {
	switch source {
	case SourceInstance, SourceDefaultType, SourceDefaultDeploy, SourceDefaultProvision:
	default:
		v.faultf("%s records unknown source %q", noun, source)
	}
}

// ref closes one reference over the image: a symbol reference must
// land in the symbol table, an output reference on a provision record
// whose pinned element declares the output. A record whose element is
// not pinned at all reports through its own resolution fault, not
// again through every reference at it.
func (v *validator) ref(noun string, ref *SymbolRef) {
	if ref.Symbol == "" {
		v.faultf("%s references no symbol", noun)
		return
	}
	if ref.Output == "" {
		if !v.symbols[ref.Symbol] {
			v.faultf("%s references undeclared symbol %s", noun, ref.Symbol)
		}
		return
	}
	target, ok := v.provisions[ref.Symbol]
	if !ok {
		v.faultf("%s references output %s.%s but the image provisions no instance %s", noun, ref.Symbol, ref.Output, ref.Symbol)
		return
	}
	es, ok := v.pinned[target]
	if !ok {
		return
	}
	for _, out := range es.Outputs {
		if out.Name == ref.Output {
			return
		}
	}
	v.faultf("%s references output %s.%s but %s pins no such output", noun, ref.Symbol, ref.Output, v.display(target))
}

// cycles rejects reference cycles among the provision-output edges,
// mirroring the linker's depth-first walk (solution's checkCycles):
// a binding referencing output a.b makes its record depend on instance
// a, only provision instances can close a loop — the length-1 loop of
// an instance referencing its own output included — and one report
// names each cycle found, in record order.
func (v *validator) cycles() {
	edges := make(map[string][]string, len(v.img.Records))
	for _, rec := range v.img.Records {
		var deps []string
		for _, b := range rec.Params {
			if b.Ref != nil && b.Ref.Output != "" {
				deps = append(deps, b.Ref.Symbol)
			}
		}
		edges[rec.Name] = deps
	}

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
				v.faultf("provision reference cycle: %s", strings.Join(cycle, " -> "))
				return true
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = done
		return false
	}
	for _, rec := range v.img.Records {
		if state[rec.Name] != unvisited {
			continue
		}
		if visit(rec.Name) {
			for _, name := range stack {
				state[name] = done
			}
			stack = stack[:0]
		}
	}
}
