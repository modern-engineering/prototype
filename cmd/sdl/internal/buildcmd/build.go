// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package buildcmd implements sdl build. It only wires the pipeline —
// load the solution directory, discover the catalogue and generate the
// compiler, drive the toolchain — keeping the verb layer as thin as
// cmd/go's, so the next verbs (echo, fmt) are new packages of the same
// shape, not surgery on this one.
package buildcmd

import (
	"context"
	"fmt"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/gen"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/load"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/work"
)

// CmdBuild is the sdl build command.
var CmdBuild = &base.Command{
	UsageLine: "sdl build [-o output] [-generation N] [-work] [dir]",
	Short:     "compile a solution directory into its desired-state image",
	Long: `Build compiles the solution in dir (default the current directory)
into its desired-state image.

Build reads every .sdl file of the directory, resolves the imported
catalogue packages exactly as the go command would in that directory —
through the enclosing module's requirements and replaces, through an
active workspace's union of modules, or, outside any module context,
at their latest versions via the ambient GOPROXY configuration — and
generates a small Go program that embeds the solution sources and links
them against the live catalogue. The program is compiled in a temporary
context mirroring that resolution and then run; it emits the image as
canonical JSON.

The -o flag writes the image to a file instead of standard output.

The -generation flag stamps the image's generation (default 1); the
generation must be positive.

The -work flag preserves the temporary work directory and prints its
location, WORK=<dir>, to standard error.

Exit status 0 means the image was emitted; 1 reports diagnostics in the
solution itself; 2 reports usage errors or failures of the toolchain.`,
}

var (
	flagOutput     string
	flagGeneration int64
	flagWork       bool
)

func init() {
	CmdBuild.Run = runBuild // break init cycle: Run references CmdBuild's flags
	CmdBuild.Flag.StringVar(&flagOutput, "o", "", "write the image to `file` instead of stdout")
	CmdBuild.Flag.Int64Var(&flagGeneration, "generation", 1, "stamp the image with generation `N`")
	CmdBuild.Flag.BoolVar(&flagWork, "work", false, "print the work directory and do not delete it")
}

// runBuild runs the pipeline front to back. Every stage returns errors
// already classified for main's exit-code translation.
func runBuild(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
	dir := "."
	switch len(args) {
	case 0:
	case 1:
		dir = args[0]
	default:
		return &base.UsageError{Msg: fmt.Sprintf("build takes at most one directory argument, got %d", len(args))}
	}
	if flagGeneration < 1 {
		// The counter is monotonic from 1 (the image edit precedent);
		// stamping zero or less is a mistyped invocation, not a build.
		return &base.UsageError{Msg: "build: generation must be positive"}
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
		return buildModuleless(ctx, s, sol, mc)
	}
	pkgs, err := gen.Discover(sol.Dir, sol.Imports, mc.Env(), s.Stderr)
	if err != nil {
		return err
	}
	source := gen.Source(sol, pkgs, flagGeneration)
	return work.Run(ctx, work.Config{
		Dir:     sol.Dir,
		Context: mc,
		Source:  source,
		Output:  flagOutput,
		Keep:    flagWork,
		Stderr:  s.Stderr,
	})
}

// buildModuleless wires the pipeline for a solution outside any module
// context, where the usual order inverts: the driver's first half
// resolves the imports into the work module before discovery has
// anywhere to root, and its second half compiles the program discovery
// made generable.
func buildModuleless(ctx context.Context, s base.Streams, sol *load.Solution, mc *work.Context) error {
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
	return m.BuildAndRun(ctx, gen.Source(sol, pkgs, flagGeneration), flagOutput)
}
