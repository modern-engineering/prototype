// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"strings"
	"testing"
)

// TestSynthesizeGoMod drives the pure half of the module synthesis: one
// go list snapshot in, one temporary go.mod out. The fixture covers the
// main module (versionless line, dir replace at a synthetic version), a
// plain dependency, a dir-replaced dependency whose relative directory
// is rebased onto the main module root, and one whose absolute
// directory contains a space.
func TestSynthesizeGoMod(t *testing.T) {
	out := strings.Join([]string{
		"example.com/user/mod  ",
		"github.com/modern-engineering/prototype v0.3.1 ",
		"golang.org/x/sync v0.21.0 ",
		"example.com/local v0.0.0 ../local",
		"example.com/spacy v1.2.3 /abs/dir with space",
		"",
	}, "\n")

	g, err := parseModuleGraph(out, "/home/u/mod")
	if err != nil {
		t.Fatalf("parseModuleGraph: %v", err)
	}
	if g.main.path != "example.com/user/mod" || g.main.dir != "/home/u/mod" {
		t.Fatalf("main module = %+v", g.main)
	}

	got := string(synthesizeGoMod(g, "1.25.0", "go1.26.5"))
	want := `module sdl.invalid/solmain

go 1.25.0

toolchain go1.26.5

require (
	example.com/user/mod v0.0.0-solution
	github.com/modern-engineering/prototype v0.3.1
	golang.org/x/sync v0.21.0
	example.com/local v0.0.0
	example.com/spacy v1.2.3
)

replace example.com/user/mod => /home/u/mod

replace example.com/local => /home/u/local

replace example.com/spacy => "/abs/dir with space"
`
	if got != want {
		t.Errorf("synthesizeGoMod mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestParseModuleGraphFaults(t *testing.T) {
	if _, err := parseModuleGraph("", "/m"); err == nil {
		t.Error("empty graph: want error, got nil")
	}
	if _, err := parseModuleGraph("a\nb\n", "/m"); err == nil {
		t.Error("two main modules: want error, got nil")
	}
}
