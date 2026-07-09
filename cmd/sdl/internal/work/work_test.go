// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"bytes"
	"strings"
	"testing"
)

// TestSynthesizeGoMod drives the pure half of the module synthesis: one
// go list snapshot in, one temporary go.mod out. The graph lines use the
// listModuleGraph template format: "path version", the main module's
// version empty, replacements behind "=> mod rpath rversion" or
// "=> dir directory".
//
// A module replacement renders the same graph line whether or not the
// module cache already holds the replacement — the template asks for the
// module identity, never Replace.Dir — so the cached and uncached shapes
// are one case here by construction, and neither can degrade to a
// cache-directory replace or be dropped.
func TestSynthesizeGoMod(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		mainDir string
		want    string
	}{
		{
			// One graph exercising every dependency shape: a plain
			// require, a dir-replaced dependency whose relative
			// directory is rebased onto the main module root, one whose
			// absolute directory contains a space, and a
			// module-to-module replacement mirrored as itself.
			name: "full graph",
			lines: []string{
				"example.com/user/mod ",
				"github.com/modern-engineering/prototype v0.3.1",
				"golang.org/x/sync v0.21.0",
				"example.com/local v0.0.0 => dir ../local",
				"example.com/spacy v1.2.3 => dir /abs/dir with space",
				"example.com/old v1.0.0 => mod example.com/new v2.0.0",
				"",
			},
			mainDir: "/home/u/mod",
			want: `module sdl.invalid/solmain

go 1.25.0

toolchain go1.26.5

require (
	example.com/user/mod v0.0.0-solution
	github.com/modern-engineering/prototype v0.3.1
	golang.org/x/sync v0.21.0
	example.com/local v0.0.0
	example.com/spacy v1.2.3
	example.com/old v1.0.0
)

replace example.com/user/mod => /home/u/mod

replace example.com/local => /home/u/local

replace example.com/spacy => "/abs/dir with space"

replace example.com/old => example.com/new v2.0.0
`,
		},
		{
			// Building inside the prototype repository itself: the
			// main-module replace is the only one needed.
			name: "prototype as main module",
			lines: []string{
				"github.com/modern-engineering/prototype ",
				"golang.org/x/sync v0.21.0",
				"",
			},
			mainDir: "/home/u/prototype",
			want: `module sdl.invalid/solmain

go 1.25.0

toolchain go1.26.5

require (
	github.com/modern-engineering/prototype v0.0.0-solution
	golang.org/x/sync v0.21.0
)

replace github.com/modern-engineering/prototype => /home/u/prototype
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := parseModuleGraph(strings.Join(tt.lines, "\n"), tt.mainDir)
			if err != nil {
				t.Fatalf("parseModuleGraph: %v", err)
			}
			got := string(synthesizeGoMod(g, "1.25.0", "go1.26.5"))
			if got != tt.want {
				t.Errorf("synthesizeGoMod mismatch\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}

// TestParseModuleGraphReplacements pins the parsed shape of the two
// replacement kinds.
func TestParseModuleGraphReplacements(t *testing.T) {
	out := "m \n" +
		"example.com/old v1.0.0 => mod example.com/new v2.0.0\n" +
		"example.com/local v0.0.0 => dir sub/dir with space\n"
	g, err := parseModuleGraph(out, "/root")
	if err != nil {
		t.Fatalf("parseModuleGraph: %v", err)
	}
	if len(g.deps) != 2 {
		t.Fatalf("deps: got %d, want 2", len(g.deps))
	}
	mod := g.deps[0]
	if mod.path != "example.com/old" || mod.version != "v1.0.0" ||
		mod.replPath != "example.com/new" || mod.replVersion != "v2.0.0" || mod.dir != "" {
		t.Errorf("module replacement parsed as %+v", mod)
	}
	dir := g.deps[1]
	if dir.path != "example.com/local" || dir.version != "v0.0.0" ||
		dir.replPath != "" || dir.replVersion != "" || dir.dir != "/root/sub/dir with space" {
		t.Errorf("directory replacement parsed as %+v", dir)
	}
}

func TestParseModuleGraphFaults(t *testing.T) {
	faults := []string{
		"",                        // no main module
		"a\nb\n",                  // two main modules
		"a\nb v1 => c\n",          // replacement without a discriminator
		"a\nb v1 => mod c\n",      // module replacement missing its version
		"a\nb v1 => zap c v2.0\n", // unknown discriminator
	}
	for _, out := range faults {
		if _, err := parseModuleGraph(out, "/m"); err == nil {
			t.Errorf("parseModuleGraph(%q): want error, got nil", out)
		}
	}
}

// TestSynthesizeGoWork drives the pure half of the workspace synthesis:
// one parsed go.work state in, one temporary go.work out — the user's
// directories and replaces mirrored, the work directory joining as a
// member.
func TestSynthesizeGoWork(t *testing.T) {
	tests := []struct {
		name string
		ws   *workspace
		want string
	}{
		{
			// Every directive shape at once: language directives, a
			// use'd directory with a space, both replacement kinds, and
			// a versioned replace target.
			name: "full workspace",
			ws: &workspace{
				goVersion: "1.25.0",
				toolchain: "go1.26.5",
				dirs:      []string{"/home/u/modA", "/home/u/dir with space"},
				replaces: []replacement{
					{oldPath: "example.com/old", oldVersion: "v1.0.0", newPath: "example.com/new", newVersion: "v2.0.0"},
					{oldPath: "example.com/local", newPath: "/home/u/local"},
				},
			},
			want: `go 1.25.0

toolchain go1.26.5

use (
	/home/u/modA
	"/home/u/dir with space"
	.
)

replace example.com/old v1.0.0 => example.com/new v2.0.0

replace example.com/local => /home/u/local
`,
		},
		{
			name: "bare workspace",
			ws:   &workspace{dirs: []string{"/w/m"}},
			want: "\nuse (\n\t/w/m\n\t.\n)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(synthesizeGoWork(tt.ws)); got != tt.want {
				t.Errorf("synthesizeGoWork mismatch\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}

func TestScaffoldGoMod(t *testing.T) {
	if got, want := string(scaffoldGoMod("1.25.0")), "module sdl.invalid/solmain\n\ngo 1.25.0\n"; got != want {
		t.Errorf("scaffoldGoMod(1.25.0) = %q, want %q", got, want)
	}
	if got, want := string(scaffoldGoMod("")), "module sdl.invalid/solmain\n"; got != want {
		t.Errorf("scaffoldGoMod(\"\") = %q, want %q", got, want)
	}
}

// TestComparePrototype pins the workspace-facing row shapes of the skew
// handshake: a version-less row is a main or workspace module — local
// source, never compared — while ordinary and replaced versions warn
// exactly as they do in module mode.
func TestComparePrototype(t *testing.T) {
	tests := []struct {
		name     string
		m        module
		wantWarn bool
	}{
		{name: "workspace member", m: module{path: prototypePath}},
		{name: "matching version", m: module{path: prototypePath, version: "v0.4.0"}},
		{name: "other version", m: module{path: prototypePath, version: "v0.3.1"}, wantWarn: true},
		{name: "dir replace", m: module{path: prototypePath, version: "v0.3.1", dir: "/src/proto"}},
		{name: "mod replace at another version", m: module{path: prototypePath, version: "v0.4.0", replPath: "example.com/fork", replVersion: "v0.5.0"}, wantWarn: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			comparePrototype(tt.m, "v0.4.0", &stderr)
			if warned := strings.Contains(stderr.String(), "warning"); warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v\nstderr: %s", warned, tt.wantWarn, stderr.String())
			}
		})
	}
}

// TestCheckPrototypeVersion drives the skew handshake over every graph
// shape the prototype module can resolve through. Only a version the
// build will genuinely use is compared: a directory replace pins no
// meaningful version (the everyday dev-loop shape), so it never warns.
func TestCheckPrototypeVersion(t *testing.T) {
	const proto = prototypePath
	tests := []struct {
		name       string
		g          *moduleGraph
		cliVersion string
		wantWarn   bool
		wantFatal  bool
	}{
		{
			name:       "prototype is the main module",
			g:          &moduleGraph{main: module{path: proto, dir: "/src/proto"}},
			cliVersion: "v0.9.9",
		},
		{
			name: "plain dependency at the CLI's own version",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.3.1"}},
			},
			cliVersion: "v0.3.1",
		},
		{
			name: "plain dependency at another version",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.3.1"}},
			},
			cliVersion: "v0.4.0",
			wantWarn:   true,
		},
		{
			name: "directory replace never warns",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.0.0", dir: "/src/proto"}},
			},
			cliVersion: "(devel)",
		},
		{
			name: "module replace compares the replacement version",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.3.1", replPath: "example.com/fork", replVersion: "v0.4.0"}},
			},
			cliVersion: "v0.4.0",
		},
		{
			name: "module replace at another version warns",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.3.1", replPath: "example.com/fork", replVersion: "v0.4.0"}},
			},
			cliVersion: "v0.3.1",
			wantWarn:   true,
		},
		{
			name: "unknown CLI version never warns",
			g: &moduleGraph{
				main: module{path: "example.com/m", dir: "/m"},
				deps: []module{{path: proto, version: "v0.3.1"}},
			},
			cliVersion: "",
		},
		{
			name:      "prototype missing from the graph",
			g:         &moduleGraph{main: module{path: "example.com/m", dir: "/m"}},
			wantFatal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			diag := checkPrototypeVersion(tt.g, "/m/go.mod", tt.cliVersion, &stderr)
			if (diag != nil) != tt.wantFatal {
				t.Errorf("diagnostic = %v, want fatal %v", diag, tt.wantFatal)
			}
			warned := strings.Contains(stderr.String(), "warning")
			if warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v\nstderr: %s", warned, tt.wantWarn, stderr.String())
			}
		})
	}
}
