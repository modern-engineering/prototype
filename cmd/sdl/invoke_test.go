// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"context"
	"testing"
)

// TestInvokeExitCodes pins the exit-2 contract of the dispatch layer:
// malformed invocations — an unknown command, a bare command group, a
// bad flag, surplus arguments — classify as usage faults, never as
// diagnostics (1) or success (0). Every case fails before its command
// would touch the file system or the toolchain.
func TestInvokeExitCodes(t *testing.T) {
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
		{"unknown help topic", []string{"help", "nope"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := invoke(context.Background(), tt.args); got != 2 {
				t.Errorf("invoke(%q) = %d, want 2", tt.args, got)
			}
		})
	}
}
