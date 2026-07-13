// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package runcmd implements sdl run: the one-command dev loop from a
// solution directory to a running single-process solution. It wires
// the same front half as buildcmd — load the directory, detect the
// module context, discover the catalogue — around gen.HostSource
// instead of gen.Source, and hands the terminal to the built host
// through the work package's Exec contract. The host it generates is
// the tailored one of the two host modes: built per invocation the way
// go test builds a per-package test binary, serving exactly one
// solution and then discarded (solution/host carries the runtime; a
// prebuilt Mode-P binary like examples/host is the other mode).
package runcmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/gen"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/load"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/work"
)

// CmdRun is the sdl run command.
var CmdRun = &base.Command{
	UsageLine: "sdl run [-extern name=value]... [-grace duration] [-work] [dir]",
	Short:     "compile and run a solution in a tailored host process",
	Long: `Run compiles the solution in dir (default the current directory) and
hosts it in one process: a tailored host binary — generated and built
per invocation, the way go test builds a per-package test binary —
compiles the solution in memory, provisions its backing-service
access, deploys every instance concurrently, and serves until
interrupted or until every instance completes.

The front half is build's: the catalogue resolves exactly as the go
command would in the solution directory, and the generated host is
compiled in a temporary context mirroring that resolution. The image
never lands on disk; the host enacts it in the same process that
compiled it, against the very catalogue the compilation checked.

The -extern flag binds one extern symbol for this run, as name=value;
repeat it per symbol. Every extern the solution declares must be
bound before anything runs.

The -grace flag sets the graceful-shutdown budget granted after the
first interrupt (default 10s).

The -work flag preserves the temporary work directory and prints its
location, WORK=<dir>, to standard error.

The first SIGINT or SIGTERM starts the hosted solution's graceful
wind-down; stopping within the grace is a clean exit.

Exit status relays the hosted process's own, the solution/host
contract: 0 means every instance completed or a signal wound the run
down; 1 a wet failure — a driver refused, an instance failed, the
wind-down overran; 2 a configuration fault, including a solution that
does not compile, because under run nothing wet has happened yet.
Faults caught before the host is generated (unparseable units,
unresolvable imports) keep sdl's own codes: 1 for solution
diagnostics, 2 for usage.`,
}

var (
	flagExterns []string
	flagGrace   time.Duration
	flagWork    bool
)

func init() {
	CmdRun.Run = runRun // break init cycle: Run references CmdRun's flags
	CmdRun.Flag.Func("extern", "bind one extern symbol as `name=value` (repeatable)", func(arg string) error {
		// The host validates again; failing here spares a toolchain
		// round trip for a mistyped invocation.
		if name, _, ok := strings.Cut(arg, "="); !ok || name == "" {
			return fmt.Errorf("%q is not name=value", arg)
		}
		flagExterns = append(flagExterns, arg)
		return nil
	})
	CmdRun.Flag.DurationVar(&flagGrace, "grace", 10*time.Second, "graceful-shutdown budget after the first signal")
	CmdRun.Flag.BoolVar(&flagWork, "work", false, "print the work directory and do not delete it")
}

// runRun runs the pipeline front to back: buildcmd's shape with the
// tailored host generated in the compiler's place and the built
// program handed the terminal instead of the emitter contract.
func runRun(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
	dir := "."
	switch len(args) {
	case 0:
	case 1:
		dir = args[0]
	default:
		return &base.UsageError{Msg: fmt.Sprintf("run takes at most one directory argument, got %d", len(args))}
	}

	sol, err := load.Dir(dir)
	if err != nil {
		return err
	}
	mc, err := work.Detect(ctx, sol.Dir)
	if err != nil {
		return err
	}
	if mc.Mode() == work.ModeNone {
		return runModuleless(ctx, s, sol, mc)
	}
	pkgs, err := gen.Discover(sol.Dir, sol.Imports, mc.Env(), s.Stderr)
	if err != nil {
		return err
	}
	return work.Run(ctx, work.Config{
		Dir:     sol.Dir,
		Context: mc,
		Source:  gen.HostSource(sol, pkgs),
		Keep:    flagWork,
		Stderr:  s.Stderr,
		Exec:    hostExec(),
	})
}

// runModuleless wires the run for a solution outside any module
// context, buildcmd's inverted order: resolve the imports into the
// work module first, then discover there, then build and hand off.
func runModuleless(ctx context.Context, s base.Streams, sol *load.Solution, mc *work.Context) error {
	paths := make([]string, len(sol.Imports))
	for i, imp := range sol.Imports {
		paths[i] = imp.Path
	}
	m, err := work.ResolveModuleless(ctx, work.ModulelessConfig{
		Context: mc,
		Imports: paths,
		Keep:    flagWork,
		Stderr:  s.Stderr,
	})
	if err != nil {
		return err
	}
	defer m.Close()
	pkgs, err := gen.Discover(m.Dir, sol.Imports, m.Env(), s.Stderr)
	if err != nil {
		return err
	}
	return m.BuildAndExec(ctx, gen.HostSource(sol, pkgs), sol.Dir, hostExec())
}

// hostExec renders the hosted process's command line back out of the
// parsed flags — -work is the driver's alone — and sizes the wind-down
// backstop.
func hostExec() *work.Exec {
	args := []string{"-grace", flagGrace.String()}
	for _, e := range flagExterns {
		args = append(args, "-extern", e)
	}
	return &work.Exec{
		Args: args,
		// The backstop arms only once the invocation context is
		// cancelled: from there the child owns the grace it was
		// granted, plus a margin for provisioning audit and exit
		// bookkeeping outside the runtime's own budget.
		WaitDelay: max(flagGrace, 0) + 5*time.Second,
	}
}
