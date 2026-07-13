// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The composer's half of the package: what a program building [Image]
// values by struct literal calls before encoding, so hand-composed
// images meet the contracts compiled images meet by construction.

package image

import (
	"cmp"
	"slices"
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
