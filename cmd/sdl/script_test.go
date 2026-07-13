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
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	scrubProxy()
	if sdlPath != "" {
		if err := os.RemoveAll(filepath.Dir(sdlPath)); err != nil {
			fmt.Fprintln(os.Stderr, "cleaning up the built CLI:", err)
		}
	}
	return code
}

// The script corpus is the shell-level contract: each transcript can
// assert only what a user observes. A failing cmp shows its diff;
// -update rewrites the failing golden sections in place; go test
// -testwork preserves each script's extracted work tree for a manual
// look.
func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:           filepath.Join("testdata", "script"),
		Setup:         scriptEnv,
		UpdateScripts: *update,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"status": cmdStatus,
		},
	})
}

// cmdStatus is the corpus's exit-code assertion — the taxonomy (0 ok,
// 1 diagnostics, 2 usage) is CLI contract, and the engine's ! prefix
// sees only pass/fail. Usage: status <code> <program> [args...]; the
// program's streams remain available to stdout/stderr/cmp.
func cmdStatus(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("status asserts an exact code; ! status is meaningless")
	}
	if len(args) < 2 {
		ts.Fatalf("usage: status <code> <program> [args...]")
	}
	want, err := strconv.Atoi(args[0])
	if err != nil {
		ts.Fatalf("status: %q is not an exit code", args[0])
	}
	code := 0
	if err := ts.Exec(args[1], args[2:]...); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			ts.Fatalf("status: running %s: %v", args[1], err)
		}
		code = exit.ExitCode()
	}
	if code != want {
		ts.Fatalf("%s exited %d, want %d", args[1], code, want)
	}
}

// scriptEnv hands each script the host's effective toolchain
// environment: scripts run hermetic (HOME=/no-home), which would
// otherwise send the go toolchain deriving caches under a home that
// does not exist and re-resolving every dependency per script.
// SDLREPO carries the repo root for the consumer-module shape the
// toolchain scripts lay down — a go.mod dir-replacing the prototype
// to this checkout, e2e's solutionModule as script lines.
func scriptEnv(env *testscript.Env) error {
	env.Setenv("SDLREPO", repoRoot)
	vars, err := hostGoEnv()
	if err != nil {
		return err
	}
	env.Vars = append(env.Vars, vars...)
	return nil
}

// hostGoEnv resolves the go env values worth carrying into a script,
// once per test run.
var hostGoEnv = sync.OnceValues(func() ([]string, error) {
	keys := []string{"GOPATH", "GOCACHE", "GOMODCACHE", "GOPROXY", "GOSUMDB", "GOFLAGS", "GOTOOLCHAIN"}
	out, err := exec.Command("go", append([]string{"env"}, keys...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("resolving go env: %v", err)
	}
	values := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(values) != len(keys) {
		return nil, fmt.Errorf("go env answered %d values for %d keys", len(values), len(keys))
	}
	vars := make([]string, len(keys))
	for i, key := range keys {
		vars[i] = key + "=" + values[i]
	}
	return vars, nil
})
