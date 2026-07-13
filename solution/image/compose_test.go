// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"reflect"
	"testing"

	"github.com/modern-engineering/prototype/solution/image"
)

// disorderedImage composes an image with every sortable slice
// deliberately out of order — packages, elements, schemas, kinds,
// symbols, settings, and all four binding compartments — while the
// records themselves stand in an order no sort would produce (deploy
// "b" before provision "a"), pinning that record order is meaning, not
// disorder.
func disorderedImage() *image.Image {
	return &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 1,
		Build: &image.Build{
			Units: []image.UnitDigest{
				// Unit order is record order: never sorted.
				{Name: "z.sdl", SHA256: "beef"},
				{Name: "a.sdl", SHA256: "cafe"},
			},
			Settings: []image.Setting{
				{Key: "sdl.version", Value: "v0.1.2"},
				{Key: "prototype.version", Value: "v0.1.0"},
			},
		},
		Catalogue: []image.Package{
			{
				Path: "example.com/beta",
				Name: "beta",
				Elements: []image.ElementSchema{
					{
						Name: "Grid",
						Kind: image.KindProvision,
						Params: []image.ParamSchema{
							{Name: "cluster"},
							{Name: "account"},
						},
						Outputs: []image.OutputSchema{
							{Name: "token", Sensitive: true},
							{Name: "config"},
						},
						Kinds: []string{image.KindAttach, image.KindSlice},
					},
					{Name: "Echo", Kind: image.KindComponent},
				},
			},
			{
				Path: "example.com/acme",
				Name: "acme",
				Elements: []image.ElementSchema{
					{Name: "Token", Kind: image.KindSymbol, Sensitive: true},
				},
			},
		},
		Symbols: []image.SymbolDef{
			{Name: "subject", Class: image.ClassVar, Value: image.String("wheel")},
			{Name: "adminKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme", Name: "Token"}},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/beta", Name: "Echo"},
				Name:    "b",
				Params: []image.Binding{
					{Key: "target", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance},
					{Key: "count", Value: image.Int(1), Source: image.SourceInstance},
				},
				Deployment: []image.Binding{
					{Key: "location", Value: image.Token("euCentral1"), Source: image.SourceInstance},
					{Key: "class", Value: image.Token("batch"), Source: image.SourceInstance},
				},
				Extensions: map[string][]image.Binding{
					"k8s.pod": {
						{Key: "replicas", Value: image.Int(3), Source: image.SourceInstance},
						{Key: "priorityClass", Value: image.Token("standard"), Source: image.SourceInstance},
					},
				},
				Metadata: []image.Binding{
					{Key: "tier", Value: image.String("gold"), Source: image.SourceInstance},
					{Key: "team", Value: image.String("search"), Source: image.SourceInstance},
				},
			},
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindSlice,
				Element: image.Ref{Package: "example.com/beta", Name: "Grid"},
				Name:    "a",
			},
		},
	}
}

// canonicalImage is disorderedImage's canonical twin, spelled out in
// full so the assertion pins every order Canonicalize owes rather than
// re-deriving them.
func canonicalImage() *image.Image {
	return &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 1,
		Build: &image.Build{
			Units: []image.UnitDigest{
				{Name: "z.sdl", SHA256: "beef"},
				{Name: "a.sdl", SHA256: "cafe"},
			},
			Settings: []image.Setting{
				{Key: "prototype.version", Value: "v0.1.0"},
				{Key: "sdl.version", Value: "v0.1.2"},
			},
		},
		Catalogue: []image.Package{
			{
				Path: "example.com/acme",
				Name: "acme",
				Elements: []image.ElementSchema{
					{Name: "Token", Kind: image.KindSymbol, Sensitive: true},
				},
			},
			{
				Path: "example.com/beta",
				Name: "beta",
				Elements: []image.ElementSchema{
					{Name: "Echo", Kind: image.KindComponent},
					{
						Name: "Grid",
						Kind: image.KindProvision,
						Params: []image.ParamSchema{
							{Name: "account"},
							{Name: "cluster"},
						},
						Outputs: []image.OutputSchema{
							{Name: "config"},
							{Name: "token", Sensitive: true},
						},
						Kinds: []string{image.KindSlice, image.KindAttach},
					},
				},
			},
		},
		Symbols: []image.SymbolDef{
			{Name: "adminKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme", Name: "Token"}},
			{Name: "subject", Class: image.ClassVar, Value: image.String("wheel")},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/beta", Name: "Echo"},
				Name:    "b",
				Params: []image.Binding{
					{Key: "count", Value: image.Int(1), Source: image.SourceInstance},
					{Key: "target", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance},
				},
				Deployment: []image.Binding{
					{Key: "class", Value: image.Token("batch"), Source: image.SourceInstance},
					{Key: "location", Value: image.Token("euCentral1"), Source: image.SourceInstance},
				},
				Extensions: map[string][]image.Binding{
					"k8s.pod": {
						{Key: "priorityClass", Value: image.Token("standard"), Source: image.SourceInstance},
						{Key: "replicas", Value: image.Int(3), Source: image.SourceInstance},
					},
				},
				Metadata: []image.Binding{
					{Key: "team", Value: image.String("search"), Source: image.SourceInstance},
					{Key: "tier", Value: image.String("gold"), Source: image.SourceInstance},
				},
			},
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindSlice,
				Element: image.Ref{Package: "example.com/beta", Name: "Grid"},
				Name:    "a",
			},
		},
	}
}

func TestCanonicalize(t *testing.T) {
	img := disorderedImage()
	img.Canonicalize()
	if want := canonicalImage(); !reflect.DeepEqual(img, want) {
		t.Errorf("Canonicalize:\n got %+v\nwant %+v", img, want)
	}

	img.Canonicalize()
	if want := canonicalImage(); !reflect.DeepEqual(img, want) {
		t.Error("Canonicalize is not idempotent: a second pass changed the image")
	}
}

// TestCanonicalizeLiftsNilSections pins the nil-to-empty lift: a
// minimal hand-composed image encodes its catalogue, symbols, and
// records as empty arrays — the spelling compiled images carry — never
// as null.
func TestCanonicalizeLiftsNilSections(t *testing.T) {
	img := &image.Image{Format: image.Format, Solution: "bare", Generation: 1}
	img.Canonicalize()
	if img.Catalogue == nil || img.Symbols == nil || img.Records == nil {
		t.Errorf("Canonicalize left a nil section: catalogue %v, symbols %v, records %v",
			img.Catalogue == nil, img.Symbols == nil, img.Records == nil)
	}
}
