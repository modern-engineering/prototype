// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestCompletionScripts pins the shipped completion scripts byte for
// byte and holds them to the shells themselves where installed: -n
// syntax-checks each script, and a scripted bash session sources the
// bash one and asserts the completer's answers over the registered
// tree. Registering a verb changes the scripts by construction;
// refresh the goldens with go test -update and review the diff as
// part of the registration.
func TestCompletionScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the built CLI; skipped in -short mode")
	}
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			res := runSDL(t, repoRoot, "completion", shell)
			if res.code != 0 {
				t.Fatalf("sdl completion %s exited %d\n%s", shell, res.code, res.stderr)
			}
			if res.stderr != "" {
				t.Errorf("stderr = %q, want empty", res.stderr)
			}
			golden := filepath.Join("testdata", "sdl."+shell)
			if *update {
				if err := os.WriteFile(golden, []byte(res.stdout), 0o666); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if res.stdout != string(want) {
				t.Errorf("script differs from %s; run go test -update and review the diff", golden)
			}

			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed; shell checks skipped", shell)
			}
			script := filepath.Join(t.TempDir(), "sdl."+shell)
			if err := os.WriteFile(script, []byte(res.stdout), 0o666); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(path, "-n", script).CombinedOutput(); err != nil {
				t.Errorf("%s -n rejected the script: %v\n%s", shell, err, out)
			}
			if shell == "bash" {
				assertBashCompleter(t, path, script)
			}
		})
	}
}

// assertBashCompleter drives the emitted bash functions the way
// readline would — COMP_WORDS set, _sdl called, COMPREPLY read — at
// the positions a user tabs at. The expected sets name today's
// registered surface; a new verb updates the top-level row together
// with the goldens.
func assertBashCompleter(t *testing.T, bash, script string) {
	complete := func(words ...string) []string {
		t.Helper()
		args := append([]string{"--norc", "-c", `
source "$1"; shift
COMP_WORDS=("$@")
COMP_CWORD=$(( ${#COMP_WORDS[@]} - 1 ))
_sdl
printf '%s\n' "${COMPREPLY[@]}"
`, "bash", script}, words...)
		out, err := exec.Command(bash, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("bash simulation %q: %v\n%s", words, err, out)
		}
		reply := strings.Fields(string(out))
		slices.Sort(reply)
		return reply
	}
	tests := []struct {
		words []string
		want  []string
	}{
		{[]string{"sdl", ""}, []string{"build", "completion", "echo", "fmt", "help", "highlight", "image", "run"}},
		{[]string{"sdl", "image", ""}, []string{"edit", "info", "records", "symbols"}},
		{[]string{"sdl", "image", "edit", "-"}, []string{"-generation"}},
		{[]string{"sdl", "run", "-"}, []string{"-extern", "-grace", "-work"}},
		{[]string{"sdl", "completion", ""}, []string{"bash", "zsh"}},
		{[]string{"sdl", "highlight", ""}, []string{"vim", "vscode"}},
		{[]string{"sdl", "help", "image", ""}, []string{"edit", "info", "records", "symbols"}},
	}
	for _, tt := range tests {
		if got := complete(tt.words...); !slices.Equal(got, tt.want) {
			t.Errorf("complete(%q) = %q, want %q", tt.words, got, tt.want)
		}
	}
}
