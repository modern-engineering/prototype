// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Sdl is the solution definition language toolchain.
//
// Usage:
//
//	sdl <command> [arguments]
//
// The commands are:
//
//	build       compile a solution directory into its desired-state image
//
// Use "sdl help <command>" for more information about a command.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/buildcmd"
)

func init() {
	// Assemble the command tree here, cmd/go style: the verb packages
	// declare their commands, main owns the list.
	base.Commands = []*base.Command{
		buildcmd.CmdBuild,
	}
}

func main() {
	os.Exit(invoke(context.Background(), os.Args[1:]))
}

// invoke dispatches one sdl invocation and returns the process exit
// code, translating the dispatched command's error per the base package
// contract: nil is 0, a DiagnosticsError prints its lines and is 1,
// everything else — usage faults and internal failures alike — is 2.
func invoke(ctx context.Context, args []string) int {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return 2
	}
	if args[0] == "help" {
		return help(args[1:])
	}
	cmd := base.Lookup(args[0])
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "sdl %s: unknown command\nRun 'sdl help' for usage.\n", args[0])
		return 2
	}

	cmd.Flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s\n", cmd.UsageLine)
		fmt.Fprintf(os.Stderr, "Run 'sdl help %s' for details.\n", cmd.Name())
	}
	if err := cmd.Flag.Parse(args[1:]); err != nil {
		// The flag package already printed the fault and the usage
		// line; -h lands here too, as in cmd/go.
		return 2
	}

	err := cmd.Run(ctx, cmd, cmd.Flag.Args())
	if err == nil {
		return 0
	}
	var diags *base.DiagnosticsError
	if errors.As(err, &diags) {
		for _, line := range diags.Lines {
			fmt.Fprintln(os.Stderr, line)
		}
		return 1
	}
	var usage *base.UsageError
	if errors.As(err, &usage) {
		fmt.Fprintf(os.Stderr, "sdl: %s\n", usage.Msg)
		fmt.Fprintf(os.Stderr, "Run 'sdl help %s' for usage.\n", cmd.Name())
		return 2
	}
	fmt.Fprintf(os.Stderr, "sdl: %v\n", err)
	return 2
}

// help implements 'sdl help [command]'.
func help(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return 0
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "usage: sdl help [command]")
		return 2
	}
	cmd := base.Lookup(args[0])
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "sdl help %s: unknown help topic\nRun 'sdl help' for usage.\n", args[0])
		return 2
	}
	fmt.Printf("usage: %s\n\n%s\n", cmd.UsageLine, cmd.Long)
	return 0
}

// printUsage renders the top-level command list. The writes are best
// effort: usage goes to a standard stream that has nowhere better to
// hear about its own failure, so the write errors are deliberately
// discarded.
func printUsage(w io.Writer) {
	printf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, format, args...)
	}
	printf("Sdl is the solution definition language toolchain.\n\n")
	printf("Usage:\n\n\tsdl <command> [arguments]\n\n")
	printf("The commands are:\n\n")
	for _, cmd := range base.Commands {
		printf("\t%-11s %s\n", cmd.Name(), cmd.Short)
	}
	printf("\nUse \"sdl help <command>\" for more information about a command.\n")
}
