// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/image"
)

// ----------------------------------------------------------------------------
// Test catalogue
//
// Real application.Descriptor values, in the shapes catalogue packages
// use: a MakeFunc constructor declaring typed flags (with a flag.Func
// flag whose String() is empty, proving canonical values never come from
// flag echo), a second flagged component, and a flagless Main adapter.

func idle() application.Runner {
	return application.RunnerFunc(func(context.Context) error { return nil })
}

func pingDescriptor() *application.Descriptor {
	return &application.Descriptor{
		Name: "ping",
		Doc:  "ping a subject at a fixed interval",
		Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
			fs := flag.NewFlagSet("ping", flag.ContinueOnError)
			fs.Int64("count", 1, "number of pings; -1 loops forever")
			fs.Duration("interval", 30*time.Second, "delay between pings")
			fs.String("target", "", "subject to ping")
			fs.Bool("verbose", false, "log every exchange")
			fs.Func("window", "reporting window", func(s string) error {
				_, err := time.ParseDuration(s)
				return err
			})
			return idle(), fs
		}),
	}
}

func pongDescriptor() *application.Descriptor {
	return &application.Descriptor{
		Name: "pong",
		Doc:  "answer pings on a subject",
		Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
			fs := flag.NewFlagSet("pong", flag.ContinueOnError)
			fs.String("subject", "", "subject to answer")
			return idle(), fs
		}),
	}
}

func quietDescriptor() *application.Descriptor {
	return &application.Descriptor{
		Name: "quiet",
		Doc:  "run silently",
		Make: func() application.Service {
			return application.Main(func(context.Context) error { return nil })
		},
	}
}

func boomDescriptor() *application.Descriptor {
	return &application.Descriptor{
		Name: "boom",
		Doc:  "panic on construction",
		Make: func() application.Service { panic("kaboom") },
	}
}

// busProvisionType registers both kinds, a two-slot parameter surface,
// and a mixed-sensitivity output scheme listed out of canonical order
// (the image must sort outputs by name).
func busProvisionType() *solution.ProvisionType {
	return &solution.ProvisionType{
		Doc: "an account carved from the shared message bus",
		Params: func(fs *flag.FlagSet) {
			fs.String("admin", "", "admin credential to provision with")
			fs.String("cluster", "", "cluster to hold the account")
		},
		Outputs: []solution.Output{
			{Name: "url", Doc: "endpoint of the account"},
			{Name: "config", Doc: "account configuration", Sensitive: true},
		},
		Kinds: solution.Slice | solution.Attach,
	}
}

// storeProvisionType registers a single kind and no parameters at all:
// the nil-Params dry path and the kind-word omission both prove out on
// it.
func storeProvisionType() *solution.ProvisionType {
	return &solution.ProvisionType{
		Doc:     "a verified attachment to the legacy store",
		Outputs: []solution.Output{{Name: "dsn", Doc: "connection string of the store"}},
		Kinds:   solution.Attach,
	}
}

// testCatalogue registers the well-behaved descriptors, both symbol
// types, and both provision types. Packages and elements are
// deliberately listed out of canonical order; the image must sort
// them.
func testCatalogue() []solution.Package {
	return []solution.Package{
		{
			Path:     "example.com/acme/quiet",
			Name:     "quiet",
			Elements: []solution.Element{solution.App("Quiet", quietDescriptor())},
		},
		{
			Path: "example.com/acme/substrate",
			Name: "substrate",
			Elements: []solution.Element{
				solution.Symbol("Secret", &solution.SymbolType{Doc: "an operator-held credential", Sensitive: true}),
				solution.Provision("Store", storeProvisionType()),
				solution.Symbol("Endpoint", &solution.SymbolType{Doc: "a site-bound network coordinate"}),
				solution.Provision("Bus", busProvisionType()),
			},
		},
		{
			Path: "example.com/acme/pingpong",
			Name: "pingpong",
			Elements: []solution.Element{
				solution.App("Pong", pongDescriptor()),
				solution.App("Ping", pingDescriptor()),
			},
		},
	}
}

// brokenCatalogue registers the panicking descriptor. It is separate
// because the image pins every registered package: one broken citizen
// anywhere in the catalogue aborts the compilation that registers it.
func brokenCatalogue() []solution.Package {
	return []solution.Package{{
		Path:     "example.com/acme/broken",
		Name:     "broken",
		Elements: []solution.Element{solution.App("Boom", boomDescriptor())},
	}}
}

// panickyProvisionCatalogue registers a provision type whose Params
// hook panics — the provision twin of brokenCatalogue.
func panickyProvisionCatalogue() []solution.Package {
	return []solution.Package{{
		Path: "example.com/acme/panicky",
		Name: "panicky",
		Elements: []solution.Element{solution.Provision("Grid", &solution.ProvisionType{
			Doc:    "panic on dry instantiation",
			Params: func(fs *flag.FlagSet) { panic("zap") },
			Kinds:  solution.Slice,
		})},
	}}
}

// compile runs MainCompile with buffered output streams.
func compile(t *testing.T, cfg solution.CompileConfig) (code int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	cfg.Output = &out
	cfg.Stderr = &errs
	code = solution.MainCompile(cfg)
	return code, out.String(), errs.String()
}

// ----------------------------------------------------------------------------
// Happy path

// mainUnit binds every literal kind, both symbol classes, and both
// provision kinds. The parameters are written out of order (bindings
// must sort by key), the duration literals in non-canonical spellings
// (values must canonicalize by literal kind, not flag echo — "window"
// is a flag.Func whose String() is always empty), target and subject
// bind by reference (subject through a sensitive extern, so the taint
// must surface), and Hush has no body at all. Ping2 wires provision
// outputs declared further down the unit (forward references), one of
// them sensitive; the bus slice binds an extern next to a literal; the
// legacy attachment omits its kind word, which resolves because Store
// registers exactly one kind.
const mainUnit = `solution sample

import (
	ff "example.com/acme/pingpong"
	"example.com/acme/quiet"
	sub "example.com/acme/substrate"
)

extern apiKey sub.Secret

var echoTarget: "com.acme.Echo"

deploy ff.Ping as Ping1 {
	window: 2h45m
	verbose: true
	target: echoTarget
	interval: 1500ms
	count: -1
}

deploy ff.Pong as Pong1 {
	subject: apiKey
}

deploy ff.Ping as Ping2 {
	target: bus.url
	window: bus.config
}

deploy quiet.Quiet as Hush

provision sub.Bus slice as bus {
	cluster: "nats://core"
	admin: apiKey
}

provision sub.Store as legacy
`

// goldenImage is the canonical image for mainUnit against
// testCatalogue: packages sorted by path, elements and symbols by
// name, bindings by key, outputs by name, the unreferenced Pong and
// Endpoint pinned all the same, durations rendered canonically
// (1500ms as 1.5s, 2h45m as 2h45m0s), reference bindings carrying
// refs instead of values, the extern-bound subject and admin tainted
// by their sensitive symbol type, the output-bound window tainted by
// its sensitive output, provision records carrying their kind
// explicitly — the omitted kind word resolved to attach — and the
// parameterless Store pinned without params.
const goldenImage = `{
  "format": "solution-image/1",
  "solution": "sample",
  "generation": 1,
  "catalogue": [
    {
      "path": "example.com/acme/pingpong",
      "name": "pingpong",
      "elements": [
        {
          "name": "Ping",
          "kind": "component",
          "doc": "ping a subject at a fixed interval",
          "params": [
            {
              "name": "count",
              "usage": "number of pings; -1 loops forever",
              "default": "1"
            },
            {
              "name": "interval",
              "usage": "delay between pings",
              "default": "30s"
            },
            {
              "name": "target",
              "usage": "subject to ping"
            },
            {
              "name": "verbose",
              "usage": "log every exchange",
              "default": "false",
              "boolean": true
            },
            {
              "name": "window",
              "usage": "reporting window"
            }
          ]
        },
        {
          "name": "Pong",
          "kind": "component",
          "doc": "answer pings on a subject",
          "params": [
            {
              "name": "subject",
              "usage": "subject to answer"
            }
          ]
        }
      ]
    },
    {
      "path": "example.com/acme/quiet",
      "name": "quiet",
      "elements": [
        {
          "name": "Quiet",
          "kind": "component",
          "doc": "run silently"
        }
      ]
    },
    {
      "path": "example.com/acme/substrate",
      "name": "substrate",
      "elements": [
        {
          "name": "Bus",
          "kind": "provision",
          "doc": "an account carved from the shared message bus",
          "params": [
            {
              "name": "admin",
              "usage": "admin credential to provision with"
            },
            {
              "name": "cluster",
              "usage": "cluster to hold the account"
            }
          ],
          "outputs": [
            {
              "name": "config",
              "sensitive": true
            },
            {
              "name": "url"
            }
          ],
          "kinds": [
            "slice",
            "attach"
          ]
        },
        {
          "name": "Endpoint",
          "kind": "symbol",
          "doc": "a site-bound network coordinate"
        },
        {
          "name": "Secret",
          "kind": "symbol",
          "doc": "an operator-held credential",
          "sensitive": true
        },
        {
          "name": "Store",
          "kind": "provision",
          "doc": "a verified attachment to the legacy store",
          "outputs": [
            {
              "name": "dsn"
            }
          ],
          "kinds": [
            "attach"
          ]
        }
      ]
    }
  ],
  "symbols": [
    {
      "name": "apiKey",
      "class": "extern",
      "type": {
        "package": "example.com/acme/substrate",
        "name": "Secret"
      }
    },
    {
      "name": "echoTarget",
      "class": "var",
      "value": {
        "kind": "string",
        "string": "com.acme.Echo"
      }
    }
  ],
  "records": [
    {
      "verb": "deploy",
      "element": {
        "package": "example.com/acme/pingpong",
        "name": "Ping"
      },
      "name": "Ping1",
      "params": [
        {
          "key": "count",
          "value": {
            "kind": "int",
            "int": -1
          },
          "source": "instance"
        },
        {
          "key": "interval",
          "value": {
            "kind": "duration",
            "duration": "1.5s"
          },
          "source": "instance"
        },
        {
          "key": "target",
          "ref": {
            "symbol": "echoTarget"
          },
          "source": "instance"
        },
        {
          "key": "verbose",
          "value": {
            "kind": "bool",
            "bool": true
          },
          "source": "instance"
        },
        {
          "key": "window",
          "value": {
            "kind": "duration",
            "duration": "2h45m0s"
          },
          "source": "instance"
        }
      ]
    },
    {
      "verb": "deploy",
      "element": {
        "package": "example.com/acme/pingpong",
        "name": "Pong"
      },
      "name": "Pong1",
      "params": [
        {
          "key": "subject",
          "ref": {
            "symbol": "apiKey"
          },
          "source": "instance",
          "sensitive": true
        }
      ]
    },
    {
      "verb": "deploy",
      "element": {
        "package": "example.com/acme/pingpong",
        "name": "Ping"
      },
      "name": "Ping2",
      "params": [
        {
          "key": "target",
          "ref": {
            "symbol": "bus",
            "output": "url"
          },
          "source": "instance"
        },
        {
          "key": "window",
          "ref": {
            "symbol": "bus",
            "output": "config"
          },
          "source": "instance",
          "sensitive": true
        }
      ]
    },
    {
      "verb": "deploy",
      "element": {
        "package": "example.com/acme/quiet",
        "name": "Quiet"
      },
      "name": "Hush"
    },
    {
      "verb": "provision",
      "kind": "slice",
      "element": {
        "package": "example.com/acme/substrate",
        "name": "Bus"
      },
      "name": "bus",
      "params": [
        {
          "key": "admin",
          "ref": {
            "symbol": "apiKey"
          },
          "source": "instance",
          "sensitive": true
        },
        {
          "key": "cluster",
          "value": {
            "kind": "string",
            "string": "nats://core"
          },
          "source": "instance"
        }
      ]
    },
    {
      "verb": "provision",
      "kind": "attach",
      "element": {
        "package": "example.com/acme/substrate",
        "name": "Store"
      },
      "name": "legacy"
    }
  ]
}
`

func TestMainCompileGolden(t *testing.T) {
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     []solution.Unit{{Name: "main.sdl", Source: mainUnit}},
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	if stdout != goldenImage {
		t.Errorf("image mismatch:\n--- got ---\n%s\n--- want ---\n%s", stdout, goldenImage)
	}
}

func TestMainCompileDeterminism(t *testing.T) {
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     []solution.Unit{{Name: "main.sdl", Source: mainUnit}},
		Catalogue: testCatalogue(),
	}
	_, first, _ := compile(t, cfg)
	_, second, _ := compile(t, cfg)
	if first != second {
		t.Error("two compilations of the same config produced different bytes")
	}
}

func TestMainCompileMultiUnit(t *testing.T) {
	units := []solution.Unit{
		{Name: "a.sdl", Source: "solution sample\n" +
			"import ff \"example.com/acme/pingpong\"\n" +
			"deploy ff.Pong as PongA {\n\tsubject: \"a\"\n}\n"},
		{Name: "b.sdl", Source: "solution sample\n" +
			"import (\n\tff \"example.com/acme/pingpong\"\n\t\"example.com/acme/quiet\"\n)\n" +
			"deploy quiet.Quiet as HushB\n" +
			"deploy ff.Ping as PingB\n"},
	}
	cfg := solution.CompileConfig{
		Solution:   "sample",
		Units:      units,
		Catalogue:  testCatalogue(),
		Generation: 7,
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if img.Generation != 7 {
		t.Errorf("Generation = %d, want 7", img.Generation)
	}
	var names []string
	for _, r := range img.Records {
		names = append(names, r.Name)
	}
	want := []string{"PongA", "HushB", "PingB"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("record order = %v, want %v (unit-then-statement order)", names, want)
	}

	// The same content at another generation is the same desired state.
	cfg.Generation = 8
	_, stdout8, _ := compile(t, cfg)
	img8, err := image.Decode(strings.NewReader(stdout8))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !image.Equal(img, img8) {
		t.Error("images at different generations are not Equal (provenance must be masked)")
	}
	if stdout == stdout8 {
		t.Error("different generations should still change the bytes")
	}
}

// TestMainCompileSymbolReferences proves the collect-then-bind phase
// order end to end: unit a's bindings reference a var and an extern
// that unit b declares, so resolution works across units and forward.
// The extern's sensitive symbol type must taint the binding that
// references it; the var-bound one stays plain.
func TestMainCompileSymbolReferences(t *testing.T) {
	units := []solution.Unit{
		{Name: "a.sdl", Source: "solution sample\n" +
			"import ff \"example.com/acme/pingpong\"\n" +
			"deploy ff.Ping as PingA {\n\ttarget: adminKey\n}\n" +
			"deploy ff.Pong as PongA {\n\tsubject: sharedSubject\n}\n"},
		{Name: "b.sdl", Source: "solution sample\n" +
			"import sub \"example.com/acme/substrate\"\n" +
			"extern adminKey sub.Secret\n" +
			"var sharedSubject: \"com.acme.Shared\"\n"},
	}
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     units,
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	wantSymbols := []image.SymbolDef{
		{Name: "adminKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme/substrate", Name: "Secret"}},
		{Name: "sharedSubject", Class: image.ClassVar, Value: image.String("com.acme.Shared")},
	}
	if len(img.Symbols) != len(wantSymbols) {
		t.Fatalf("Symbols = %+v, want %+v", img.Symbols, wantSymbols)
	}
	for i, want := range wantSymbols {
		got := img.Symbols[i]
		if got.Name != want.Name || got.Class != want.Class ||
			(got.Type == nil) != (want.Type == nil) || (got.Type != nil && *got.Type != *want.Type) ||
			(got.Value == nil) != (want.Value == nil) || (got.Value != nil && *got.Value != *want.Value) {
			t.Errorf("Symbols[%d] = %+v, want %+v", i, got, want)
		}
	}

	if len(img.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(img.Records))
	}
	ping := img.Records[0]
	if len(ping.Params) != 1 || ping.Params[0].Ref == nil ||
		ping.Params[0].Ref.Symbol != "adminKey" || ping.Params[0].Value != nil {
		t.Errorf("PingA target = %+v, want a bare ref to adminKey", ping.Params)
	}
	if len(ping.Params) == 1 && !ping.Params[0].Sensitive {
		t.Error("PingA target references a sensitive extern; the binding must be tainted")
	}
	pong := img.Records[1]
	if len(pong.Params) != 1 || pong.Params[0].Ref == nil ||
		pong.Params[0].Ref.Symbol != "sharedSubject" || pong.Params[0].Value != nil {
		t.Errorf("PongA subject = %+v, want a bare ref to sharedSubject", pong.Params)
	}
	if len(pong.Params) == 1 && pong.Params[0].Sensitive {
		t.Error("PongA subject references a plain var; the binding must not be tainted")
	}
}

// TestMainCompilePerFileImports proves import scope is the unit, the
// Go source-file model: two units bind the same alias to different
// packages and a third binds a second alias to a path its peer also
// imports, and every reference resolves through its own unit's table.
func TestMainCompilePerFileImports(t *testing.T) {
	units := []solution.Unit{
		{Name: "a.sdl", Source: "solution sample\n" +
			"import ff \"example.com/acme/pingpong\"\n" +
			"deploy ff.Ping as P\n"},
		{Name: "b.sdl", Source: "solution sample\n" +
			"import ff \"example.com/acme/quiet\"\n" +
			"deploy ff.Quiet as Q\n"},
		{Name: "c.sdl", Source: "solution sample\n" +
			"import pp \"example.com/acme/pingpong\"\n" +
			"deploy pp.Pong as R\n"},
	}
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     units,
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	want := []image.Ref{
		{Package: "example.com/acme/pingpong", Name: "Ping"},
		{Package: "example.com/acme/quiet", Name: "Quiet"},
		{Package: "example.com/acme/pingpong", Name: "Pong"},
	}
	if len(img.Records) != len(want) {
		t.Fatalf("got %d records, want %d", len(img.Records), len(want))
	}
	for i, ref := range want {
		if img.Records[i].Element != ref {
			t.Errorf("record %d element = %+v, want %+v (the unit's own binding of the alias)", i, img.Records[i].Element, ref)
		}
	}
}

// TestMainCompileDottedKeys drives a composite (flattened) parameter
// name end to end: a catalogue flag named with dots binds from the
// dotted SDL key like any other slot, and the binding carries the full
// key.
func TestMainCompileDottedKeys(t *testing.T) {
	catalogue := []solution.Package{{
		Path: "example.com/acme/deep",
		Name: "deep",
		Elements: []solution.Element{solution.App("Deep", &application.Descriptor{
			Name: "deep",
			Doc:  "declares a flattened composite parameter",
			Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
				fs := flag.NewFlagSet("deep", flag.ContinueOnError)
				fs.Int64("retry.max", 3, "retry budget")
				return idle(), fs
			}),
		})},
	}}
	units := []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
		"import deep \"example.com/acme/deep\"\n" +
		"deploy deep.Deep as D {\n" +
		"\tretry.max: 7\n" +
		"}\n"}}
	code, stdout, stderr := compile(t, solution.CompileConfig{
		Solution: "sample", Units: units, Catalogue: catalogue,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty (dotted names are expressible)", stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	params := img.Records[0].Params
	if len(params) != 1 || params[0].Key != "retry.max" ||
		params[0].Value == nil || params[0].Value.Int != 7 || params[0].Source != image.SourceInstance {
		t.Errorf("params = %+v, want retry.max bound to 7", params)
	}
}

// TestMainCompileInexpressibleParamWarning pins the registration-time
// warning: a catalogue flag whose name no SDL key can spell — a
// character beyond the ident-and-dot grammar, or a keyword segment —
// warns on stderr, once per flag, and never fails the compilation.
func TestMainCompileInexpressibleParamWarning(t *testing.T) {
	catalogue := []solution.Package{{
		Path: "example.com/acme/hyphen",
		Name: "hyphen",
		Elements: []solution.Element{solution.App("H", &application.Descriptor{
			Name: "hyphen",
			Doc:  "declares parameters SDL cannot spell",
			Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
				fs := flag.NewFlagSet("hyphen", flag.ContinueOnError)
				fs.String("deploy", "", "a keyword as a flag name")
				fs.String("log-level", "info", "a hyphenated flag name")
				fs.String("ok.name", "", "an expressible dotted name")
				return idle(), fs
			}),
		})},
	}}
	units := []solution.Unit{{Name: "u.sdl", Source: "solution sample\n"}}
	code, stdout, stderr := compile(t, solution.CompileConfig{
		Solution: "sample", Units: units, Catalogue: catalogue,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	want := "sdl: warning: element hyphen.H: parameter \"deploy\" is not expressible as an SDL key\n" +
		"sdl: warning: element hyphen.H: parameter \"log-level\" is not expressible as an SDL key\n"
	if stderr != want {
		t.Errorf("stderr:\n%s--- want ---\n%s", stderr, want)
	}
	if _, err := image.Decode(strings.NewReader(stdout)); err != nil {
		t.Errorf("image does not decode despite the warning: %v", err)
	}
}

// TestMainCompileDefaultMergeOrder proves the four value tiers over
// one parameter: the catalogue slot default (count is 1 in the pinned
// schema) yields no binding at all, and each SDL layer above it —
// default deploy, default ff.Ping, the instance body — wins over the
// ones below, with Source naming the winner.
func TestMainCompileDefaultMergeOrder(t *testing.T) {
	const (
		verbDefault = "default deploy {\n\tcount: 2\n}\n"
		typeDefault = "default ff.Ping {\n\tcount: 3\n}\n"
	)
	tests := []struct {
		name       string
		decls      string // between the import and the deploy
		body       string // the instance body, or ""
		wantInt    int64
		wantSource string
		wantNone   bool // count stays with the pinned schema
	}{
		{"catalogue slot only", "", "", 0, "", true},
		{"verb default", verbDefault, "", 2, image.SourceDefaultDeploy, false},
		{"type default over verb default", verbDefault + typeDefault, "", 3, image.SourceDefaultType, false},
		{"instance over both defaults", verbDefault + typeDefault, " {\n\tcount: 4\n}", 4, image.SourceInstance, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				tt.decls +
				"deploy ff.Ping as P" + tt.body + "\n"
			cfg := solution.CompileConfig{
				Solution:  "sample",
				Units:     []solution.Unit{{Name: "u.sdl", Source: source}},
				Catalogue: testCatalogue(),
			}
			code, stdout, stderr := compile(t, cfg)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
			}
			img, err := image.Decode(strings.NewReader(stdout))
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			var count *image.Binding
			for i, b := range img.Records[0].Params {
				if b.Key == "count" {
					count = &img.Records[0].Params[i]
				}
			}
			if tt.wantNone {
				if count != nil {
					t.Fatalf("count = %+v, want no binding (catalogue defaults stay in the schema)", *count)
				}
				return
			}
			if count == nil {
				t.Fatalf("no count binding; params = %+v", img.Records[0].Params)
			}
			if count.Value == nil || count.Value.Int != tt.wantInt || count.Source != tt.wantSource {
				t.Errorf("count = %+v, want value %d from %q", *count, tt.wantInt, tt.wantSource)
			}
		})
	}
}

// TestMainCompileDefaultRefs sends references through both default
// layers: the element default wires a sensitive extern (the taint must
// survive the fold), the verb default wires a var into every element
// that declares the key — and passes elements that do not declare it.
func TestMainCompileDefaultRefs(t *testing.T) {
	units := []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
		"import (\n" +
		"\tff \"example.com/acme/pingpong\"\n" +
		"\tsub \"example.com/acme/substrate\"\n" +
		")\n" +
		"extern apiKey sub.Secret\n" +
		"var subj: \"com.acme.Echo\"\n" +
		"default ff.Pong {\n" +
		"\tsubject: apiKey\n" +
		"}\n" +
		"default deploy {\n" +
		"\ttarget: subj\n" +
		"}\n" +
		"deploy ff.Pong as PongD\n" +
		"deploy ff.Ping as PingD\n"}}
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     units,
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(img.Records) != 2 {
		t.Fatalf("got %d records, want 2", len(img.Records))
	}
	pong := img.Records[0]
	if len(pong.Params) != 1 || pong.Params[0].Key != "subject" ||
		pong.Params[0].Ref == nil || pong.Params[0].Ref.Symbol != "apiKey" ||
		pong.Params[0].Source != image.SourceDefaultType || !pong.Params[0].Sensitive {
		t.Errorf("PongD params = %+v, want one sensitive default-type ref to apiKey\n(the verb default's target must pass Pong by: no such parameter)", pong.Params)
	}
	ping := img.Records[1]
	if len(ping.Params) != 1 || ping.Params[0].Key != "target" ||
		ping.Params[0].Ref == nil || ping.Params[0].Ref.Symbol != "subj" ||
		ping.Params[0].Source != image.SourceDefaultDeploy || ping.Params[0].Sensitive {
		t.Errorf("PingD params = %+v, want one plain default-deploy ref to subj", ping.Params)
	}
}

// TestMainCompileProvisionDefaults proves the verb tier folds by the
// record's own verb: default provision reaches provision records with
// its own Source and never touches deploys, the type default overrides
// it key-wise, elements that do not declare a defaulted key are passed
// by, and provision-type defaults validate against the dry Params
// schema like component defaults do.
func TestMainCompileProvisionDefaults(t *testing.T) {
	units := []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
		"import (\n" +
		"\tff \"example.com/acme/pingpong\"\n" +
		"\tsub \"example.com/acme/substrate\"\n" +
		")\n" +
		"default provision {\n" +
		"\tcluster: \"nats://default\"\n" +
		"\tadmin: \"root\"\n" +
		"}\n" +
		"default sub.Bus {\n" +
		"\tcluster: \"nats://bus\"\n" +
		"}\n" +
		"provision sub.Bus slice as bus\n" +
		"provision sub.Store as legacy\n" +
		"deploy ff.Ping as P\n"}}
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     units,
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(img.Records) != 3 {
		t.Fatalf("got %d records, want 3", len(img.Records))
	}
	bus := img.Records[0]
	if bus.Kind != image.KindSlice || len(bus.Params) != 2 ||
		bus.Params[0].Key != "admin" || bus.Params[0].Value == nil ||
		bus.Params[0].Value.Str != "root" || bus.Params[0].Source != image.SourceDefaultProvision ||
		bus.Params[1].Key != "cluster" || bus.Params[1].Value == nil ||
		bus.Params[1].Value.Str != "nats://bus" || bus.Params[1].Source != image.SourceDefaultType {
		t.Errorf("bus = %+v, want admin from %q and cluster from %q",
			bus, image.SourceDefaultProvision, image.SourceDefaultType)
	}
	if legacy := img.Records[1]; len(legacy.Params) != 0 {
		t.Errorf("legacy params = %+v, want none: Store declares neither defaulted key", legacy.Params)
	}
	if ping := img.Records[2]; len(ping.Params) != 0 {
		t.Errorf("P params = %+v, want none: default provision must not fold into deploys", ping.Params)
	}
}

// TestMainCompileCompartments proves the section compartments end to
// end: on takes literals and opaque profile tokens — a bare identifier
// never resolves against the namespace, even when it spells a declared
// symbol — metadata takes string literals, and each compartment merges
// per record across the same tiers as params (verb default under type
// default under instance), each verb reaching only its own records,
// with Source naming every binding's layer.
func TestMainCompileCompartments(t *testing.T) {
	units := []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
		"import (\n" +
		"\tff \"example.com/acme/pingpong\"\n" +
		"\tsub \"example.com/acme/substrate\"\n" +
		")\n" +
		"var awsUsEast1: 1s\n" + // a symbol spelled like the token below; the token must win
		"default deploy {\n" +
		"\ton {\n" +
		"\t\tlocation: awsUsEast1\n" +
		"\t\ttier: \"bronze\"\n" +
		"\t}\n" +
		"\tmetadata {\n" +
		"\t\towner: \"core\"\n" +
		"\t}\n" +
		"}\n" +
		"default ff.Ping {\n" +
		"\ton {\n" +
		"\t\tlocation: euWest1\n" +
		"\t}\n" +
		"}\n" +
		"deploy ff.Ping as P {\n" +
		"\ton {\n" +
		"\t\ttier: \"gold\"\n" +
		"\t\treplicas: 3\n" +
		"\t}\n" +
		"\tmetadata {\n" +
		"\t\tteam: \"search\"\n" +
		"\t}\n" +
		"}\n" +
		"deploy ff.Pong as Q\n" +
		"provision sub.Bus slice as bus {\n" +
		"\ton {\n" +
		"\t\tlocation: dcLocal\n" +
		"\t}\n" +
		"}\n"}}
	cfg := solution.CompileConfig{
		Solution:  "sample",
		Units:     units,
		Catalogue: testCatalogue(),
	}
	code, stdout, stderr := compile(t, cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(img.Records) != 3 {
		t.Fatalf("got %d records, want 3", len(img.Records))
	}

	checkBindings(t, "P.On", img.Records[0].On, []wantBinding{
		{"location", image.Token("euWest1"), image.SourceDefaultType},
		{"replicas", image.Int(3), image.SourceInstance},
		{"tier", image.String("gold"), image.SourceInstance},
	})
	checkBindings(t, "P.Metadata", img.Records[0].Metadata, []wantBinding{
		{"owner", image.String("core"), image.SourceDefaultDeploy},
		{"team", image.String("search"), image.SourceInstance},
	})
	checkBindings(t, "Q.On", img.Records[1].On, []wantBinding{
		{"location", image.Token("awsUsEast1"), image.SourceDefaultDeploy},
		{"tier", image.String("bronze"), image.SourceDefaultDeploy},
	})
	checkBindings(t, "Q.Metadata", img.Records[1].Metadata, []wantBinding{
		{"owner", image.String("core"), image.SourceDefaultDeploy},
	})
	checkBindings(t, "bus.On", img.Records[2].On, []wantBinding{
		{"location", image.Token("dcLocal"), image.SourceInstance},
	})
	if img.Records[2].Metadata != nil {
		t.Errorf("bus.Metadata = %+v, want none: default deploy must not reach provisions", img.Records[2].Metadata)
	}
}

// A wantBinding is one expected compartment binding: a literal or
// token value with its provenance.
type wantBinding struct {
	key    string
	value  *image.Value
	source string
}

// checkBindings compares one compartment against its expectation.
func checkBindings(t *testing.T, what string, got []image.Binding, want []wantBinding) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %+v, want %d bindings", what, got, len(want))
		return
	}
	for i, w := range want {
		g := got[i]
		if g.Key != w.key || g.Source != w.source || g.Ref != nil ||
			g.Value == nil || *g.Value != *w.value {
			t.Errorf("%s[%d] = %+v, want %s=%+v from %q", what, i, g, w.key, *w.value, w.source)
		}
	}
}

// ----------------------------------------------------------------------------
// Diagnostics

func TestMainCompileDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		units     []solution.Unit
		catalogue []solution.Package // nil means testCatalogue
		wantCode  int
		want      []string // exact stderr lines
	}{
		{
			name: "unknown alias",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy zz.Ping as Broken\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:8: package zz is not imported"},
		},
		{
			name: "unregistered package",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import zz \"example.com/zzz\"\n" +
				"deploy zz.Ping as Broken\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:3:8: package "example.com/zzz" (imported as zz) is not registered in the compile catalogue`},
		},
		{
			name: "unregistered package without alias",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import \"example.com/zzz\"\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:2:8: import "example.com/zzz": package not registered in the compile catalogue`},
		},
		{
			name: "unknown element",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Gong as Broken\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:3:8: unknown element Gong in package "example.com/acme/pingpong"`},
		},
		{
			name: "unknown parameter",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tnope: 1\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:4:2: unknown parameter nope: element ff.Ping has no such parameter"},
		},
		{
			name: "bad literals for typed flags",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tcount: \"many\"\n" +
				"\twindow: \"soon\"\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				"u.sdl:4:9: invalid value for parameter count: parse error",
				`u.sdl:5:10: invalid value for parameter window: time: invalid duration "soon"`,
			},
		},
		{
			name: "duplicate symbol across units",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"deploy ff.Ping as Twin\n"},
				{Name: "b.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"deploy ff.Pong as Twin\n"},
			},
			wantCode: 1,
			want:     []string{"b.sdl:3:19: duplicate symbol Twin (first declared at a.sdl:3:19)"},
		},
		{
			// The regression pin of the collect phase: names enter the
			// namespace in unit-then-statement order before anything
			// binds, so every duplicate anchors at the one owning
			// declaration, never at a nearer duplicate.
			name: "duplicate symbols anchor at the first declaration",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"deploy ff.Ping as Twin\n" +
					"deploy ff.Pong as Twin\n"},
				{Name: "b.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"deploy ff.Pong as Twin\n"},
			},
			wantCode: 1,
			want: []string{
				"a.sdl:4:19: duplicate symbol Twin (first declared at a.sdl:3:19)",
				"b.sdl:3:19: duplicate symbol Twin (first declared at a.sdl:3:19)",
			},
		},
		{
			// Import scope is the unit: rebinding a name to a second
			// path is a fault only within one unit's own import block.
			name: "conflicting import name within one unit",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import (\n" +
				"\tff \"example.com/acme/pingpong\"\n" +
				"\tff \"example.com/acme/quiet\"\n" +
				")\n" +
				"deploy ff.Ping as P\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:4:2: import name ff already bound to "example.com/acme/pingpong" (first imported at u.sdl:3:2)`},
		},
		{
			// A peer unit's import satisfies nothing: the referencing
			// unit itself must import what it names.
			name: "package imported only by a peer unit",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import (\n" +
					"\tff \"example.com/acme/pingpong\"\n" +
					"\tsub \"example.com/acme/substrate\"\n" +
					")\n" +
					"deploy ff.Ping as P\n"},
				{Name: "b.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"extern key sub.Secret\n"},
			},
			wantCode: 1,
			want:     []string{"b.sdl:3:12: package sub is not imported"},
		},
		{
			name: "solution mismatch",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution wrong\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:1:10: solution mismatch: unit declares wrong, want sample"},
		},
		{
			name: "bare type reference in an importless unit",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"deploy Ping as P\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:2:8: unqualified type reference Ping: element references must be package-qualified through an import (e.g. pkg.Ping)"},
		},
		{
			name: "provision of a component",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"provision ff.Ping slice as Q\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: Q.config\n" +
				"\ton {\n" +
				"\t\tlocation: here\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				"u.sdl:3:11: cannot provision ff.Ping: element is a component, not a provision type",
				"u.sdl:5:10: instance Q has no outputs: only provision instances emit outputs",
			},
		},
		{
			name: "unknown section",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tmount {\n" +
				"\t\tpath: \"/data\"\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:4:2: unknown section mount: sections are on and metadata"},
		},
		{
			name: "duplicate section in one body",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\ton {\n" +
				"\t\tlocation: here\n" +
				"\t}\n" +
				"\ton {\n" +
				"\t\tlocation: there\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:7:2: duplicate on section (first declared at u.sdl:4:2)"},
		},
		{
			name: "nested section inside on",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\ton {\n" +
				"\t\tinner {\n" +
				"\t\t\tlocation: here\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:5:3: on sections take parameters only, not nested sections"},
		},
		{
			name: "output reference inside on",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\ton {\n" +
				"\t\tlocation: acct.config\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:5:13: on values are literals or profile tokens, not output references"},
		},
		{
			name: "metadata value beyond a string literal",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tmetadata {\n" +
				"\t\tteam: search\n" +
				"\t\tsize: 7\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				"u.sdl:5:9: metadata values must be string literals",
				"u.sdl:6:9: metadata values must be string literals",
			},
		},
		{
			name: "unknown output",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import (\n" +
				"\tff \"example.com/acme/pingpong\"\n" +
				"\tsub \"example.com/acme/substrate\"\n" +
				")\n" +
				"provision sub.Bus slice as bus\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: bus.nope\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:8:14: unknown output nope: provision type substrate.Bus declares no such output"},
		},
		{
			name: "dotted reference to a value symbol",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"var subj: \"x\"\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: subj.out\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:5:10: symbol subj is a var, not a provision instance"},
		},
		{
			name: "provision referencing its own output",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Bus slice as bus {\n" +
				"\tadmin: bus.config\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:28: provision reference cycle: bus -> bus"},
		},
		{
			name: "two provisions cycling through outputs",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Bus slice as b1 {\n" +
				"\tadmin: b2.config\n" +
				"}\n" +
				"provision sub.Bus slice as b2 {\n" +
				"\tadmin: b1.config\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:28: provision reference cycle: b1 -> b2 -> b1"},
		},
		{
			name: "omitted kind word on a two-kind type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Bus as b\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:11: missing provision kind: type sub.Bus registers both slice and attach"},
		},
		{
			name: "kind word the type does not register",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Store slice as s\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:21: type sub.Store does not register slice"},
		},
		{
			name: "kind word that is no kind at all",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Bus wedge as w\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:19: unknown provision kind wedge: kinds are slice and attach"},
		},
		{
			name: "provision of a symbol type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Secret slice as s\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:11: cannot provision sub.Secret: element is a symbol type, not a provision type"},
		},
		{
			name: "deploy of a provision type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"deploy sub.Bus as b\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:8: cannot deploy sub.Bus: element is a provision type, not a component"},
		},
		{
			name: "extern of a provision type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"extern k sub.Bus\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:10: element sub.Bus is a provision type, not a symbol type"},
		},
		{
			name: "provision default validates eagerly",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"default sub.Bus {\n" +
				"\tnope: 1\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:4:2: unknown parameter nope: element sub.Bus has no such parameter"},
		},
		{
			name: "duplicate default for an element type",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"default ff.Ping {\n" +
					"\tcount: 5\n" +
					"}\n"},
				// The second default names the same element through its
				// own alias; identity is the resolved element, not the
				// spelling.
				{Name: "b.sdl", Source: "solution sample\n" +
					"import zz \"example.com/acme/pingpong\"\n" +
					"default zz.Ping {\n" +
					"\tcount: 7\n" +
					"}\n"},
			},
			wantCode: 1,
			want:     []string{"b.sdl:3:1: duplicate default for zz.Ping (first declared at a.sdl:3:1)"},
		},
		{
			name: "duplicate default for a verb",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"default deploy {\n" +
				"\tcount: 5\n" +
				"}\n" +
				"default deploy {\n" +
				"\tcount: 7\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:6:1: duplicate default for deploy (first declared at u.sdl:3:1)"},
		},
		{
			name: "default for an unknown element",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"default ff.Nope {\n" +
				"\tcount: 5\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:3:9: unknown element Nope in package "example.com/acme/pingpong"`},
		},
		{
			name: "default for a symbol type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"default sub.Secret {\n" +
				"\tcount: 5\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:9: cannot default sub.Secret: a symbol type takes no parameters"},
		},
		{
			// Type-scoped defaults validate eagerly: the faults surface
			// with no record of the element anywhere in the solution.
			// The on section is valid and simply never folds.
			name: "default body validates without records",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"default ff.Ping {\n" +
				"\tnope: 1\n" +
				"\tcount: \"many\"\n" +
				"\ton {\n" +
				"\t\tlocation: here\n" +
				"\t}\n" +
				"}\n" +
				"default deploy {\n" +
				"\ttarget: missing\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				"u.sdl:4:2: unknown parameter nope: element ff.Ping has no such parameter",
				`u.sdl:5:9: invalid value for parameter count: parse error`,
				"u.sdl:11:10: undefined symbol missing",
			},
		},
		{
			// A verb default validates once per element, however many
			// records fold it: one fault line for two Ping deploys.
			name: "verb default faults surface once per element",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"default deploy {\n" +
				"\tcount: \"many\"\n" +
				"}\n" +
				"deploy ff.Ping as P1\n" +
				"deploy ff.Ping as P2\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:4:9: invalid value for parameter count: parse error"},
		},
		{
			name: "undefined symbol",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: missing\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:4:10: undefined symbol missing"},
		},
		{
			name: "instance referenced as a value",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Pong as Echo\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: Echo\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:5:10: instance Echo has no value"},
		},
		{
			name: "extern with an unknown type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"extern key sub.Missing\n"}},
			wantCode: 1,
			want:     []string{`u.sdl:3:12: unknown element Missing in package "example.com/acme/substrate"`},
		},
		{
			name: "extern with a component type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"extern key ff.Ping\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:12: element ff.Ping is a component, not a symbol type"},
		},
		{
			name: "deploy of a symbol type",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"deploy sub.Secret as S\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:3:8: cannot deploy sub.Secret: element is a symbol type, not a component"},
		},
		{
			name: "duplicate symbols across classes",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"var Ping1: \"taken\"\n" +
					"deploy ff.Ping as Ping1\n"},
				{Name: "b.sdl", Source: "solution sample\n" +
					"import sub \"example.com/acme/substrate\"\n" +
					"extern token sub.Secret\n" +
					"var token: \"x\"\n"},
			},
			wantCode: 1,
			want: []string{
				"a.sdl:4:19: duplicate symbol Ping1 (first declared at a.sdl:3:5)",
				"b.sdl:4:5: duplicate symbol token (first declared at b.sdl:3:8)",
			},
		},
		{
			name: "var literal rejected by the referencing slots",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"var soon: \"whenever\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twindow: soon\n" +
				"\tinterval: soon\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				`u.sdl:5:10: invalid value for parameter window: var soon: time: invalid duration "whenever"`,
				"u.sdl:6:12: invalid value for parameter interval: var soon: parse error",
			},
		},
		{
			name: "unknown parameter bound to an extern",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import (\n" +
				"\tff \"example.com/acme/pingpong\"\n" +
				"\tsub \"example.com/acme/substrate\"\n" +
				")\n" +
				"extern apiKey sub.Secret\n" +
				"deploy ff.Ping as P {\n" +
				"\tnope: apiKey\n" +
				"}\n"}},
			wantCode: 1,
			want:     []string{"u.sdl:8:2: unknown parameter nope: element ff.Ping has no such parameter"},
		},
		{
			name: "parse errors aggregate across units",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\ndeploy ff.Ping as 7\n"},
				{Name: "b.sdl", Source: "solution sample\ndeploy ff.Ping as 7\n"},
			},
			wantCode: 1,
			want: []string{
				"a.sdl:2:19: expected identifier, found 7",
				"a.sdl:2:19: expected newline, found 7",
				"b.sdl:2:19: expected identifier, found 7",
				"b.sdl:2:19: expected newline, found 7",
			},
		},
		{
			name: "panicking Make attributed to the referencing statement",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import \"example.com/acme/broken\"\n" +
				"deploy broken.Boom as B\n"}},
			catalogue: brokenCatalogue(),
			wantCode:  2,
			want:      []string{"u.sdl:3:8: element broken.Boom: Make panicked: kaboom"},
		},
		{
			name:      "panicking Make of an unreferenced element",
			units:     []solution.Unit{{Name: "u.sdl", Source: "solution sample\n"}},
			catalogue: brokenCatalogue(),
			wantCode:  2,
			want:      []string{"compile: element broken.Boom: Make panicked: kaboom"},
		},
		{
			name: "panicking Params attributed to the referencing statement",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import \"example.com/acme/panicky\"\n" +
				"provision panicky.Grid slice as g\n"}},
			catalogue: panickyProvisionCatalogue(),
			wantCode:  2,
			want:      []string{"u.sdl:3:11: element panicky.Grid: Params panicked: zap"},
		},
		{
			name:      "panicking Params of an unreferenced element",
			units:     []solution.Unit{{Name: "u.sdl", Source: "solution sample\n"}},
			catalogue: panickyProvisionCatalogue(),
			wantCode:  2,
			want:      []string{"compile: element panicky.Grid: Params panicked: zap"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalogue := tt.catalogue
			if catalogue == nil {
				catalogue = testCatalogue()
			}
			cfg := solution.CompileConfig{
				Solution:  "sample",
				Units:     tt.units,
				Catalogue: catalogue,
			}
			code, stdout, stderr := compile(t, cfg)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty (no image on failure)", stdout)
			}
			want := strings.Join(tt.want, "\n") + "\n"
			if stderr != want {
				t.Errorf("stderr mismatch:\n--- got ---\n%s--- want ---\n%s", stderr, want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Usage errors (exit 2)

func TestMainCompileUsageErrors(t *testing.T) {
	unit := solution.Unit{Name: "u.sdl", Source: "solution sample\n"}
	pkg := func(elements ...solution.Element) []solution.Package {
		return []solution.Package{{Path: "example.com/p", Name: "p", Elements: elements}}
	}
	tests := []struct {
		name string
		cfg  solution.CompileConfig
		want string
	}{
		{
			name: "empty solution name",
			cfg:  solution.CompileConfig{Units: []solution.Unit{unit}},
			want: "compile: config: empty solution name",
		},
		{
			name: "no units",
			cfg:  solution.CompileConfig{Solution: "sample"},
			want: "compile: config: no units to compile",
		},
		{
			name: "duplicate unit names",
			cfg:  solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit, unit}},
			want: `compile: config: duplicate unit name "u.sdl"`,
		},
		{
			name: "empty package path",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: []solution.Package{{Name: "p"}}},
			want: "compile: catalogue: package 0 has an empty import path",
		},
		{
			name: "invalid package name",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: []solution.Package{{Path: "example.com/p", Name: "go-p"}}},
			want: `compile: catalogue: package "example.com/p": name "go-p" is not a valid Go identifier`,
		},
		{
			name: "package registered twice",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: append(pkg(), pkg()...)},
			want: `compile: catalogue: package "example.com/p" registered twice`,
		},
		{
			name: "nil element",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(nil)},
			want: `compile: catalogue: package "example.com/p": element 0 is nil`,
		},
		{
			name: "invalid element name",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.App("not-ident", quietDescriptor()))},
			want: `compile: catalogue: package "example.com/p": element name "not-ident" is not a valid Go identifier`,
		},
		{
			name: "duplicate element",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.App("X", quietDescriptor()), solution.App("X", quietDescriptor()))},
			want: `compile: catalogue: package "example.com/p": duplicate element X`,
		},
		{
			name: "nil descriptor",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.App("X", nil))},
			want: `compile: catalogue: package "example.com/p": element X has a nil descriptor`,
		},
		{
			name: "descriptor without Make",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.App("X", &application.Descriptor{Name: "x"}))},
			want: `compile: catalogue: package "example.com/p": element X: descriptor has no Make factory`,
		},
		{
			name: "nil symbol type",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Symbol("X", nil))},
			want: `compile: catalogue: package "example.com/p": element X has a nil symbol type`,
		},
		{
			name: "nil provision type",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Provision("X", nil))},
			want: `compile: catalogue: package "example.com/p": element X has a nil provision type`,
		},
		{
			// Kinds are the author's explicit declaration (A-11): the
			// compiler never defaults them, so forgetting both is a
			// registration fault, not a slice.
			name: "provision type without kinds",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Provision("X", &solution.ProvisionType{Doc: "kindless"}))},
			want: `compile: catalogue: package "example.com/p": element X registers no provision kinds (declare Slice, Attach, or both)`,
		},
		{
			name: "provision type with unknown kind bits",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Provision("X", &solution.ProvisionType{Kinds: 1 << 5}))},
			want: `compile: catalogue: package "example.com/p": element X registers unknown provision kinds 0b100000`,
		},
		{
			name: "provision type with a duplicate output",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Provision("X", &solution.ProvisionType{
					Kinds:   solution.Slice,
					Outputs: []solution.Output{{Name: "dsn"}, {Name: "dsn"}},
				}))},
			want: `compile: catalogue: package "example.com/p": element X declares output dsn twice`,
		},
		{
			name: "provision type with an invalid output name",
			cfg: solution.CompileConfig{Solution: "sample", Units: []solution.Unit{unit},
				Catalogue: pkg(solution.Provision("X", &solution.ProvisionType{
					Kinds:   solution.Attach,
					Outputs: []solution.Output{{Name: "not name"}},
				}))},
			want: `compile: catalogue: package "example.com/p": element X output name "not name" is not a valid Go identifier`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := compile(t, tt.cfg)
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tt.want+"\n" {
				t.Errorf("stderr = %q, want %q", stderr, tt.want+"\n")
			}
		})
	}
}
