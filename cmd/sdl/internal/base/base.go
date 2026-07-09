// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package base defines the command dispatch foundation shared by the
// subcommands of sdl, after the shape of cmd/go's internal/base: each
// verb package declares a [Command] and wires its Run and flags in its
// own init, and package main only assembles the [Commands] list and
// translates returned errors into the process exit code.
//
// # Exit codes
//
// The sdl command distinguishes faults in the solution's own material
// from faults in the invocation or in sdl itself:
//
//	0  success
//	1  solution diagnostics (a [DiagnosticsError]): syntax errors,
//	   unresolvable imports, failed checks — anything the solution
//	   author fixes by editing the solution
//	2  everything else: usage errors (a [UsageError] or a flag parsing
//	   failure) and internal or environmental failures
//
// Only package main maps errors to codes; commands return errors and
// never call os.Exit.
package base

import (
	"context"
	"flag"
	"strings"
)

// A Command is an implementation of an sdl command like sdl build, or
// a command group like sdl image that only dispatches to the
// subcommands it carries (cmd/go's precedent: go mod, go tool).
type Command struct {
	// Run runs the command. The args are the arguments after the
	// command name, with the command's flags already parsed out.
	//
	// The returned error selects the exit code (see the package
	// documentation); nil means success. Run is nil on a command
	// group.
	Run func(ctx context.Context, cmd *Command, args []string) error

	// UsageLine is the one-line usage message, opening with the full
	// command path ("sdl image edit"); [Command.LongName] and
	// [Command.Name] derive from it.
	UsageLine string

	// Short is the short description shown in the 'sdl help' output.
	Short string

	// Long is the long message shown in the 'sdl help <this-command>'
	// output.
	Long string

	// Commands are the subcommands of a command group, in help order;
	// empty for a runnable command. Main dispatches through the group
	// one name at a time.
	Commands []*Command

	// Flag is a set of flags specific to this command. The zero value
	// is ready for the command's init to populate; main wires Usage and
	// parses it before calling Run.
	Flag flag.FlagSet
}

// LongName returns the command's long name: the usage line's command
// path with the leading "sdl" dropped ("image edit").
func (c *Command) LongName() string {
	name := c.UsageLine
	if i := strings.Index(name, " ["); i >= 0 {
		name = name[:i]
	}
	if i := strings.Index(name, " <"); i >= 0 {
		name = name[:i]
	}
	return strings.TrimPrefix(name, "sdl ")
}

// Name returns the command's name: the last word of the long name.
func (c *Command) Name() string {
	name := c.LongName()
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// Runnable reports whether the command runs; a command group only
// dispatches.
func (c *Command) Runnable() bool { return c.Run != nil }

// Commands lists the top-level commands. The order here is the order
// in which they are printed by 'sdl help'. Package main populates the
// list; keeping the assembly there avoids initialization cycles
// between the verb packages and base.
var Commands []*Command

// Lookup returns the command of cmds with the given name, or nil.
// Dispatch through a command group looks up one level at a time.
func Lookup(cmds []*Command, name string) *Command {
	for _, cmd := range cmds {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

// A DiagnosticsError reports solution diagnostics: faults in the
// solution's own material rather than in the invocation or in sdl
// itself. Package main prints its Lines, one per line, and exits 1.
//
// Diagnostics reach the user through two routes. Faults detected in
// process carry their positioned "file:line:col: message" lines here for
// main to print; the generated solution compiler writes its diagnostics
// to the inherited stderr itself, so the driver relays them by returning
// a DiagnosticsError with no lines at all.
type DiagnosticsError struct {
	Lines []string // positioned diagnostic lines still to print; may be empty
}

// Error returns a one-line summary; the individual diagnostics live in
// Lines.
func (e *DiagnosticsError) Error() string {
	switch len(e.Lines) {
	case 0:
		return "solution diagnostics"
	case 1:
		return e.Lines[0]
	}
	return e.Lines[0] + " (and more diagnostics)"
}

// A UsageError reports a malformed invocation. Package main prints the
// message together with a pointer at 'sdl help' and exits 2.
type UsageError struct {
	Msg string
}

// Error returns the usage message.
func (e *UsageError) Error() string { return e.Msg }
