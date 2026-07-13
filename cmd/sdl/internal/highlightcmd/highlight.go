// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package highlightcmd implements sdl highlight: editor syntax
// definitions for solution units, generated from the grammar code —
// sdl/token's vocabulary, the linker's body vocabulary
// (solution.Vocabulary), and the scanner's lexical shapes — so an
// editor's view of the language tracks the toolchain instead of a
// hand-maintained copy that would drift. The verb only writes to
// standard output; installing the output in an editor is
// deliberately the user's affair, and the help carries the wiring.
package highlightcmd

import (
	"context"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// CmdHighlight is the sdl highlight command.
var CmdHighlight = &base.Command{
	UsageLine: "sdl highlight <vim|vscode>",
	Short:     "emit an editor syntax definition for solution units",
	Long: `Highlight prints a syntax-highlighting definition for solution units
to standard output. The definition is generated from the grammar code
itself — the keyword, punctuation, and literal vocabulary of the
token package, and the section and top-level-field vocabulary the
linker checks statement bodies against — never maintained by hand, so
the emitted file always matches the grammar of the sdl binary that
wrote it. Regenerate after upgrading the toolchain.

The targets are:

	vim     a Vim syntax file (Vimscript)
	vscode  a TextMate grammar in JSON, the form Visual Studio Code
	        and other TextMate-based editors load

Highlight only writes the definition; packaging it into an editor's
own configuration is the user's affair. Wiring the current user's
setup looks as follows.

Vim (for Neovim, read ~/.config/nvim for ~/.vim):

	mkdir -p ~/.vim/syntax ~/.vim/ftdetect
	sdl highlight vim > ~/.vim/syntax/sdl.vim
	echo 'au BufRead,BufNewFile *.sdl setfiletype sdl' \
		> ~/.vim/ftdetect/sdl.vim

Visual Studio Code loads grammars from extensions; a minimal local
one is a directory holding the grammar and a manifest:

	mkdir -p ~/.vscode/extensions/sdl-syntax/syntaxes
	sdl highlight vscode \
		> ~/.vscode/extensions/sdl-syntax/syntaxes/sdl.tmLanguage.json

with this manifest as ~/.vscode/extensions/sdl-syntax/package.json:

	{
		"name": "sdl-syntax",
		"version": "0.0.0",
		"engines": { "vscode": "^1.0.0" },
		"contributes": {
			"languages": [
				{ "id": "sdl", "extensions": [".sdl"] }
			],
			"grammars": [
				{
					"language": "sdl",
					"scopeName": "source.sdl",
					"path": "./syntaxes/sdl.tmLanguage.json"
				}
			]
		}
	}

Exit status 0 means the definition was printed; 2 reports an unknown
target or a malformed invocation.`,
}

func init() {
	CmdHighlight.Run = runHighlight
}

// targets maps each highlight target to its emitter. All emitters
// render the same collected grammar; a target is an editor's file
// format, never its own idea of the language.
var targets = map[string]func(io.Writer, grammar) error{
	"vim":    emitVim,
	"vscode": emitVSCode,
}

func runHighlight(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return &base.UsageError{Msg: fmt.Sprintf("highlight takes exactly one target argument, got %d", len(args))}
	}
	emit, ok := targets[args[0]]
	if !ok {
		return &base.UsageError{Msg: fmt.Sprintf("unknown highlight target %q (targets are vim and vscode)", args[0])}
	}
	return emit(s.Stdout, collect())
}
