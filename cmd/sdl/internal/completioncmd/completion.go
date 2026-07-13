// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package completioncmd implements sdl completion: shell completion
// scripts generated from the live command tree — every registered
// command, subcommand group, and flag surface, walked the way
// dispatch walks them (base.Commands, then each verb's flag set) — so
// the shell's view of the CLI tracks the dispatcher instead of a
// hand-maintained copy that would drift. The emitted script is
// static: it names the commands of the binary that wrote it and never
// calls back into sdl at completion time. Completing solution
// material itself —
// instance names, catalogue elements, .sdl file contents — would need
// the shell to call back into sdl and stays a recorded door until
// static words stop being enough.
package completioncmd

import (
	"context"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// CmdCompletion is the sdl completion command.
var CmdCompletion = &base.Command{
	UsageLine: "sdl completion <bash|zsh>",
	Short:     "emit a shell completion script for the sdl command",
	Long: `Completion prints a command-completion script for the named shell to
standard output. The script is generated from the live command tree
of this very binary — every registered command, subcommand group, and
flag surface, walked the same way dispatch walks them — so completion
always matches the sdl that wrote it. The script itself is static:
it never calls back into sdl, and regenerating it after a toolchain
upgrade is what keeps new verbs known to the shell.

The shells are:

	bash    a completion script for bash (complete -F)
	zsh     a completion function in the autoload form compinit loads

Completion only writes the script; wiring it into a shell is the
user's affair.

Bash — source the script from ~/.bashrc:

	sdl completion bash > ~/.sdl-completion.bash
	echo 'source ~/.sdl-completion.bash' >> ~/.bashrc

Zsh — install the script as _sdl in a directory on $fpath and let
compinit pick it up, for example with ~/.zsh/completions on the path:

	mkdir -p ~/.zsh/completions
	sdl completion zsh > ~/.zsh/completions/_sdl

and in ~/.zshrc, before compinit runs:

	fpath=(~/.zsh/completions $fpath)
	autoload -Uz compinit && compinit

The script completes command words, subcommands, help topics, and
flags; where none of those apply the shell's own filename completion
takes over, which is what the file and directory arguments want.
Completing solution material itself — instance names, catalogue
elements — would need the shell to call back into sdl and is
deliberately left out.

Exit status 0 means the script was printed; 2 reports an unknown
shell or a malformed invocation.`,
}

func init() {
	CmdCompletion.Run = runCompletion
}

// shells maps each supported shell to its emitter. Both render the
// same walked tree; a shell is a syntax for the words, never its own
// idea of the CLI.
var shells = map[string]func(io.Writer, []spec) error{
	"bash": emitBash,
	"zsh":  emitZsh,
}

func runCompletion(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return &base.UsageError{Msg: fmt.Sprintf("completion takes exactly one shell argument, got %d", len(args))}
	}
	emit, ok := shells[args[0]]
	if !ok {
		return &base.UsageError{Msg: fmt.Sprintf("unknown completion shell %q (shells are bash and zsh)", args[0])}
	}
	return emit(s.Stdout, collect(base.Commands))
}
