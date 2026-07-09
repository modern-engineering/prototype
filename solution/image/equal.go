// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image

import "slices"

// Equal reports whether two images describe the same desired state. All
// provenance is masked: Generation and every [Binding]'s Source never
// enter the comparison, so a round trip through re-rendering and
// re-compilation compares equal even when values arrive through
// different layers or the producer stamped a different generation.
//
// The comparison is structural over the canonical order producers emit
// (packages by path, elements by name, symbols by name, records in
// statement order, bindings by key); images holding the same content
// in a non-canonical order compare unequal.
func Equal(a, b *Image) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Format == b.Format &&
		a.Solution == b.Solution &&
		slices.EqualFunc(a.Catalogue, b.Catalogue, packageEqual) &&
		slices.EqualFunc(a.Symbols, b.Symbols, symbolEqual) &&
		slices.EqualFunc(a.Records, b.Records, recordEqual)
}

func packageEqual(a, b Package) bool {
	return a.Path == b.Path &&
		a.Name == b.Name &&
		slices.EqualFunc(a.Elements, b.Elements, elementEqual)
}

func elementEqual(a, b ElementSchema) bool {
	return a.Name == b.Name &&
		a.Kind == b.Kind &&
		a.Doc == b.Doc &&
		a.Sensitive == b.Sensitive &&
		slices.Equal(a.Params, b.Params) &&
		slices.Equal(a.Outputs, b.Outputs) &&
		slices.Equal(a.Kinds, b.Kinds)
}

func symbolEqual(a, b SymbolDef) bool {
	return a.Name == b.Name &&
		a.Class == b.Class &&
		refEqual(a.Type, b.Type) &&
		valueEqual(a.Value, b.Value)
}

func recordEqual(a, b Record) bool {
	return a.Verb == b.Verb &&
		a.Kind == b.Kind &&
		a.Element == b.Element &&
		a.Name == b.Name &&
		slices.EqualFunc(a.Params, b.Params, bindingEqual)
}

// bindingEqual compares the key, the payload (literal value or symbol
// reference), and the taint; Source is provenance and stays masked.
func bindingEqual(a, b Binding) bool {
	return a.Key == b.Key &&
		a.Sensitive == b.Sensitive &&
		valueEqual(a.Value, b.Value) &&
		symbolRefEqual(a.Ref, b.Ref)
}

func refEqual(a, b *Ref) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func symbolRefEqual(a, b *SymbolRef) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// valueEqual compares the literal a value represents: the kind and the
// arm it selects. Unselected arms are ignored, so a hand-built value
// with residue in another arm still equals its round-tripped, clean
// form.
func valueEqual(a, b *Value) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KindString:
		return a.Str == b.Str
	case KindInt:
		return a.Int == b.Int
	case KindBool:
		return a.Bool == b.Bool
	case KindDuration:
		return a.Dur == b.Dur
	}
	return *a == *b
}
