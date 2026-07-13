// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/modern-engineering/prototype/solution"
)

// MainTailored is the Mode-T bridge: the whole main body of a
// tailored host — the program a dev-loop driver generates and builds
// per invocation around an embedded solution, the way go test builds
// a per-package test binary (A-13's one-command loop). It compiles
// the solution in memory with [solution.Compile] — diagnostics print
// positioned to cc.Stderr exactly as `sdl build`'s exit-1 contract
// prints them — and hosts the resulting image through [Main] against
// the same catalogue the compilation linked. args is the child's
// command line after the program name:
//
//	-extern name=value   bind one extern symbol (repeatable)
//	-grace duration      graceful-shutdown budget (default 10s)
//
// A solution that does not compile exits 2, not build's 1: under run
// semantics the failure sits in the run's inputs, and nothing wet ran.
func MainTailored(cc solution.CompileConfig, args []string) int {
	externs := make(map[string]string)
	fs := flag.NewFlagSet(cc.Solution, flag.ContinueOnError)
	fs.Func("extern", "bind one extern symbol as `name=value` (repeatable)", func(arg string) error {
		name, value, ok := strings.Cut(arg, "=")
		if !ok || name == "" {
			return fmt.Errorf("%q is not name=value", arg)
		}
		externs[name] = value
		return nil
	})
	grace := fs.Duration("grace", 10*time.Second, "graceful-shutdown budget after the first signal")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	img, err := solution.Compile(cc)
	if err != nil {
		return 2
	}
	return Main(Config{
		Image:     img,
		Catalogue: cc.Catalogue,
		Externs:   externs,
		Grace:     *grace,
	})
}
