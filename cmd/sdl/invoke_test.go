// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"context"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

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
