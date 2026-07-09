// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package gen

import (
	"errors"
	"io"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/load"
	"github.com/modern-engineering/prototype/sdl/token"
)

// TestDiscoverRejectsInternal pins the boundary diagnostic: an
// internal catalogue package is rejected up front, positioned at its
// import spec, before the toolchain is ever asked to load anything —
// the generated compiler builds as its own module and could not
// import it anyway.
func TestDiscoverRejectsInternal(t *testing.T) {
	imports := []load.Import{
		{Path: "example.com/mod/internal/cat", Pos: token.Position{Filename: "a.sdl", Line: 3, Column: 8}},
		{Path: "example.com/mod/ok", Pos: token.Position{Filename: "b.sdl", Line: 4, Column: 8}},
	}
	_, err := Discover(t.TempDir(), imports, nil, io.Discard)
	var diags *base.DiagnosticsError
	if !errors.As(err, &diags) {
		t.Fatalf("Discover error = %v (%T), want *base.DiagnosticsError", err, err)
	}
	want := `a.sdl:3:8: import "example.com/mod/internal/cat": internal package: the generated compiler builds outside the package's internal boundary and cannot import it; export the catalogue package`
	if len(diags.Lines) != 1 || diags.Lines[0] != want {
		t.Errorf("diagnostics = %q, want the one line %q", diags.Lines, want)
	}
}

// TestInternalPath drives the element rule: only a whole path element
// named internal marks the boundary.
func TestInternalPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"internal", true},
		{"internal/cat", true},
		{"example.com/mod/internal", true},
		{"example.com/mod/internal/cat", true},
		{"example.com/internals/cat", false},
		{"example.com/mod/myinternal", false},
		{"example.com/ok", false},
	}
	for _, tt := range tests {
		if got := internalPath(tt.path); got != tt.want {
			t.Errorf("internalPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
