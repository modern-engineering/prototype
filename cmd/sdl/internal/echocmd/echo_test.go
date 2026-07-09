// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package echocmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// render drives one image through its encoded form into echo, the
// route the command takes.
func render(t *testing.T, img *image.Image) (string, error) {
	t.Helper()
	var encoded bytes.Buffer
	if err := img.Encode(&encoded); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := echo(&encoded, &out)
	return out.String(), err
}

// pkg builds a schema-less catalogue pin: echo reads only Path and
// Name.
func pkg(path, name string) image.Package {
	return image.Package{Path: path, Name: name, Elements: []image.ElementSchema{{Name: "X", Kind: image.KindComponent}}}
}

func record(pkgPath, element, name string, params ...image.Binding) image.Record {
	return image.Record{
		Verb:    image.VerbDeploy,
		Element: image.Ref{Package: pkgPath, Name: element},
		Name:    name,
		Params:  params,
	}
}

func binding(key string, v *image.Value) image.Binding {
	return image.Binding{Key: key, Value: v, Source: image.SourceInstance}
}

// TestEchoGolden pins the whole canonical unit for an image exercising
// the aliasing rules — a package named unlike its path tail, an alias
// collision resolved by numeric suffix — plus the symbol table (two
// externs come back factored, one var single-form, types qualified by
// the same reference names the records use), every value kind, a
// reference binding, an output-reference binding, provision records of
// both kinds (the kind word always written), and a parameterless
// record.
func TestEchoGolden(t *testing.T) {
	img := &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 7,
		Catalogue: []image.Package{
			// Sorted by path, as compilation emits them.
			pkg("example.com/acme/util-go", "util"), // name differs from the path tail
			pkg("example.com/beta/util", "util"),    // collides: becomes util2
			pkg("example.com/ff", "ff"),             // name matches the tail: no alias
		},
		Symbols: []image.SymbolDef{
			{Name: "apiKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme/util-go", Name: "Token"}},
			{Name: "rootKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/beta/util", Name: "Token"}},
			{Name: "subject", Class: image.ClassVar, Value: image.String("com.acme.Echo")},
		},
		Records: []image.Record{
			record("example.com/acme/util-go", "Server", "S1",
				binding("retries", image.Int(-3)),
				binding("timeout", image.Duration(90*time.Minute)),
				image.Binding{Key: "token", Ref: &image.SymbolRef{Symbol: "apiKey"}, Source: image.SourceInstance, Sensitive: true},
				image.Binding{Key: "wire", Ref: &image.SymbolRef{Symbol: "grid", Output: "config"}, Source: image.SourceInstance, Sensitive: true},
			),
			record("example.com/beta/util", "Cache", "C1",
				binding("enabled", image.Bool(true)),
				binding("name", image.String(`say "hi"`)),
			),
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindSlice,
				Element: image.Ref{Package: "example.com/beta/util", Name: "Grid"},
				Name:    "grid",
				Params: []image.Binding{
					{Key: "admin", Ref: &image.SymbolRef{Symbol: "rootKey"}, Source: image.SourceInstance, Sensitive: true},
				},
			},
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindAttach,
				Element: image.Ref{Package: "example.com/ff", Name: "Store"},
				Name:    "legacy",
			},
			record("example.com/ff", "Pong", "Pong"),
		},
	}
	want := `solution sample

import (
	util "example.com/acme/util-go"
	util2 "example.com/beta/util"
	"example.com/ff"
)

extern (
	apiKey util.Token
	rootKey util2.Token
)

var subject: "com.acme.Echo"

deploy util.Server as S1 {
	retries: -3
	timeout: 1h30m0s
	token: apiKey
	wire: grid.config
}

deploy util2.Cache as C1 {
	enabled: true
	name: "say \"hi\""
}

provision util2.Grid slice as grid {
	admin: rootKey
}

provision ff.Store attach as legacy

deploy ff.Pong as Pong
`
	got, err := render(t, img)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("echoed unit:\n%s--- want ---\n%s", got, want)
	}
}

// TestEchoSinglePackage keeps a lone import in single form.
func TestEchoSinglePackage(t *testing.T) {
	img := &image.Image{
		Format:    image.Format,
		Solution:  "one",
		Catalogue: []image.Package{pkg("example.com/ff", "ff")},
		Records:   []image.Record{record("example.com/ff", "Ping", "P")},
	}
	want := `solution one

import "example.com/ff"

deploy ff.Ping as P
`
	got, err := render(t, img)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("echoed unit:\n%s--- want ---\n%s", got, want)
	}
}

// TestEchoFaults exercises the exit-2 material: images echo cannot or
// must not render.
func TestEchoFaults(t *testing.T) {
	valid := func() *image.Image {
		return &image.Image{
			Format:    image.Format,
			Solution:  "sample",
			Catalogue: []image.Package{pkg("example.com/ff", "ff")},
			Records:   []image.Record{record("example.com/ff", "Ping", "P", binding("count", image.Int(1)))},
		}
	}
	tests := map[string]struct {
		mutate func(*image.Image)
		want   string
	}{
		"UnsupportedVerb": {
			mutate: func(img *image.Image) { img.Records[0].Verb = "destroy" },
			want:   `verb "destroy"`,
		},
		"DeployWithProvisionKind": {
			mutate: func(img *image.Image) { img.Records[0].Kind = image.KindSlice },
			want:   `provision kind "slice"`,
		},
		"ProvisionWithUnknownKind": {
			mutate: func(img *image.Image) {
				img.Records[0].Verb = image.VerbProvision
				img.Records[0].Kind = "grow"
			},
			want: `provision kind "grow"`,
		},
		"ProvisionWithoutKind": {
			mutate: func(img *image.Image) { img.Records[0].Verb = image.VerbProvision },
			want:   `provision kind ""`,
		},
		"UnrenderableOutput": {
			mutate: func(img *image.Image) {
				img.Records[0].Params[0] = image.Binding{Key: "count", Ref: &image.SymbolRef{Symbol: "grid", Output: "log-level"}}
			},
			want: "SDL identifier",
		},
		"UnpinnedElementPackage": {
			mutate: func(img *image.Image) { img.Records[0].Element.Package = "example.com/other" },
			want:   "not pinned",
		},
		"NoValue": {
			mutate: func(img *image.Image) { img.Records[0].Params[0].Value = nil },
			want:   "no value",
		},
		"UnrenderableKey": {
			mutate: func(img *image.Image) { img.Records[0].Params[0].Key = "log-level" },
			want:   "SDL identifier",
		},
		"ValueAndRef": {
			mutate: func(img *image.Image) { img.Records[0].Params[0].Ref = &image.SymbolRef{Symbol: "x"} },
			want:   "both a value and a reference",
		},
		"UnrenderableRef": {
			mutate: func(img *image.Image) {
				img.Records[0].Params[0] = image.Binding{Key: "count", Ref: &image.SymbolRef{Symbol: "log-level"}}
			},
			want: "SDL identifier",
		},
		"UnknownSymbolClass": {
			mutate: func(img *image.Image) {
				img.Symbols = []image.SymbolDef{{Name: "x", Class: "weak"}}
			},
			want: `class "weak"`,
		},
		"BadSymbolName": {
			mutate: func(img *image.Image) {
				img.Symbols = []image.SymbolDef{{Name: "not name", Class: image.ClassVar, Value: image.Int(1)}}
			},
			want: "SDL identifier",
		},
		"ExternWithoutType": {
			mutate: func(img *image.Image) {
				img.Symbols = []image.SymbolDef{{Name: "x", Class: image.ClassExtern}}
			},
			want: "has no type",
		},
		"ExternTypeUnpinned": {
			mutate: func(img *image.Image) {
				img.Symbols = []image.SymbolDef{{Name: "x", Class: image.ClassExtern,
					Type: &image.Ref{Package: "example.com/other", Name: "Token"}}}
			},
			want: "not pinned",
		},
		"VarWithoutValue": {
			mutate: func(img *image.Image) {
				img.Symbols = []image.SymbolDef{{Name: "x", Class: image.ClassVar}}
			},
			want: "no value",
		},
		"KeywordInstanceName": {
			mutate: func(img *image.Image) { img.Records[0].Name = "deploy" },
			want:   "SDL identifier",
		},
		"BadPackageName": {
			mutate: func(img *image.Image) { img.Catalogue[0].Name = "not name" },
			want:   "SDL identifier",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			img := valid()
			tt.mutate(img)
			out, err := render(t, img)
			if err == nil {
				t.Fatalf("echo succeeded, want an error\n--- printed ---\n%s", out)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

// TestEchoFormatGate rejects other formats before rendering anything.
func TestEchoFormatGate(t *testing.T) {
	var out bytes.Buffer
	err := echo(strings.NewReader(`{"format":"solution-image/999"}`), &out)
	if err == nil || !strings.Contains(err.Error(), "format") {
		t.Fatalf("err %v, want a format error", err)
	}
	if out.Len() > 0 {
		t.Errorf("echo printed despite the format fault: %q", out.String())
	}
}
