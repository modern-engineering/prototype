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
	"os"

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
catalogue packages in the enclosing Go module context, and generates a
small Go program that embeds the solution sources and links them against
the live catalogue. The program is compiled in a temporary module
mirroring the solution module's dependency resolution and then run; it
emits the image as canonical JSON.

The -o flag writes the image to a file instead of standard output.

The -generation flag stamps the image's generation (default 1).

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
func runBuild(ctx context.Context, cmd *base.Command, args []string) error {
	dir := "."
	switch len(args) {
	case 0:
	case 1:
		dir = args[0]
	default:
		return &base.UsageError{Msg: fmt.Sprintf("build takes at most one directory argument, got %d", len(args))}
	}

	sol, err := load.Dir(dir)
	if err != nil {
		return err
	}
	pkgs, err := gen.Discover(sol.Dir, sol.Imports, os.Stderr)
	if err != nil {
		return err
	}
	source := gen.Source(sol, pkgs, flagGeneration)
	return work.Run(ctx, work.Config{
		Dir:    sol.Dir,
		Source: source,
		Output: flagOutput,
		Keep:   flagWork,
		Stderr: os.Stderr,
	})
}
