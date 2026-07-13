// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package image defines the desired-state image: the intermediate
// representation a compiled solution emits and deployment environments
// reconcile (the IL of D-08). An image is a JSON document that pins the
// catalogue schemas the solution was compiled against and carries one
// record per deployment statement; consumers never see the solution's
// source text.
//
// # Canonical form
//
// One desired state has exactly one byte representation. [Image.Encode]
// writes two-space-indented JSON with object fields in struct declaration
// order and a trailing newline; producers supply the element orders the
// schema prescribes: catalogue packages sorted by path, elements by name,
// parameter schemas in flag.FlagSet.VisitAll order (lexicographic),
// symbols sorted by name, records in unit-then-statement order, and
// bindings by key (extension stanzas are a map, so encoding/json emits
// their qualifiers sorted). The canonical order is what lets [Equal]
// compare structurally and keeps images diffable.
//
// # Provenance
//
// Generation, the [Build] block, and every [Binding]'s Source describe
// how the image came to be, not what state it desires. [Equal]
// therefore masks all three: two compilations of the same solution are
// equal even when produced at different generations, from differently
// spelled sources, or with values arriving through different default
// layers.
//
// The type set is deliberately minimal: this rung emits components,
// provision types, symbol types, deploy and provision records, and the
// symbol table their bindings reference; later rungs extend the schema
// by adding fields (JSON forward compatibility by addition), never by
// reshaping the ones below.
package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Format identifies the image schema this package reads and writes.
// [Decode] rejects documents declaring any other format. At the
// prototype stage compartment reshapes ride within the one format —
// images regenerate with the tool that reads them — and the version
// bumps once an image consumer outlives its producer.
const Format = "solution-image/1"

// The element kinds a catalogue schema pins.
const (
	// KindComponent marks an element schema pinned from an application
	// descriptor.
	KindComponent = "component"

	// KindProvision marks an element schema pinned from a provision
	// type: substrate access an instance holds, sliced or attached.
	KindProvision = "provision"

	// KindSymbol marks an element schema pinned from a symbol type:
	// the class of late-bound values an extern symbol declares.
	KindSymbol = "symbol"
)

// The record verbs, one per statement class.
const (
	// VerbDeploy marks a record produced by a deploy statement.
	VerbDeploy = "deploy"

	// VerbProvision marks a record produced by a provision statement.
	VerbProvision = "provision"
)

// The provision kinds a record may carry: a slice owns the partition it
// carves, an attachment verifies substrate it never owns (A-11).
const (
	KindSlice  = "slice"
	KindAttach = "attach"
)

// The symbol classes of [SymbolDef], mirroring linkage: a var is bound
// at compile time and a site may rebind it; an extern is unbound in
// the image and the site must bind it.
const (
	ClassVar    = "var"
	ClassExtern = "extern"
)

// The binding sources, recording which layer bound a parameter: the
// instance's own statement, or one of the default layers folded under
// it (the merge order is catalogue slot default, then the verb default
// — default-deploy or default-provision, matching the record's verb —
// then default-type, then instance; the catalogue layer lives in the
// pinned schema and emits no binding). Source is provenance: [Equal]
// masks it.
const (
	SourceInstance         = "instance"
	SourceDefaultType      = "default-type"
	SourceDefaultDeploy    = "default-deploy"
	SourceDefaultProvision = "default-provision"
)

// An Image is one solution's desired state: the identity header, the
// pinned catalogue schemas the solution was compiled against, and the
// records deployment environments reconcile.
type Image struct {
	// Format names the image schema; always [Format].
	Format string `json:"format"`

	// Solution is the solution name shared by every compiled unit.
	Solution string `json:"solution"`

	// Generation is the producer-supplied monotonic generation of this
	// desired state; the producer's -generation flag stamps it, and an
	// unset generation defaults to 1. The deploying pipeline owns the
	// counter: it bumps the generation whenever it ships changed
	// desired content, so consumers meeting two images that carry the
	// same generation but different content treat that as the
	// pipeline's error, never as a difference to reconcile. It is
	// provenance: [Equal] masks it.
	Generation int64 `json:"generation"`

	// Build is the image's governance block, absent from images that
	// predate it. It is provenance in whole: [Equal] masks it.
	Build *Build `json:"build,omitempty"`

	// Catalogue pins every registered package, sorted by Path.
	Catalogue []Package `json:"catalogue"`

	// Symbols is the solution's symbol table, sorted by Name. Record
	// bindings reference into it; instance names, though they share
	// the solution's namespace, are carried by the records themselves.
	Symbols []SymbolDef `json:"symbols"`

	// Records hold the deployment statements in unit-then-statement
	// order.
	Records []Record `json:"records"`
}

// A Build is the image's governance block: the provenance of the
// compilation that produced it, in the mold of debug.BuildInfo. Its
// contents serve skew detection, audit, and blast-radius tracing —
// never reconciliation: nothing here is desired state, so [Equal]
// masks the block in whole. Semantic linkage facts (extern must-bind,
// var overridability, sensitivity) belong to the symbol table, not
// here.
//
// The block deliberately carries no wall-clock stamp: two
// compilations of the same inputs stay byte-identical. A vcs-style
// stamp and guardrail fields (delete protection) reopen here when
// enactment needs them.
type Build struct {
	// Units digest every compiled unit, in unit order — the order
	// records keep.
	Units []UnitDigest `json:"units"`

	// Settings record tool-chain facts of the producing build, sorted
	// by key. Current keys: "sdl.version", the producing CLI's module
	// version, and "prototype.version", the framework library the
	// compiler linked against. A setting appears only when a real
	// module version resolved; locally sourced builds record nothing,
	// keeping images machine-independent.
	Settings []Setting `json:"settings,omitempty"`
}

// A UnitDigest is one compiled unit's content digest.
type UnitDigest struct {
	// Name is the unit's filename, as diagnostics position it.
	Name string `json:"name"`

	// SHA256 is the lowercase-hex SHA-256 of the unit's exact source
	// text, so `shasum -a 256 <unit>` verifies a source file against
	// the image it produced.
	SHA256 string `json:"sha256"`
}

// A Setting is one key-value fact of the governance block.
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// A Package pins the compiled-against schema of one catalogue package.
type Package struct {
	// Path is the package's Go import path, the stable half of every
	// element reference.
	Path string `json:"path"`

	// Name is the package's Go package name. It is not derivable from
	// Path (major-version suffixes, hyphenated repositories), so the
	// image carries it for consumers that render references.
	Name string `json:"name"`

	// Elements are the package's registered elements, sorted by Name.
	Elements []ElementSchema `json:"elements"`
}

// An ElementSchema pins one element's parameter surface as it was at
// compile time.
type ElementSchema struct {
	// Name is the element's exported identifier within its package.
	Name string `json:"name"`

	// Kind classifies the element: [KindComponent], [KindProvision],
	// or [KindSymbol].
	Kind string `json:"kind"`

	// Doc is the element's documentation, copied from its descriptor.
	Doc string `json:"doc,omitempty"`

	// Params describe the element's declared parameters in
	// flag.FlagSet.VisitAll order (lexicographic). Empty for flagless
	// elements; symbol types have none.
	Params []ParamSchema `json:"params,omitempty"`

	// Outputs is a provision type's output scheme, sorted by Name.
	// The scheme belongs to the type, not the kind: slices and
	// attachments of one type emit the same outputs.
	Outputs []OutputSchema `json:"outputs,omitempty"`

	// Kinds are the provision kinds the type registers: a non-empty
	// subset of [KindSlice] and [KindAttach], in that order.
	Kinds []string `json:"kinds,omitempty"`

	// Sensitive marks a symbol type whose values must not be logged
	// or exposed. Bindings referencing an extern of this type carry
	// the taint.
	Sensitive bool `json:"sensitive,omitempty"`
}

// An OutputSchema describes one reconcile-time output a provision type
// declares.
type OutputSchema struct {
	// Name is the output's name; record bindings reference it through
	// [SymbolRef].
	Name string `json:"name"`

	// Type is the output's declared scalar type: string, int, bool,
	// or duration. Empty means string — the permissive default, and
	// the spelling under which images predating typed outputs decode.
	// The binding site validates the rendered value against its own
	// slot; the type is what static checks and typed transports hold
	// the value to before then.
	Type string `json:"type,omitempty"`

	// Sensitive marks outputs that must not be logged or exposed.
	// Bindings referencing the output carry the taint.
	Sensitive bool `json:"sensitive,omitempty"`
}

// A ParamSchema describes one parameter an element declares.
type ParamSchema struct {
	// Name is the flag name.
	Name string `json:"name"`

	// Usage is the flag's usage string.
	Usage string `json:"usage,omitempty"`

	// Default is the flag's DefValue: the default as the flag renders
	// it. It is a schema fact pinned verbatim; an empty string for
	// flag.Func-based flags is honest.
	Default string `json:"default,omitempty"`

	// Boolean marks flags that may be set without a value
	// (flag.Value's IsBoolFlag contract).
	Boolean bool `json:"boolean,omitempty"`
}

// A SymbolDef is one row of the image's symbol table: a var with its
// compile-bound literal default, or an extern the deploying site must
// bind. Records reference symbols by name through [SymbolRef]; the
// table is the one place a site override rebinds, reaching every use
// site uniformly (A-10).
type SymbolDef struct {
	// Name is the symbol's solution-wide name.
	Name string `json:"name"`

	// Class is the symbol's linkage class, [ClassVar] or [ClassExtern].
	Class string `json:"class"`

	// Type references the symbol-type element an extern declares,
	// resolving into the catalogue section; nil for vars.
	Type *Ref `json:"type,omitempty"`

	// Value is a var's compile-bound literal default; nil for externs,
	// whose values exist only once a site binds them.
	Value *Value `json:"value,omitempty"`
}

// A Record is one reconciliation unit: the effective desired state of
// one deployment statement.
type Record struct {
	// Verb names the statement class, [VerbDeploy] or [VerbProvision].
	Verb string `json:"verb"`

	// Kind is a provision record's kind, [KindSlice] or [KindAttach];
	// empty for deploy records. The compiler resolves an omitted kind
	// word, so the image always carries it explicitly.
	Kind string `json:"kind,omitempty"`

	// Element resolves into the image's catalogue section.
	Element Ref `json:"element"`

	// Name is the instance name, the reconciliation key.
	Name string `json:"name"`

	// Params are the bound parameters, sorted by Key: the compartment
	// the application consumes, typed by the pinned schema.
	Params []Binding `json:"params,omitempty"`

	// Deployment is the statement's deployment-intent compartment,
	// sorted by Key: the top-level fields, a closed per-verb scheme
	// (deploy: location; provision: none yet). Values are literals or
	// opaque profile tokens ([KindToken]); the compartment is typed by
	// the platform's profile, outside the solution's namespace and the
	// binding DAG. Provision records carry it empty while their scheme
	// is empty; if provision ever grows top-level fields that are not
	// deployment intent, renaming the compartment is the door to
	// reopen.
	Deployment []Binding `json:"deployment,omitempty"`

	// Extensions are the statement's advisory with-stanzas, keyed by
	// dotted qualifier, each stanza sorted by Key. A controller that
	// recognizes a qualifier applies its stanza; one that does not
	// ignores it, and unknown stanzas ride the image opaquely
	// (discovered stanza schemes are a later rung). Values are
	// literals or opaque tokens, outside the solution's namespace and
	// the binding DAG; an empty stanza still names its scheme.
	Extensions map[string][]Binding `json:"extensions,omitempty"`

	// Metadata is the statement's carried-through compartment nobody
	// interprets, sorted by Key; values are strings.
	Metadata []Binding `json:"metadata,omitempty"`
}

// A Ref addresses an element pinned in the image's catalogue section.
type Ref struct {
	// Package is the defining package's import path.
	Package string `json:"package"`

	// Name is the element's name within that package.
	Name string `json:"name"`
}

// A Binding is one bound parameter: the key, the payload — a canonical
// literal value or a symbol reference, exactly one of the two — and
// the provenance of the binding.
type Binding struct {
	// Key is the parameter name.
	Key string `json:"key"`

	// Value is the canonical bound value of a literal binding; nil
	// when Ref is set.
	Value *Value `json:"value,omitempty"`

	// Ref is the symbol reference of a reference binding: records
	// carry references, never inlined symbol values, so rebinding one
	// symbol reaches every use site uniformly (A-10).
	Ref *SymbolRef `json:"ref,omitempty"`

	// Source records which layer bound the value. It is provenance:
	// [Equal] masks it.
	Source string `json:"source"`

	// Sensitive marks a binding tainted by a sensitive symbol type:
	// consumers must not log or expose the value it resolves to.
	Sensitive bool `json:"sensitive,omitempty"`
}

// A SymbolRef is a record binding's reference to a late-bound value:
// a row of the image's symbol table, or — with Output set — a
// provision instance's reconcile-time output.
type SymbolRef struct {
	// Symbol is the referenced symbol's name: a var or extern of the
	// symbol table or, when Output is set, a provision record's
	// instance name.
	Symbol string `json:"symbol"`

	// Output names the referenced output in the instance's provision
	// type scheme; empty for symbol-table references.
	Output string `json:"output,omitempty"`
}

// Encode writes the image in its canonical byte form: two-space-indented
// JSON with object fields in struct declaration order (json.MarshalIndent
// preserves it) and a trailing newline. Encoding the same image twice
// yields identical bytes; element ordering within slices is the
// producer's contract, not enforced here.
func (img *Image) Encode(w io.Writer) error {
	data, err := json.MarshalIndent(img, "", "  ")
	if err != nil {
		return fmt.Errorf("image: encode: %w", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("image: encode: %w", err)
	}
	return nil
}

// Decode reads one image from r, validating that the document declares
// the [Format] this package understands and that nothing but
// whitespace follows it: an image is one JSON document, so trailing
// data means the input is not an image at all — a concatenation, a
// corrupted rewrite — and quietly ignoring the rest would validate the
// wrong bytes.
func Decode(r io.Reader) (*Image, error) {
	dec := json.NewDecoder(r)
	var img Image
	if err := dec.Decode(&img); err != nil {
		return nil, fmt.Errorf("image: decode: %w", err)
	}
	if img.Format != Format {
		return nil, fmt.Errorf("image: decode: format %q is not %q", img.Format, Format)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("image: decode: trailing data after the image document")
	}
	return &img, nil
}
