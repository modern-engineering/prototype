// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Sdl; see doc.go for the command overview, the compilation
// architecture, and the phase glossary.

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/buildcmd"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/echocmd"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/fmtcmd"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/imagecmd"
)

func init() {
	// Assemble the command tree here, cmd/go style: the verb packages
	// declare their commands, main owns the list.
	base.Commands = []*base.Command{
		buildcmd.CmdBuild,
		echocmd.CmdEcho,
		fmtcmd.CmdFmt,
		imagecmd.CmdImage,
	}
}

func main() {
	// An interrupt cancels the invocation context rather than killing
	// the process outright: context-aware children (the go toolchain,
	// the generated compiler) die with it, the command unwinds through
	// its defers, and base.Exit drains whatever cleanups — work
	// directories, notably — the run registered along the way.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := invoke(ctx, os.Args[1:])
	stop()
	base.Exit(code)
}

// invoke dispatches one sdl invocation and returns the process exit
// code, translating the dispatched command's error per the base package
// contract: nil is 0, a DiagnosticsError prints its lines and is 1, a
// RelayedExit is a child's code passed through verbatim, everything
// else — usage faults and internal failures alike — is 2. Command
// groups dispatch one name at a time, cmd/go's BigCmdLoop.
func invoke(ctx context.Context, args []string) int {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return 2
	}
	if args[0] == "help" {
		return help(args[1:])
	}
	cmds, path := base.Commands, "sdl"
	for {
		cmd := base.Lookup(cmds, args[0])
		if cmd == nil {
			fmt.Fprintf(os.Stderr, "%s %s: unknown command\nRun 'sdl help' for usage.\n", path, args[0])
			return 2
		}
		if len(cmd.Commands) > 0 {
			path += " " + args[0]
			args = args[1:]
			if len(args) == 0 {
				fmt.Fprintf(os.Stderr, "usage: %s\n", cmd.UsageLine)
				fmt.Fprintf(os.Stderr, "Run 'sdl help %s' for details.\n", cmd.LongName())
				return 2
			}
			cmds = cmd.Commands
			continue
		}
		return run(ctx, cmd, args[1:])
	}
}

// run parses one runnable command's flags and translates its error
// into the exit code.
func run(ctx context.Context, cmd *base.Command, args []string) int {
	cmd.Flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s\n", cmd.UsageLine)
		fmt.Fprintf(os.Stderr, "Run 'sdl help %s' for details.\n", cmd.LongName())
	}
	if err := cmd.Flag.Parse(args); err != nil {
		// The flag package already printed the fault and the usage
		// line; -h lands here too, as in cmd/go.
		return 2
	}

	err := cmd.Run(ctx, cmd, cmd.Flag.Args())
	if err == nil {
		return 0
	}
	var relay *base.RelayedExit
	if errors.As(err, &relay) {
		// The child owned the streams and has already reported; its
		// code carries its own contract through unchanged.
		return relay.Code
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
		fmt.Fprintf(os.Stderr, "Run 'sdl help %s' for usage.\n", cmd.LongName())
		return 2
	}
	fmt.Fprintf(os.Stderr, "sdl: %v\n", err)
	return 2
}

// help implements 'sdl help [command...]', walking command groups the
// same way dispatch does.
func help(args []string) int {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return 0
	}
	cmds := base.Commands
	var cmd *base.Command
	for i, name := range args {
		cmd = base.Lookup(cmds, name)
		if cmd == nil {
			fmt.Fprintf(os.Stderr, "sdl help %s: unknown help topic\nRun 'sdl help' for usage.\n", strings.Join(args[:i+1], " "))
			return 2
		}
		cmds = cmd.Commands
	}
	fmt.Printf("usage: %s\n\n%s\n", cmd.UsageLine, cmd.Long)
	if len(cmd.Commands) > 0 {
		fmt.Printf("\nThe commands are:\n\n")
		for _, sub := range cmd.Commands {
			fmt.Printf("\t%-11s %s\n", sub.Name(), sub.Short)
		}
		fmt.Printf("\nUse \"sdl help %s <command>\" for more information about a command.\n", cmd.LongName())
	}
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
