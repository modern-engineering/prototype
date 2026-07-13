// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// A command's returned error selects the exit code and the stderr
// rendering, the base package's whole taxonomy held in process for
// every verb at once: solution diagnostics print their positioned
// lines and exit 1 — or print nothing when a child already reported
// them — a relayed child exit passes through verbatim and silent, a
// usage fault points at help and exits 2, and anything else reports
// as an sdl fault at 2. Classification survives wrapping, so verbs
// may annotate on the way up. The payload stream stays untouched
// throughout: errors never corrupt an emitted image.
func TestExitTaxonomy(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantStderr string
	}{
		{"success", nil, 0, ""},
		{"diagnostics print their lines", &base.DiagnosticsError{Lines: []string{"a.sdl:1:2: boom", "b.sdl:3:4: bang"}}, 1,
			"a.sdl:1:2: boom\nb.sdl:3:4: bang\n"},
		{"relayed diagnostics stay silent", &base.DiagnosticsError{}, 1, ""},
		{"wrapped diagnostics still classify", fmt.Errorf("link: %w", &base.DiagnosticsError{}), 1, ""},
		{"relayed exit passes verbatim", &base.RelayedExit{Code: 7}, 7, ""},
		{"usage fault points at help", &base.UsageError{Msg: "scratch takes no arguments"}, 2,
			"sdl: scratch takes no arguments\nRun 'sdl help scratch' for usage.\n"},
		{"internal fault", errors.New("toolchain exploded"), 2, "sdl: toolchain exploded\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &base.Command{
				UsageLine: "sdl scratch",
				Run: func(ctx context.Context, s base.Streams, cmd *base.Command, args []string) error {
					return tt.err
				},
			}
			var stdout, stderr strings.Builder
			s := base.Streams{Stdout: &stdout, Stderr: &stderr}
			if got := run(context.Background(), s, cmd, nil); got != tt.wantCode {
				t.Errorf("run() = %d, want %d", got, tt.wantCode)
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.wantStderr)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want it untouched", stdout.String())
			}
		})
	}
}

// Help renders the registered command tree, never a hand-maintained
// copy: 'sdl help' lists every top-level command with its short
// description on stdout, a command topic prints its usage line and
// long text, a group topic lists its subcommands behind a deeper help
// pointer, and a bare 'sdl' says the same top-level words on stderr
// at exit 2 — the tree itself is the oracle here, so registering a
// verb is all it takes to be spoken for.
func TestHelpRendersCommandTree(t *testing.T) {
	helpOut := func(t *testing.T, args ...string) string {
		t.Helper()
		var stdout, stderr strings.Builder
		s := base.Streams{Stdout: &stdout, Stderr: &stderr}
		if code := invoke(context.Background(), s, args); code != 0 {
			t.Fatalf("invoke(%q) = %d, want 0\n%s", args, code, stderr.String())
		}
		if stderr.Len() != 0 {
			t.Errorf("invoke(%q) wrote on stderr:\n%s", args, stderr.String())
		}
		return stdout.String()
	}

	t.Run("top-level usage", func(t *testing.T) {
		out := helpOut(t, "help")
		for _, cmd := range base.Commands {
			if !strings.Contains(out, cmd.Name()) || !strings.Contains(out, cmd.Short) {
				t.Errorf("usage does not carry %q with its description", cmd.Name())
			}
		}
	})

	t.Run("command topic", func(t *testing.T) {
		cmd := base.Lookup(base.Commands, "build")
		out := helpOut(t, "help", "build")
		if !strings.Contains(out, "usage: "+cmd.UsageLine) || !strings.Contains(out, cmd.Long) {
			t.Errorf("help build does not carry the usage line and the long text:\n%s", out)
		}
	})

	t.Run("group topic lists subcommands", func(t *testing.T) {
		group := base.Lookup(base.Commands, "image")
		out := helpOut(t, "help", "image")
		for _, sub := range group.Commands {
			if !strings.Contains(out, sub.Name()) || !strings.Contains(out, sub.Short) {
				t.Errorf("help image does not carry subcommand %q", sub.Name())
			}
		}
		if !strings.Contains(out, `"sdl help image <command>"`) {
			t.Errorf("help image does not point at the deeper help:\n%s", out)
		}
	})

	t.Run("bare sdl says usage on stderr", func(t *testing.T) {
		var stdout, stderr strings.Builder
		s := base.Streams{Stdout: &stdout, Stderr: &stderr}
		if code := invoke(context.Background(), s, nil); code != 2 {
			t.Fatalf("invoke() = %d, want 2", code)
		}
		if stdout.Len() != 0 {
			t.Errorf("bare sdl wrote on stdout:\n%s", stdout.String())
		}
		if got, want := stderr.String(), helpOut(t, "help"); got != want {
			t.Errorf("bare sdl's stderr differs from 'sdl help':\n%s", got)
		}
	})
}

// Malformed invocations — an unknown command, a bare command group, a
// bad flag, surplus arguments — classify as usage faults: exit 2, a
// report on the error stream, and a silent payload stream. Every case
// fails before its command would touch the file system or the
// toolchain.
func TestUsageFaults(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no arguments", nil},
		{"unknown command", []string{"frobnicate"}},
		{"bare command group", []string{"image"}},
		{"unknown subcommand", []string{"image", "nope"}},
		{"bad flag", []string{"echo", "-nope"}},
		{"surplus arguments", []string{"echo", "a.json", "b.json"}},
		{"missing image edit argument", []string{"image", "edit"}},
		{"missing highlight target", []string{"highlight"}},
		{"unknown highlight target", []string{"highlight", "emacs"}},
		{"missing completion shell", []string{"completion"}},
		{"unknown completion shell", []string{"completion", "fish"}},
		{"bad lsp flag", []string{"lsp", "-nope"}},
		{"surplus lsp arguments", []string{"lsp", "unit.sdl"}},
		{"bad vet flag", []string{"vet", "-nope"}},
		{"unknown help topic", []string{"help", "nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			s := base.Streams{Stdout: &stdout, Stderr: &stderr}
			if got := invoke(context.Background(), s, tt.args); got != 2 {
				t.Errorf("invoke(%q) = %d, want 2", tt.args, got)
			}
			if stderr.Len() == 0 {
				t.Errorf("invoke(%q) reported nothing on stderr", tt.args)
			}
			if stdout.Len() != 0 {
				t.Errorf("invoke(%q) wrote on stdout:\n%s", tt.args, stdout.String())
			}
		})
	}
}
