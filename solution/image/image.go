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
// records in unit-then-statement order, and bindings by key. The
// canonical order is what lets [Equal] compare structurally and keeps
// images diffable.
//
// # Provenance
//
// Generation and every [Binding]'s Source describe how the image came to
// be, not what state it desires. [Equal] therefore masks both: two
// compilations of the same solution are equal even when produced at
// different generations or, later, with values arriving through
// different default layers.
//
// The type set is deliberately minimal: this rung emits components and
// deploy records only, and later rungs extend the schema by adding
// fields (JSON forward compatibility by addition), never by reshaping
// the ones below.
package image

import (
	"encoding/json"
	"fmt"
	"io"
)

// Format identifies the image schema this package reads and writes.
// [Decode] rejects documents declaring any other format.
const Format = "solution-image/1"

// KindComponent marks an element schema pinned from an application
// descriptor. It is the only element kind this rung emits.
const KindComponent = "component"

// VerbDeploy marks a record produced by a deploy statement. It is the
// only verb this rung emits.
const VerbDeploy = "deploy"

// SourceInstance records that the instance's own statement bound the
// parameter. It is the only binding source this rung emits; later rungs
// add the default layers.
const SourceInstance = "instance"

// An Image is one solution's desired state: the identity header, the
// pinned catalogue schemas the solution was compiled against, and the
// records deployment environments reconcile.
type Image struct {
	// Format names the image schema; always [Format].
	Format string `json:"format"`

	// Solution is the solution name shared by every compiled unit.
	Solution string `json:"solution"`

	// Generation is the producer-supplied monotonic generation of this
	// desired state. It is provenance: [Equal] masks it.
	Generation int64 `json:"generation"`

	// Catalogue pins every registered package, sorted by Path.
	Catalogue []Package `json:"catalogue"`

	// Records hold the deployment statements in unit-then-statement
	// order.
	Records []Record `json:"records"`
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

	// Kind classifies the element; always [KindComponent] in this rung.
	Kind string `json:"kind"`

	// Doc is the element's documentation, copied from its descriptor.
	Doc string `json:"doc,omitempty"`

	// Params describe the element's declared parameters in
	// flag.FlagSet.VisitAll order (lexicographic). Empty for flagless
	// elements.
	Params []ParamSchema `json:"params,omitempty"`
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

// A Record is one reconciliation unit: the effective desired state of
// one deployment statement.
type Record struct {
	// Verb names the statement class; always [VerbDeploy] in this rung.
	Verb string `json:"verb"`

	// Element resolves into the image's catalogue section.
	Element Ref `json:"element"`

	// Name is the instance name, the reconciliation key.
	Name string `json:"name"`

	// Params are the bound parameters, sorted by Key.
	Params []Binding `json:"params,omitempty"`
}

// A Ref addresses an element pinned in the image's catalogue section.
type Ref struct {
	// Package is the defining package's import path.
	Package string `json:"package"`

	// Name is the element's name within that package.
	Name string `json:"name"`
}

// A Binding is one bound parameter: the key, the canonical value, and
// the provenance of the binding.
type Binding struct {
	// Key is the parameter name.
	Key string `json:"key"`

	// Value is the canonical bound value.
	Value *Value `json:"value"`

	// Source records which layer bound the value; always
	// [SourceInstance] in this rung. It is provenance: [Equal] masks it.
	Source string `json:"source"`
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
// the [Format] this package understands.
func Decode(r io.Reader) (*Image, error) {
	var img Image
	if err := json.NewDecoder(r).Decode(&img); err != nil {
		return nil, fmt.Errorf("image: decode: %w", err)
	}
	if img.Format != Format {
		return nil, fmt.Errorf("image: decode: format %q is not %q", img.Format, Format)
	}
	return &img, nil
}
