// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package completioncmd

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

var update = flag.Bool("update", false, "rewrite golden files from the observed output")

// testTree builds a small command tree exercising every emitter
// branch: a command group, boolean and value flags, a repeatable
// flag, an enumerated first argument, file arguments, a command that
// takes nothing at all, and apostrophes in the described surface.
// The real tree lives in package main; its scripts are pinned by the
// e2e goldens there, while these goldens pin the emitters' rendering
// of each tree shape.
func testTree() []*base.Command {
	pack := &base.Command{
		UsageLine: "sdl pack [-o output] [-force] [-tag name=value]... [dir]",
		Short:     "pack a directory's material",
	}
	var (
		packOut   string
		packForce bool
	)
	pack.Flag.StringVar(&packOut, "o", "", "write the archive to `file` instead of stdout")
	pack.Flag.BoolVar(&packForce, "force", false, "overwrite what's already there")
	pack.Flag.Func("tag", "attach one `name=value` tag (repeatable)", func(string) error { return nil })

	pick := &base.Command{
		UsageLine: "sdl pick <red|green>",
		Short:     "pick a color",
	}

	wait := &base.Command{
		UsageLine: "sdl wait",
		Short:     "wait for nothing in particular",
	}

	info := &base.Command{
		UsageLine: "sdl tool info [target]",
		Short:     "print a tool's description",
	}
	run := &base.Command{
		UsageLine: "sdl tool run [-n N] [file ...]",
		Short:     "run a tool",
	}
	var runN int
	run.Flag.IntVar(&runN, "n", 1, "repeat `N` times")
	tool := &base.Command{
		UsageLine: "sdl tool <command> [arguments]",
		Short:     "toolbox of subcommands",
		Commands:  []*base.Command{info, run},
	}

	return []*base.Command{pack, pick, tool, wait}
}

// goldens maps each shell to its golden file; the tests iterate the
// real emitter table so a shell cannot register without a golden.
var goldens = map[string]string{
	"bash": "tree.bash",
	"zsh":  "tree.zsh",
}

// generate renders one shell's script for the test tree through the
// emitter table, the route runCompletion takes.
func generate(t *testing.T, shell string) string {
	t.Helper()
	emit, ok := shells[shell]
	if !ok {
		t.Fatalf("no emitter registered for shell %q", shell)
	}
	var buf bytes.Buffer
	if err := emit(&buf, collect(testTree())); err != nil {
		t.Fatalf("emit %s: %v", shell, err)
	}
	return buf.String()
}

// TestGolden pins every shell's rendering of the test tree byte for
// byte. The goldens regenerate with -update; a diff here is a change
// to how the emitters render a tree shape and reviews as one.
func TestGolden(t *testing.T) {
	if len(goldens) != len(shells) {
		t.Fatalf("%d goldens for %d shells; give every shell a golden", len(goldens), len(shells))
	}
	for shell, golden := range goldens {
		t.Run(shell, func(t *testing.T) {
			got := generate(t, shell)
			path := filepath.Join("testdata", golden)
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o666); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run go test -update to write it)", err)
			}
			if got != string(want) {
				t.Errorf("%s output differs from %s; run go test -update and review the diff", shell, path)
			}
		})
	}
}

// TestCollect pins the walk itself: flag kinds and placeholders from
// the flag set, enumerations and file arguments from the usage lines,
// subcommands from the group.
func TestCollect(t *testing.T) {
	specs := collect(testTree())
	want := []spec{
		{
			name:  "pack",
			short: "pack a directory's material",
			flags: []flagSpec{
				{name: "force", usage: "overwrite what's already there", boolean: true},
				{name: "o", arg: "file", usage: "write the archive to file instead of stdout"},
				{name: "tag", arg: "name=value", usage: "attach one name=value tag (repeatable)", repeats: true},
			},
			files: true,
		},
		{
			name:  "pick",
			short: "pick a color",
			words: []string{"red", "green"},
		},
		{
			name:  "tool",
			short: "toolbox of subcommands",
			subs: []spec{
				{name: "info", short: "print a tool's description", files: true},
				{name: "run", short: "run a tool", flags: []flagSpec{{name: "n", arg: "N", usage: "repeat N times"}}, files: true},
			},
		},
		{
			name:  "wait",
			short: "wait for nothing in particular",
		},
	}
	if !reflect.DeepEqual(specs, want) {
		t.Errorf("collect(testTree()) =\n%+v\nwant\n%+v", specs, want)
	}
}

// TestPositionals covers the usage-line trailer parser over the
// shapes the registered commands actually write.
func TestPositionals(t *testing.T) {
	tests := []struct {
		usageLine string
		longName  string
		words     []string
		files     bool
	}{
		{"sdl build [-o output] [-generation N] [-work] [dir]", "build", nil, true},
		{"sdl echo [image]", "echo", nil, true},
		{"sdl fmt [-l] [-w] [-d] [path ...]", "fmt", nil, true},
		{"sdl highlight <vim|vscode>", "highlight", []string{"vim", "vscode"}, false},
		{"sdl image edit [-generation N] <image>", "image edit", nil, true},
		{"sdl run [-extern name=value]... [-grace duration] [-work] [dir]", "run", nil, true},
		{"sdl wait", "wait", nil, false},
		{"sdl odd <name=value|other>", "odd", nil, true},
	}
	for _, tt := range tests {
		words, files := positionals(tt.usageLine, tt.longName)
		if !slices.Equal(words, tt.words) || files != tt.files {
			t.Errorf("positionals(%q) = %q, %v; want %q, %v",
				tt.usageLine, words, files, tt.words, tt.files)
		}
	}
}

// TestRunCompletionUsage pins the usage contract: anything but
// exactly one known shell is a usage fault, decided before a byte of
// output is written.
func TestRunCompletionUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"bash", "zsh"}, {"fish"}} {
		err := runCompletion(context.Background(), CmdCompletion, args)
		var usage *base.UsageError
		if !errors.As(err, &usage) {
			t.Errorf("runCompletion(%q) = %v, want a UsageError", args, err)
		}
	}
}

// TestScriptSyntax holds each golden to its shell's own parser via
// the shells' -n mode, where the shell is installed.
func TestScriptSyntax(t *testing.T) {
	for shell := range shells {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed; syntax smoke skipped", shell)
			}
			script := filepath.Join(t.TempDir(), "script")
			if err := os.WriteFile(script, []byte(generate(t, shell)), 0o666); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command(path, "-n", script).CombinedOutput(); err != nil {
				t.Errorf("%s -n rejected the generated script: %v\n%s", shell, err, out)
			}
		})
	}
}

// TestBashCompleter drives the generated bash functions the way
// readline would — COMP_WORDS set, _sdl called, COMPREPLY read — and
// asserts the words the tree promises at each position.
func TestBashCompleter(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed; completer simulation skipped")
	}
	script := filepath.Join(t.TempDir(), "tree.bash")
	if err := os.WriteFile(script, []byte(generate(t, "bash")), 0o666); err != nil {
		t.Fatal(err)
	}
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
		{[]string{"sdl", ""}, []string{"help", "pack", "pick", "tool", "wait"}},
		{[]string{"sdl", "p"}, []string{"pack", "pick"}},
		{[]string{"sdl", "pack", "-"}, []string{"-force", "-o", "-tag"}},
		{[]string{"sdl", "pick", ""}, []string{"green", "red"}},
		{[]string{"sdl", "tool", ""}, []string{"info", "run"}},
		{[]string{"sdl", "tool", "run", "-"}, []string{"-n"}},
		{[]string{"sdl", "help", ""}, []string{"pack", "pick", "tool", "wait"}},
		{[]string{"sdl", "help", "tool", ""}, []string{"info", "run"}},
		{[]string{"sdl", "wait", ""}, nil},
	}
	for _, tt := range tests {
		if got := complete(tt.words...); !slices.Equal(got, tt.want) {
			t.Errorf("complete(%q) = %q, want %q", tt.words, got, tt.want)
		}
	}
}
