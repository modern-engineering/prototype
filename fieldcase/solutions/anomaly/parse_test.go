// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package anomaly

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/modern-engineering/prototype/sdl/parser"
)

// TestUnitsParse feeds every unit of the solution through the SDL
// parser: stage 1's syntactic gate. Full compilation — linking against
// the catalogue and emitting the image — is stage 2's `sdl build`.
func TestUnitsParse(t *testing.T) {
	units, err := filepath.Glob("*.sdl")
	if err != nil {
		t.Fatal(err)
	}
	if want := 3; len(units) != want {
		t.Fatalf("found %d units, want %d (substrate, visibility, detection)", len(units), want)
	}
	for _, name := range units {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parser.ParseFile(name, src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if f.Solution == nil {
				t.Fatal("unit has no solution clause")
			}
			if got, want := f.Solution.Name.Name, "anomaly"; got != want {
				t.Errorf("solution clause names %q, want %q (units are peers of one solution)", got, want)
			}
		})
	}
}
