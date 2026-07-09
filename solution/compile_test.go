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

// testCatalogue registers the well-behaved descriptors. Packages and
// elements are deliberately listed out of canonical order; the image
// must sort them.
func testCatalogue() []solution.Package {
	return []solution.Package{
		{
			Path:     "example.com/acme/quiet",
			Name:     "quiet",
			Elements: []solution.Element{solution.App("Quiet", quietDescriptor())},
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

// mainUnit binds every literal kind. The parameters are written out of
// order (bindings must sort by key), the duration literals in
// non-canonical spellings (values must canonicalize by literal kind, not
// flag echo — "window" is a flag.Func whose String() is always empty),
// and the second deploy has no body at all.
const mainUnit = `solution sample

import (
	ff "example.com/acme/pingpong"
	"example.com/acme/quiet"
)

deploy ff.Ping as Ping1 {
	window: 2h45m
	verbose: true
	target: "com.acme.Echo"
	interval: 1500ms
	count: -1
}

deploy quiet.Quiet as Hush
`

// goldenImage is the canonical image for mainUnit against
// testCatalogue: packages sorted by path, elements by name, bindings by
// key, the unreferenced Pong pinned all the same, and durations
// rendered canonically (1500ms as 1.5s, 2h45m as 2h45m0s).
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
          "value": {
            "kind": "string",
            "string": "com.acme.Echo"
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
        "package": "example.com/acme/quiet",
        "name": "Quiet"
      },
      "name": "Hush"
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
			name: "conflicting import name",
			units: []solution.Unit{
				{Name: "a.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/pingpong\"\n" +
					"deploy ff.Ping as P\n"},
				// The deploy resolves through the first binding (ff.Ping
				// exists in pingpong), so the conflict is the only fault.
				{Name: "b.sdl", Source: "solution sample\n" +
					"import ff \"example.com/acme/quiet\"\n" +
					"deploy ff.Ping as Q\n"},
			},
			wantCode: 1,
			want:     []string{`b.sdl:2:8: import name ff already bound to "example.com/acme/pingpong" (first imported at a.sdl:2:8)`},
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
			name: "language beyond this rung",
			units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"extern natsAdmin ff.Secret\n" +
				"var greeting: \"hi\"\n" +
				"default ff.Ping {\n" +
				"\tcount: 5\n" +
				"}\n" +
				"provision ff.Ping slice as Q\n" +
				"deploy ff.Ping as P {\n" +
				"\ttarget: greeting\n" +
				"\ton {\n" +
				"\t\tlocation: here\n" +
				"\t}\n" +
				"}\n"}},
			wantCode: 1,
			want: []string{
				"u.sdl:3:1: extern declarations not yet supported by this compiler rung",
				"u.sdl:4:1: var declarations not yet supported by this compiler rung",
				"u.sdl:5:1: default declarations not yet supported by this compiler rung",
				"u.sdl:8:1: provision declarations not yet supported by this compiler rung",
				"u.sdl:10:10: symbol reference greeting not yet supported by this compiler rung",
				"u.sdl:11:2: on sections not yet supported by this compiler rung",
			},
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
