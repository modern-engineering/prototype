// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The script-shaped half of the e2e proof: each file under
// testdata/script is one txtar transcript of an sdl session — argv,
// exit verdicts, both streams — with the fixture tree in file
// sections right beside the assertions that read them, so a
// positioned diagnostic is checkable against its source by eye. The
// testscript engine re-executes this very test binary as the sdl
// command, no build step in between. The engine is a dependency by
// ruling, with a recorded reopening trigger: should go-internal chafe
// at prototype pace, fall back to a bespoke txtar runner — the
// scripts themselves are engine-portable text.
package main_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	sdl "github.com/modern-engineering/prototype/cmd/sdl"
)

func TestMain(m *testing.M) {
	// A re-execution of this binary as sdl diverts into the real
	// dispatch before any test machinery runs; the top-level go test
	// execution falls through into the tests, with the Go-shaped e2e
	// suite's artifact built around them.
	testscript.Main(e2eM{m}, map[string]func(){"sdl": sdl.Main})
}

// e2eM wraps the test run in the Go-shaped e2e suite's artifact
// lifecycle — locate the repo, build the CLI once, clean it up —
// under testscript.Main, which owns the process exit.
type e2eM struct{ m *testing.M }

func (w e2eM) Run() int {
	flag.Parse()
	var err error
	if repoRoot, err = locateRepo(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !testing.Short() {
		if err := buildCLI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	code := w.m.Run()
	if sdlPath != "" {
		if err := os.RemoveAll(filepath.Dir(sdlPath)); err != nil {
			fmt.Fprintln(os.Stderr, "cleaning up the built CLI:", err)
		}
	}
	return code
}

// TestScript runs the script corpus. A failing cmp shows its diff;
// -update rewrites the failing golden sections in place; go test
// -testwork preserves each script's extracted work tree for a manual
// look.
func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:           filepath.Join("testdata", "script"),
		UpdateScripts: *update,
	})
}
