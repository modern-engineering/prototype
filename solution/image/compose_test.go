// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"reflect"
	"strings"
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

// TestValidateAcceptsConsistentImages: the equality suite's image and
// the canonicalization suite's deliberately disordered one both pass —
// validation judges consistency, never order — and so does an image
// whose provenance block is arbitrary, since provenance is not judged
// at all.
func TestValidateAcceptsConsistentImages(t *testing.T) {
	if err := testImage().Validate(); err != nil {
		t.Errorf("Validate(testImage) = %v, want nil", err)
	}
	if err := disorderedImage().Validate(); err != nil {
		t.Errorf("Validate(disorderedImage) = %v, want nil — validation must not judge order", err)
	}
	provenance := testImage()
	provenance.Build = &image.Build{
		Units:    []image.UnitDigest{{Name: "?", SHA256: "not a digest"}},
		Settings: []image.Setting{{Key: "z"}, {Key: "a"}},
	}
	if err := provenance.Validate(); err != nil {
		t.Errorf("Validate = %v, want nil — the Build block is provenance and goes unjudged", err)
	}
}

func TestValidateNilImage(t *testing.T) {
	var img *image.Image
	if err := img.Validate(); err == nil {
		t.Error("Validate(nil) = nil, want an error")
	}
}

// TestValidateJoinsFaults: a broken image reports every fault at once,
// one line each.
func TestValidateJoinsFaults(t *testing.T) {
	img := testImage()
	img.Format = "solution-image/9"
	img.Records[0].Name = ""
	err := img.Validate()
	if err == nil {
		t.Fatal("Validate = nil, want faults")
	}
	for _, want := range []string{
		`image: format "solution-image/9" is not "solution-image/1"`,
		`image: record 0 declares no instance name`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate misses the line %q; got:\n%s", want, err)
		}
	}
}

// TestValidateFindsFaults drives one inconsistency at a time through a
// consistent base image and pins the fault line it must earn.
func TestValidateFindsFaults(t *testing.T) {
	pingpong := "example.com/acme/pingpong"
	tests := []struct {
		name   string
		mutate func(img *image.Image)
		want   string
	}{
		{"foreign format", func(img *image.Image) {
			img.Format = "solution-image/9"
		}, `image: format "solution-image/9" is not "solution-image/1"`},
		{"missing solution name", func(img *image.Image) {
			img.Solution = ""
		}, `image: no solution name`},
		{"zero generation", func(img *image.Image) {
			img.Generation = 0
		}, `image: generation 0 is not positive`},

		{"duplicate package pin", func(img *image.Image) {
			img.Catalogue = append(img.Catalogue, image.Package{Path: pingpong, Name: "pingpong"})
		}, `image: catalogue pins package "example.com/acme/pingpong" twice`},
		{"empty package path", func(img *image.Image) {
			img.Catalogue[0].Path = ""
		}, `image: catalogue pins a package with an empty path`},
		{"nameless package", func(img *image.Image) {
			img.Catalogue[0].Name = ""
		}, `image: package "example.com/acme/pingpong" has no name`},
		{"duplicate element pin", func(img *image.Image) {
			img.Catalogue[0].Elements = append(img.Catalogue[0].Elements, image.ElementSchema{Name: "Ping", Kind: image.KindComponent})
		}, `image: package "example.com/acme/pingpong" pins element Ping twice`},
		{"nameless element", func(img *image.Image) {
			img.Catalogue[0].Elements = append(img.Catalogue[0].Elements, image.ElementSchema{Kind: image.KindComponent})
		}, `image: package "example.com/acme/pingpong" pins an element with an empty name`},
		{"unknown element kind", func(img *image.Image) {
			img.Catalogue[0].Elements[0].Kind = "widget"
		}, `image: element pingpong.Ping: unknown kind "widget"`},
		{"scheme without a qualifier", func(img *image.Image) {
			img.Catalogue[0].Elements = append(img.Catalogue[0].Elements, image.ElementSchema{Name: "Sched", Kind: image.KindScheme})
		}, `image: scheme element pingpong.Sched pins no qualifier`},
		{"qualifier on a component", func(img *image.Image) {
			img.Catalogue[0].Elements[0].Qualifier = "k8s.pod"
		}, `image: element pingpong.Ping pins qualifier "k8s.pod" but is a component, not a scheme`},
		{"duplicate scheme qualifier", func(img *image.Image) {
			img.Catalogue[0].Elements = append(img.Catalogue[0].Elements,
				image.ElementSchema{Name: "Pod", Kind: image.KindScheme, Qualifier: "k8s.pod"},
				image.ElementSchema{Name: "Pod2", Kind: image.KindScheme, Qualifier: "k8s.pod"},
			)
		}, `image: scheme qualifier "k8s.pod" pinned by both pingpong.Pod and pingpong.Pod2`},
		{"duplicate parameter schema", func(img *image.Image) {
			img.Catalogue[0].Elements[0].Params = append(img.Catalogue[0].Elements[0].Params, image.ParamSchema{Name: "count"})
		}, `image: element pingpong.Ping pins parameter count twice`},
		{"duplicate output schema", func(img *image.Image) {
			img.Catalogue[0].Elements[2].Outputs = append(img.Catalogue[0].Elements[2].Outputs, image.OutputSchema{Name: "config"})
		}, `image: element pingpong.Grid pins output config twice`},
		{"unknown registered provision kind", func(img *image.Image) {
			img.Catalogue[0].Elements[2].Kinds = append(img.Catalogue[0].Elements[2].Kinds, "blob")
		}, `image: element pingpong.Grid registers unknown provision kind "blob"`},

		{"duplicate symbol", func(img *image.Image) {
			img.Symbols = append(img.Symbols, image.SymbolDef{Name: "subject", Class: image.ClassVar, Value: image.String("x")})
		}, `image: symbol subject declared twice`},
		{"nameless symbol", func(img *image.Image) {
			img.Symbols[1].Name = ""
		}, `image: a symbol declares no name`},
		{"unknown symbol class", func(img *image.Image) {
			img.Symbols[0].Class = "static"
		}, `image: symbol adminKey: unknown class "static"`},
		{"extern with a value", func(img *image.Image) {
			img.Symbols[0].Value = image.String("leak")
		}, `image: extern adminKey pins a value, which only vars carry`},
		{"extern without a type", func(img *image.Image) {
			img.Symbols[0].Type = nil
		}, `image: extern adminKey declares no type`},
		{"extern type unpinned", func(img *image.Image) {
			img.Symbols[0].Type = &image.Ref{Package: pingpong, Name: "Missing"}
		}, `image: extern adminKey declares type pingpong.Missing but the image pins no such element`},
		{"extern type not a symbol type", func(img *image.Image) {
			img.Symbols[0].Type = &image.Ref{Package: pingpong, Name: "Ping"}
		}, `image: extern adminKey declares type pingpong.Ping but the image pins a component, not a symbol type`},
		{"var with a type", func(img *image.Image) {
			img.Symbols[1].Type = &image.Ref{Package: pingpong, Name: "Token"}
		}, `image: var subject declares a type, which only externs carry`},
		{"var without a value", func(img *image.Image) {
			img.Symbols[1].Value = nil
		}, `image: var subject pins no value`},
		{"var pinning a token", func(img *image.Image) {
			img.Symbols[1].Value = image.Token("euCentral1")
		}, `image: var subject pins opaque token "euCentral1"; tokens belong to the deployment and extension compartments`},
		{"var of unknown value kind", func(img *image.Image) {
			img.Symbols[1].Value = &image.Value{Kind: "blob"}
		}, `image: var subject: unknown value kind "blob"`},

		{"duplicate instance", func(img *image.Image) {
			img.Records[1].Name = "Ping1"
		}, `image: instance Ping1 declared twice`},
		{"nameless record", func(img *image.Image) {
			img.Records[0].Name = ""
		}, `image: record 0 declares no instance name`},
		{"unknown verb", func(img *image.Image) {
			img.Records[0].Verb = "destroy"
		}, `image: record Ping1 declares unknown verb "destroy"`},
		{"deploy with a provision kind", func(img *image.Image) {
			img.Records[0].Kind = image.KindSlice
		}, `image: deploy record Ping1 carries provision kind "slice"`},
		{"provision without a kind", func(img *image.Image) {
			img.Records[1].Kind = ""
		}, `image: record grid1 declares unknown provision kind ""`},
		{"record element unpinned", func(img *image.Image) {
			img.Records[0].Element.Name = "Missing"
		}, `image: record Ping1 references pingpong.Missing but the image pins no such element`},
		{"deploy of a provision pin", func(img *image.Image) {
			img.Records[0].Element.Name = "Grid"
		}, `image: record Ping1 deploys pingpong.Grid, which the image pins as a provision, not a component`},
		{"provision of a component pin", func(img *image.Image) {
			img.Records[1].Element.Name = "Ping"
		}, `image: record grid1 provisions pingpong.Ping, which the image pins as a component, not a provision type`},

		{"binding with both arms", func(img *image.Image) {
			img.Records[0].Params[0].Ref = &image.SymbolRef{Symbol: "subject"}
		}, `image: record Ping1: binding count carries both a value and a reference`},
		{"binding with neither arm", func(img *image.Image) {
			img.Records[0].Params[0].Value = nil
		}, `image: record Ping1: binding count carries neither value nor reference`},
		{"token in params", func(img *image.Image) {
			img.Records[0].Params[0].Value = image.Token("euCentral1")
		}, `image: record Ping1: binding count pins opaque token "euCentral1"; tokens belong to the deployment and extension compartments`},
		{"unknown value kind", func(img *image.Image) {
			img.Records[0].Params[0].Value = &image.Value{Kind: "blob"}
		}, `image: record Ping1: binding count: unknown value kind "blob"`},
		{"duplicate binding key", func(img *image.Image) {
			img.Records[0].Params = append(img.Records[0].Params, img.Records[0].Params[0])
		}, `image: record Ping1 binds count twice in params`},
		{"unknown binding source", func(img *image.Image) {
			img.Records[0].Params[0].Source = "guess"
		}, `image: record Ping1: binding count records unknown source "guess"`},
		{"undeclared symbol", func(img *image.Image) {
			img.Records[0].Params[3].Ref = &image.SymbolRef{Symbol: "nobody"}
		}, `image: record Ping1: binding target references undeclared symbol nobody`},
		{"reference to no symbol", func(img *image.Image) {
			img.Records[0].Params[3].Ref = &image.SymbolRef{}
		}, `image: record Ping1: binding target references no symbol`},
		{"output of a missing instance", func(img *image.Image) {
			img.Records[0].Params[4].Ref = &image.SymbolRef{Symbol: "ghost", Output: "config"}
		}, `image: record Ping1: binding wire references output ghost.config but the image provisions no instance ghost`},
		{"output of a deploy record", func(img *image.Image) {
			img.Records[0].Params[4].Ref = &image.SymbolRef{Symbol: "Ping1", Output: "config"}
		}, `image: record Ping1: binding wire references output Ping1.config but the image provisions no instance Ping1`},
		{"output the pin lacks", func(img *image.Image) {
			img.Records[0].Params[4].Ref = &image.SymbolRef{Symbol: "grid1", Output: "url"}
		}, `image: record Ping1: binding wire references output grid1.url but pingpong.Grid pins no such output`},

		{"reference in deployment", func(img *image.Image) {
			img.Records[0].Deployment[0] = image.Binding{Key: "location", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance}
		}, `image: record Ping1: top-level field location carries a symbol reference; top-level values never resolve symbols`},
		{"deployment field without a value", func(img *image.Image) {
			img.Records[0].Deployment[0].Value = nil
		}, `image: record Ping1: top-level field location carries no value`},
		{"duplicate deployment field", func(img *image.Image) {
			img.Records[0].Deployment = append(img.Records[0].Deployment, img.Records[0].Deployment[0])
		}, `image: record Ping1 binds location twice in deployment`},
		{"reference in a with-stanza", func(img *image.Image) {
			img.Records[0].Extensions["k8s.pod"][1] = image.Binding{Key: "replicas", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance}
		}, `image: record Ping1: with k8s.pod parameter replicas carries a symbol reference; with-stanza values never resolve symbols`},
		{"duplicate stanza key", func(img *image.Image) {
			ext := img.Records[0].Extensions
			ext["k8s.pod"] = append(ext["k8s.pod"], ext["k8s.pod"][1])
		}, `image: record Ping1 binds replicas twice in with k8s.pod`},
		{"stanza with no qualifier", func(img *image.Image) {
			img.Records[0].Extensions[""] = nil
		}, `image: record Ping1 declares a with-stanza with no qualifier`},
		{"metadata non-string", func(img *image.Image) {
			img.Records[1].Metadata[0].Value = image.Int(3)
		}, `image: record grid1: metadata parameter team: metadata values are strings, not "int"`},
		{"reference in metadata", func(img *image.Image) {
			img.Records[1].Metadata[0] = image.Binding{Key: "team", Ref: &image.SymbolRef{Symbol: "subject"}, Source: image.SourceInstance}
		}, `image: record grid1: metadata parameter team carries a symbol reference; metadata values never resolve symbols`},
		{"duplicate metadata key", func(img *image.Image) {
			img.Records[1].Metadata = append(img.Records[1].Metadata, img.Records[1].Metadata[0])
		}, `image: record grid1 binds team twice in metadata`},

		{"self provision cycle", func(img *image.Image) {
			img.Records[1].Params = append(img.Records[1].Params,
				image.Binding{Key: "loop", Ref: &image.SymbolRef{Symbol: "grid1", Output: "config"}, Source: image.SourceInstance})
		}, `image: provision reference cycle: grid1 -> grid1`},
		{"two-step provision cycle", func(img *image.Image) {
			img.Records[1].Params = append(img.Records[1].Params,
				image.Binding{Key: "loop", Ref: &image.SymbolRef{Symbol: "grid2", Output: "config"}, Source: image.SourceInstance})
			img.Records = append(img.Records, image.Record{
				Verb:    image.VerbProvision,
				Kind:    image.KindAttach,
				Element: image.Ref{Package: pingpong, Name: "Grid"},
				Name:    "grid2",
				Params: []image.Binding{
					{Key: "loop", Ref: &image.SymbolRef{Symbol: "grid1", Output: "config"}, Source: image.SourceInstance},
				},
			})
		}, `image: provision reference cycle: grid1 -> grid2 -> grid1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := testImage()
			tt.mutate(img)
			err := img.Validate()
			if err == nil {
				t.Fatalf("Validate = nil, want the fault %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Validate misses the fault %q; got:\n%s", tt.want, err)
			}
		})
	}
}
