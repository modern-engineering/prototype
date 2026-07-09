// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// testImage builds a fresh image with one pinned package and one record;
// every call returns an independent value so tests may mutate freely.
func testImage() *image.Image {
	return &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 1,
		Catalogue: []image.Package{{
			Path: "example.com/acme/pingpong",
			Name: "pingpong",
			Elements: []image.ElementSchema{{
				Name: "Ping",
				Kind: image.KindComponent,
				Doc:  "ping a subject at a fixed interval",
				Params: []image.ParamSchema{
					{Name: "count", Usage: "number of pings", Default: "1"},
					{Name: "interval", Usage: "delay between pings", Default: "30s"},
				},
			}},
		}},
		Records: []image.Record{{
			Verb:    image.VerbDeploy,
			Element: image.Ref{Package: "example.com/acme/pingpong", Name: "Ping"},
			Name:    "Ping1",
			Params: []image.Binding{
				{Key: "count", Value: image.Int(-1), Source: image.SourceInstance},
				{Key: "interval", Value: image.Duration(time.Second), Source: image.SourceInstance},
			},
		}},
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
		{"solution rename", func(img *image.Image) {
			img.Solution = "other"
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
