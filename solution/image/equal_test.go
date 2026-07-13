// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// testImage builds a fresh image with one pinned package (a component,
// a symbol type, and a provision type), a symbol of each class, and
// two records: a deploy binding literals, a plain reference, a tainted
// reference, and a tainted output reference, carrying a deployment
// field and two extension stanzas (one empty, pinning that presence
// counts), and a provision slice binding a literal; every call returns
// an independent value so tests may mutate freely.
func testImage() *image.Image {
	return &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 1,
		Catalogue: []image.Package{{
			Path: "example.com/acme/pingpong",
			Name: "pingpong",
			Elements: []image.ElementSchema{
				{
					Name: "Ping",
					Kind: image.KindComponent,
					Doc:  "ping a subject at a fixed interval",
					Params: []image.ParamSchema{
						{Name: "count", Usage: "number of pings", Default: "1"},
						{Name: "interval", Usage: "delay between pings", Default: "30s"},
					},
				},
				{
					Name:      "Token",
					Kind:      image.KindSymbol,
					Doc:       "an operator-held credential",
					Sensitive: true,
				},
				{
					Name:    "Grid",
					Kind:    image.KindProvision,
					Doc:     "an account on the shared grid",
					Params:  []image.ParamSchema{{Name: "cluster", Usage: "cluster to hold the account"}},
					Outputs: []image.OutputSchema{{Name: "config", Sensitive: true}},
					Kinds:   []string{image.KindSlice, image.KindAttach},
				},
			},
		}},
		Symbols: []image.SymbolDef{
			{Name: "adminKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme/pingpong", Name: "Token"}},
			{Name: "subject", Class: image.ClassVar, Value: image.String("com.acme.Echo")},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/acme/pingpong", Name: "Ping"},
				Name:    "Ping1",
				Params: []image.Binding{
					{Key: "count", Value: image.Int(-1), Source: image.SourceInstance},
					{Key: "interval", Value: image.Duration(time.Second), Source: image.SourceInstance},
					{Key: "key", Ref: &image.SymbolRef{Symbol: "adminKey"}, Source: image.SourceInstance, Sensitive: true},
					{Key: "target", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance},
					{Key: "wire", Ref: &image.SymbolRef{Symbol: "grid1", Output: "config"}, Source: image.SourceInstance, Sensitive: true},
				},
				Deployment: []image.Binding{
					{Key: "location", Value: image.Token("euCentral1"), Source: image.SourceDefaultDeploy},
				},
				Extensions: map[string][]image.Binding{
					"k8s.pod": {
						{Key: "priorityClass", Value: image.Token("standard"), Source: image.SourceDefaultDeploy},
						{Key: "replicas", Value: image.Int(3), Source: image.SourceInstance},
					},
					"k8s.workload": {},
				},
			},
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindSlice,
				Element: image.Ref{Package: "example.com/acme/pingpong", Name: "Grid"},
				Name:    "grid1",
				Params: []image.Binding{
					{Key: "cluster", Value: image.String("us-east"), Source: image.SourceInstance},
				},
				Metadata: []image.Binding{
					{Key: "team", Value: image.String("search"), Source: image.SourceDefaultProvision},
				},
			},
		},
	}
}

func TestEqualMasksProvenance(t *testing.T) {
	base := testImage()

	same := testImage()
	if !image.Equal(base, same) {
		t.Fatal("Equal(base, identical copy) = false, want true")
	}

	generation := testImage()
	generation.Generation = 99
	if !image.Equal(base, generation) {
		t.Error("Equal must mask Generation: differing generations compared unequal")
	}

	source := testImage()
	source.Records[0].Params[0].Source = "default-type"
	if !image.Equal(base, source) {
		t.Error("Equal must mask Binding.Source: differing sources compared unequal")
	}

	compartment := testImage()
	compartment.Records[1].Metadata[0].Source = image.SourceInstance
	if !image.Equal(base, compartment) {
		t.Error("Equal must mask Source in metadata bindings too")
	}

	routed := testImage()
	routed.Records[0].Deployment[0].Source = image.SourceInstance
	routed.Records[0].Extensions["k8s.pod"][0].Source = image.SourceInstance
	if !image.Equal(base, routed) {
		t.Error("Equal must mask Source in deployment and extension bindings too")
	}

	both := testImage()
	both.Generation = 7
	for i := range both.Records[0].Params {
		both.Records[0].Params[i].Source = "default-deploy"
	}
	if !image.Equal(base, both) {
		t.Error("Equal must mask all provenance at once")
	}
}

func TestEqualCatchesRealDifferences(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(img *image.Image)
	}{
		{"param value change", func(img *image.Image) {
			img.Records[0].Params[0].Value = image.Int(-2)
		}},
		{"param value kind change", func(img *image.Image) {
			img.Records[0].Params[0].Value = image.String("-1")
		}},
		{"binding key change", func(img *image.Image) {
			img.Records[0].Params[0].Key = "repeat"
		}},
		{"binding dropped", func(img *image.Image) {
			img.Records[0].Params = img.Records[0].Params[:1]
		}},
		{"record rename", func(img *image.Image) {
			img.Records[0].Name = "Ping2"
		}},
		{"record element change", func(img *image.Image) {
			img.Records[0].Element.Name = "Pong"
		}},
		{"record added", func(img *image.Image) {
			img.Records = append(img.Records, image.Record{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/acme/pingpong", Name: "Ping"},
				Name:    "Ping2",
			})
		}},
		{"catalogue schema change", func(img *image.Image) {
			img.Catalogue[0].Elements[0].Params[0].Default = "2"
		}},
		{"catalogue doc change", func(img *image.Image) {
			img.Catalogue[0].Elements[0].Doc = "changed"
		}},
		{"catalogue package rename", func(img *image.Image) {
			img.Catalogue[0].Name = "pp"
		}},
		{"catalogue sensitivity change", func(img *image.Image) {
			img.Catalogue[0].Elements[1].Sensitive = false
		}},
		{"solution rename", func(img *image.Image) {
			img.Solution = "other"
		}},
		{"symbol rename", func(img *image.Image) {
			img.Symbols[0].Name = "rootKey"
		}},
		{"symbol class change", func(img *image.Image) {
			img.Symbols[1].Class = image.ClassExtern
		}},
		{"symbol type change", func(img *image.Image) {
			img.Symbols[0].Type = &image.Ref{Package: "example.com/acme/pingpong", Name: "Other"}
		}},
		{"symbol type dropped", func(img *image.Image) {
			img.Symbols[0].Type = nil
		}},
		{"symbol value change", func(img *image.Image) {
			img.Symbols[1].Value = image.String("com.acme.Other")
		}},
		{"symbol dropped", func(img *image.Image) {
			img.Symbols = img.Symbols[:1]
		}},
		{"binding ref change", func(img *image.Image) {
			img.Records[0].Params[3].Ref = &image.SymbolRef{Symbol: "other"}
		}},
		{"binding ref becomes value", func(img *image.Image) {
			img.Records[0].Params[3].Ref = nil
			img.Records[0].Params[3].Value = image.String("com.acme.Echo")
		}},
		{"binding taint change", func(img *image.Image) {
			img.Records[0].Params[2].Sensitive = false
		}},
		{"record kind change", func(img *image.Image) {
			img.Records[1].Kind = image.KindAttach
		}},
		{"binding output change", func(img *image.Image) {
			img.Records[0].Params[4].Ref = &image.SymbolRef{Symbol: "grid1", Output: "url"}
		}},
		{"output ref becomes symbol ref", func(img *image.Image) {
			img.Records[0].Params[4].Ref = &image.SymbolRef{Symbol: "grid1"}
		}},
		{"catalogue output change", func(img *image.Image) {
			img.Catalogue[0].Elements[2].Outputs[0].Sensitive = false
		}},
		{"catalogue output type change", func(img *image.Image) {
			img.Catalogue[0].Elements[2].Outputs[0].Type = "int"
		}},
		{"catalogue provision kinds change", func(img *image.Image) {
			img.Catalogue[0].Elements[2].Kinds = []string{image.KindSlice}
		}},
		{"deployment field change", func(img *image.Image) {
			img.Records[0].Deployment[0].Value = image.Token("usEast1")
		}},
		{"deployment binding dropped", func(img *image.Image) {
			img.Records[0].Deployment = nil
		}},
		{"extension binding change", func(img *image.Image) {
			img.Records[0].Extensions["k8s.pod"][1].Value = image.Int(4)
		}},
		{"extension qualifier renamed", func(img *image.Image) {
			ext := img.Records[0].Extensions
			ext["k8s.job"] = ext["k8s.pod"]
			delete(ext, "k8s.pod")
		}},
		{"empty extension stanza dropped", func(img *image.Image) {
			delete(img.Records[0].Extensions, "k8s.workload")
		}},
		{"metadata value change", func(img *image.Image) {
			img.Records[1].Metadata[0].Value = image.String("core")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := testImage()
			changed := testImage()
			tt.mutate(changed)
			if image.Equal(base, changed) {
				t.Error("Equal = true after a real difference, want false")
			}
		})
	}
}

// TestEqualExtensionsNilEmpty pins the two nil-tolerance levels of the
// extensions comparison: an absent map equals an empty one, and a nil
// stanza equals an empty stanza — while stanza presence itself stays a
// real difference (covered by the mutation cases above).
func TestEqualExtensionsNilEmpty(t *testing.T) {
	emptyMap := testImage()
	emptyMap.Records[1].Extensions = map[string][]image.Binding{}
	if !image.Equal(testImage(), emptyMap) {
		t.Error("Equal(nil extensions, empty extensions) = false, want true")
	}

	nilStanza := testImage()
	nilStanza.Records[0].Extensions["k8s.workload"] = nil
	if !image.Equal(testImage(), nilStanza) {
		t.Error("Equal(empty stanza, nil stanza) = false, want true")
	}
}

func TestEqualNilAndDirtyArms(t *testing.T) {
	if !image.Equal(nil, nil) {
		t.Error("Equal(nil, nil) = false, want true")
	}
	if image.Equal(testImage(), nil) || image.Equal(nil, testImage()) {
		t.Error("Equal(image, nil) = true, want false")
	}

	// Equality follows the represented literal: residue in an
	// unselected arm does not count.
	dirty := testImage()
	dirty.Records[0].Params[0].Value = &image.Value{Kind: image.KindInt, Int: -1, Str: "junk"}
	if !image.Equal(testImage(), dirty) {
		t.Error("Equal must ignore unselected value arms")
	}
}
