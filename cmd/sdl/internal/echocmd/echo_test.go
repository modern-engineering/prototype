// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package echocmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/sdl/parser"
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
// dotted parameter key, a reference binding, an output-reference
// binding, the full compartment set (a top-level field, a params
// section, two extension stanzas — one folded from a default, one
// empty but still naming its scheme — and metadata), provision records
// of both kinds (the kind word always written), and a parameterless
// record. The rendered unit must also parse cleanly, the other half
// of the round-trip contract.
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
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/acme/util-go", Name: "Server"},
				Name:    "S1",
				Params: []image.Binding{
					binding("retries", image.Int(-3)),
					binding("retry.backoff.base", image.Duration(250*time.Millisecond)),
					binding("timeout", image.Duration(90*time.Minute)),
					{Key: "token", Ref: &image.SymbolRef{Symbol: "apiKey"}, Source: image.SourceInstance, Sensitive: true},
					{Key: "wire", Ref: &image.SymbolRef{Symbol: "grid", Output: "config"}, Source: image.SourceInstance, Sensitive: true},
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
				Metadata: []image.Binding{
					{Key: "team", Value: image.String("search"), Source: image.SourceInstance},
					{Key: "tier", Value: image.String("gold"), Source: image.SourceInstance},
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
	location: euCentral1
	params {
		retries: -3
		retry.backoff.base: 250ms
		timeout: 1h30m0s
		token: apiKey
		wire: grid.config
	}
	with k8s.pod {
		priorityClass: standard
		replicas: 3
	}
	with k8s.workload {}
}

deploy util2.Cache as C1 {
	params {
		enabled: true
		name: "say \"hi\""
	}
}

provision util2.Grid slice as grid {
	params {
		admin: rootKey
	}
	metadata {
		team: "search"
		tier: "gold"
	}
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
	if _, err := parser.ParseFile("echo.sdl", []byte(got)); err != nil {
		t.Errorf("echoed unit does not parse: %v", err)
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

// TestEchoKeywordPackageName routes a package whose registered Go name
// is an SDL keyword through the alias machinery: the reference name
// takes the numeric suffix, exactly as a name collision would, since
// the bare keyword could never appear in a reference — and the unit
// renders instead of failing.
func TestEchoKeywordPackageName(t *testing.T) {
	img := &image.Image{
		Format:    image.Format,
		Solution:  "kw",
		Catalogue: []image.Package{pkg("example.com/extern", "extern")},
		Records:   []image.Record{record("example.com/extern", "Ping", "P")},
	}
	want := `solution kw

import extern2 "example.com/extern"

deploy extern2.Ping as P
`
	got, err := render(t, img)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("echoed unit:\n%s--- want ---\n%s", got, want)
	}
	if _, err := parser.ParseFile("echo.sdl", []byte(got)); err != nil {
		t.Errorf("echoed unit does not parse: %v", err)
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
		"TokenInParams": {
			mutate: func(img *image.Image) { img.Records[0].Params[0].Value = image.Token("here") },
			want:   `value kind "token"`,
		},
		"TokenInMetadata": {
			mutate: func(img *image.Image) {
				img.Records[0].Metadata = []image.Binding{{Key: "team", Value: image.Token("search")}}
			},
			want: "metadata values are strings",
		},
		"RefInMetadata": {
			mutate: func(img *image.Image) {
				img.Records[0].Metadata = []image.Binding{{Key: "team", Ref: &image.SymbolRef{Symbol: "subject"}}}
			},
			want: "never resolve symbols",
		},
		"RefInDeployment": {
			mutate: func(img *image.Image) {
				img.Records[0].Deployment = []image.Binding{{Key: "location", Ref: &image.SymbolRef{Symbol: "subject"}}}
			},
			want: "never resolve symbols",
		},
		"RefInExtension": {
			mutate: func(img *image.Image) {
				img.Records[0].Extensions = map[string][]image.Binding{
					"k8s.pod": {{Key: "replicas", Ref: &image.SymbolRef{Symbol: "subject"}}},
				}
			},
			want: "never resolve symbols",
		},
		"UnrenderableToken": {
			mutate: func(img *image.Image) {
				img.Records[0].Deployment = []image.Binding{{Key: "location", Value: image.Token("eu west")}}
			},
			want: "SDL identifier",
		},
		"UnrenderableQualifier": {
			mutate: func(img *image.Image) {
				img.Records[0].Extensions = map[string][]image.Binding{"eu west": {}}
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

// TestEchoComposedImage closes the loop for the programmatic frontend:
// an image composed by struct literal — no source text anywhere in its
// life — walks the composer's whole chain (Canonicalize, Validate,
// Encode) and echoes as the same canonical unit a compiled pingpong
// would, proving the IR is one meeting point for every frontend.
func TestEchoComposedImage(t *testing.T) {
	const (
		ffPath        = "github.com/modern-engineering/prototype/examples/ff"
		substratePath = "github.com/modern-engineering/prototype/examples/substrate"
	)
	symbol := func(key, name string) image.Binding {
		return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: name}, Source: image.SourceInstance}
	}
	output := func(key, instance, output string) image.Binding {
		return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: instance, Output: output}, Source: image.SourceInstance}
	}
	img := &image.Image{
		Format:     image.Format,
		Solution:   "pingpong",
		Generation: 1,
		Catalogue: []image.Package{
			{Path: substratePath, Name: "substrate", Elements: []image.ElementSchema{
				{Name: "StandIn", Kind: image.KindProvision,
					Outputs: []image.OutputSchema{{Name: "config", Type: "string"}},
					Kinds:   []string{image.KindAttach}},
				{Name: "Endpoint", Kind: image.KindSymbol},
			}},
			{Path: ffPath, Name: "ff", Elements: []image.ElementSchema{
				{Name: "Ping", Kind: image.KindComponent},
				{Name: "Pong", Kind: image.KindComponent},
			}},
		},
		Symbols: []image.SymbolDef{
			{Name: "natsEndpoint", Class: image.ClassExtern, Type: &image.Ref{Package: substratePath, Name: "Endpoint"}},
			{Name: "echoSubject", Class: image.ClassVar, Value: image.String("ping")},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindAttach,
				Element: image.Ref{Package: substratePath, Name: "StandIn"},
				Name:    "natsStandIn",
				Params:  []image.Binding{symbol("endpoint", "natsEndpoint")},
			},
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: ffPath, Name: "Ping"},
				Name:    "Ping1",
				Params: []image.Binding{
					symbol("target", "echoSubject"),
					binding("count", image.Int(3)),
					output("nats", "natsStandIn", "config"),
					binding("interval", image.Duration(500*time.Millisecond)),
				},
			},
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: ffPath, Name: "Pong"},
				Name:    "Pong",
				Params: []image.Binding{
					symbol("subject", "echoSubject"),
					output("nats", "natsStandIn", "config"),
				},
			},
		},
	}
	img.Canonicalize()
	if err := img.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	want := `solution pingpong

import (
	"github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/substrate"
)

extern natsEndpoint substrate.Endpoint

var echoSubject: "ping"

provision substrate.StandIn attach as natsStandIn {
	params {
		endpoint: natsEndpoint
	}
}

deploy ff.Ping as Ping1 {
	params {
		count: 3
		interval: 500ms
		nats: natsStandIn.config
		target: echoSubject
	}
}

deploy ff.Pong as Pong {
	params {
		nats: natsStandIn.config
		subject: echoSubject
	}
}
`
	got, err := render(t, img)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("echoed unit:\n%s--- want ---\n%s", got, want)
	}
	if _, err := parser.ParseFile("echo.sdl", []byte(got)); err != nil {
		t.Errorf("echoed unit does not parse: %v", err)
	}
}
