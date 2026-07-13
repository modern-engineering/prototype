// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package lspcmd implements sdl lsp, a language server for solution
// units speaking the Language Server Protocol over standard streams.
// The protocol layer is hand-rolled JSON-RPC 2.0 (jsonrpc.go): the LSP
// base protocol is a page of framing rules, x/tools keeps its own
// implementation internal, and a dependency would outweigh the code.
// The language layer is borrowed whole from the toolchain: published
// diagnostics are vetcmd.CheckSource's single-file findings, and the
// completion list is generated from sdl/token's keywords and
// solution.Vocabulary — the highlight and vet stance carried into a
// live editor session, so this server cannot disagree with the
// compiler about the language either.
//
// The v0 surface is deliberately small: full-document sync, error
// diagnostics, one context-free completion list, one message handled
// at a time. Recorded doors, each reopened by the first editor
// session that needs it: incremental sync, UTF-16 column arithmetic
// (diagnostic columns are byte counts today), position-encoding
// negotiation, catalogue-aware completion, and gating traffic on the
// initialize handshake — the server currently answers any client
// that talks to it, the permissive default.
package lspcmd

import (
	"context"
	"errors"
	"io"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// CmdLsp is the sdl lsp command.
var CmdLsp = &base.Command{
	UsageLine: "sdl lsp",
	Short:     "serve the language server protocol for solution units",
	Long: `Lsp serves the Language Server Protocol to an editor over standard
input and output, wiring live diagnostics and completion for solution
units into any LSP-capable editor. An editor's language client starts
it and owns the whole session; running it by hand at a terminal does
nothing useful.

The server is basic by design. Documents sync whole rather than
incrementally; opening or changing a unit re-checks it and publishes
positioned diagnostics, and closing it clears them. The checks are
sdl vet's single-file half — syntax errors plus the source shapes the
build is certain to reject — and share vet's catalogue-free stance:
element resolution, parameter names and types, and value binding stay
with sdl build. Checks that judge a whole unit set at once, duplicate
names across sibling units and unused symbols, need every unit of the
solution and stay with sdl vet; run it beside the editor session.

Completion is context-free: at any position the server offers the
grammar's keywords, the section words, and the statement verbs'
top-level fields, generated from the same vocabulary the compiler
enforces.

Neovim wires the built-in LSP client in init.lua (the filetype line
is redundant once the ftdetect file from 'sdl help highlight' is
installed):

	vim.filetype.add({ extension = { sdl = "sdl" } })
	vim.api.nvim_create_autocmd("FileType", {
		pattern = "sdl",
		callback = function()
			vim.lsp.start({ name = "sdl", cmd = { "sdl", "lsp" } })
		end,
	})

Visual Studio Code reaches a language server through a client
extension; a minimal local one is a directory holding a manifest, an
activation script, and the client library:

	mkdir -p ~/.vscode/extensions/sdl-lsp
	cd ~/.vscode/extensions/sdl-lsp
	npm install vscode-languageclient

with this package.json:

	{
		"name": "sdl-lsp",
		"version": "0.0.0",
		"main": "./extension.js",
		"engines": { "vscode": "^1.75.0" },
		"activationEvents": ["onLanguage:sdl"],
		"contributes": {
			"languages": [
				{ "id": "sdl", "extensions": [".sdl"] }
			]
		}
	}

and this extension.js:

	const { LanguageClient } = require("vscode-languageclient/node");
	let client;
	exports.activate = function () {
		client = new LanguageClient("sdl", "sdl lsp",
			{ command: "sdl", args: ["lsp"] },
			{ documentSelector: [{ language: "sdl" }] });
		client.start();
	};
	exports.deactivate = function () {
		return client && client.stop();
	};

Together with the syntax definitions of sdl highlight, this completes
the editor kit that accompanies solution units.

Exit status 0 means the client ended the session cleanly — a shutdown
request followed by the exit notification or by closing the stream;
2 reports a malformed invocation, malformed protocol traffic, or a
session that ended without the shutdown handshake.`,
}

func init() {
	CmdLsp.Run = runLsp
}

func runLsp(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
	if len(args) != 0 {
		return &base.UsageError{Msg: "lsp takes no arguments"}
	}
	return run(ctx, s.Stdin, s.Stdout)
}

// run speaks one session over the caller's streams and folds an
// interrupt into the verdict. The client owns the session's end, but
// a done ctx must still unblock the pending read (main folds the
// interrupt signal into ctx). Closing the input is no lever when it
// is a standard stream: blocking-mode descriptors sit outside the
// runtime's poller, so a close does not interrupt a read already
// parked on them. Serving from an in-process pipe restores the lever
// — pipe reads do unblock when either end closes — at the price of
// one pump goroutine, which an interrupt strands on its blocked read
// until the input ends: an instant later in production, where the
// process exits, and at the test's own hand under a bubble.
func run(ctx context.Context, in io.Reader, out io.Writer) error {
	pr, pw := io.Pipe()
	go func() {
		_, err := io.Copy(pw, in)
		pw.CloseWithError(err) // nil folds to EOF: the input's end is the pipe's end
	}()
	unblock := context.AfterFunc(ctx, func() { pr.CloseWithError(errors.New("session interrupted")) })
	defer unblock()
	err := serve(pr, out)
	if err != nil && ctx.Err() != nil {
		return errors.New("session interrupted")
	}
	return err
}
