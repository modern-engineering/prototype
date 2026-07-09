// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package fmtcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

const canonical = `solution sample

import ff "example.com/ff"

deploy ff.Ping as P1 {
	count: 3
}
`

// messy is the same unit with non-canonical layout only: extra spaces,
// a one-line body, and surplus blank lines.
const messy = `solution   sample

import ff "example.com/ff"



deploy ff.Ping as P1 { count: 3 }
`

const broken = `solution sample

deploy deploy deploy
`

// tree lays out a solution corpus for walking: a canonical file, two
// messy ones (one nested), and paths a directory walk must skip.
func tree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"clean.sdl":           canonical,
		"messy.sdl":           messy,
		"nested/inner.sdl":    messy,
		".hidden/skipped.sdl": messy,
		".dotfile.sdl":        messy,
		"notes.txt":           "not a unit at all",
		"nested/notes.md":     "also not a unit",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// format runs one formatter invocation, returning the streams and the
// verdict error.
func format(t *testing.T, f *formatter, paths ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errb bytes.Buffer
	f.stdout, f.stderr = &out, &errb
	err = f.run(paths)
	return out.String(), errb.String(), err
}

func TestStdoutMode(t *testing.T) {
	dir := tree(t)
	stdout, stderr, err := format(t, &formatter{}, filepath.Join(dir, "messy.sdl"))
	if err != nil || stderr != "" {
		t.Fatalf("err %v, stderr %q", err, stderr)
	}
	if stdout != canonical {
		t.Errorf("stdout:\n%s--- want ---\n%s", stdout, canonical)
	}
	// The source is untouched without -w.
	src, err := os.ReadFile(filepath.Join(dir, "messy.sdl"))
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != messy {
		t.Error("stdout mode rewrote the file")
	}
}

// TestStdoutModeCanonical prints even units that are already canonical,
// the gofmt filter contract.
func TestStdoutModeCanonical(t *testing.T) {
	dir := tree(t)
	stdout, _, err := format(t, &formatter{}, filepath.Join(dir, "clean.sdl"))
	if err != nil {
		t.Fatal(err)
	}
	if stdout != canonical {
		t.Errorf("stdout:\n%s--- want ---\n%s", stdout, canonical)
	}
}

func TestListWalk(t *testing.T) {
	dir := tree(t)
	stdout, stderr, err := format(t, &formatter{list: true}, dir)
	var diags *base.DiagnosticsError
	if !errors.As(err, &diags) {
		t.Fatalf("err %v, want a DiagnosticsError (exit 1)", err)
	}
	if stderr != "" {
		t.Errorf("stderr: %q", stderr)
	}
	want := []string{
		filepath.Join(dir, "messy.sdl"),
		filepath.Join(dir, "nested", "inner.sdl"),
	}
	if got := strings.Fields(stdout); !slices.Equal(got, want) {
		t.Errorf("listed %q, want %q (dot paths and non-.sdl files skipped)", got, want)
	}
}

func TestListClean(t *testing.T) {
	dir := tree(t)
	stdout, _, err := format(t, &formatter{list: true}, filepath.Join(dir, "clean.sdl"))
	if err != nil {
		t.Fatalf("err %v, want nil (exit 0)", err)
	}
	if stdout != "" {
		t.Errorf("stdout: %q, want empty", stdout)
	}
}

func TestWrite(t *testing.T) {
	dir := tree(t)
	path := filepath.Join(dir, "messy.sdl")
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := format(t, &formatter{write: true}, dir)
	if err != nil || stdout != "" || stderr != "" {
		t.Fatalf("err %v, stdout %q, stderr %q", err, stdout, stderr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != canonical {
		t.Errorf("rewritten file:\n%s--- want ---\n%s", got, canonical)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o640 {
		t.Errorf("permissions after rewrite: %o, want 640", perm)
	}

	// The tree is canonical now: a second listing finds nothing.
	stdout, _, err = format(t, &formatter{list: true}, dir)
	if err != nil || stdout != "" {
		t.Errorf("after -w: err %v, listed %q; want a clean tree", err, stdout)
	}
}

func TestDiff(t *testing.T) {
	dir := tree(t)
	path := filepath.Join(dir, "messy.sdl")
	stdout, _, err := format(t, &formatter{diff: true}, path)
	var diags *base.DiagnosticsError
	if !errors.As(err, &diags) {
		t.Fatalf("err %v, want a DiagnosticsError (exit 1)", err)
	}
	for _, want := range []string{
		"--- " + path + ".orig\n",
		"+++ " + path + "\n",
		"-solution   sample\n",
		"+solution sample\n",
		"-deploy ff.Ping as P1 { count: 3 }\n",
		"+deploy ff.Ping as P1 {\n",
		"+\tcount: 3\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("diff output is missing %q:\n%s", want, stdout)
		}
	}
}

// TestSyntaxErrors pins the exit-2 path: positioned reports on stderr,
// the remaining files still processed, and errors trumping differences.
func TestSyntaxErrors(t *testing.T) {
	dir := tree(t)
	if err := os.WriteFile(filepath.Join(dir, "broken.sdl"), []byte(broken), 0o666); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := format(t, &formatter{list: true}, dir)
	if err == nil || errors.As(err, new(*base.DiagnosticsError)) {
		t.Fatalf("err %v, want a plain error (exit 2)", err)
	}
	if !strings.Contains(stderr, filepath.Join(dir, "broken.sdl")+":3:8: ") {
		t.Errorf("stderr lacks a positioned syntax report:\n%s", stderr)
	}
	// The broken unit does not hide the messy ones.
	if !strings.Contains(stdout, filepath.Join(dir, "messy.sdl")) {
		t.Errorf("differing files were not listed alongside the fault:\n%s", stdout)
	}
}

func TestMissingPath(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := format(t, &formatter{}, filepath.Join(dir, "absent.sdl"))
	if err == nil {
		t.Fatal("err nil, want a plain error (exit 2)")
	}
	if stderr == "" {
		t.Error("no report on stderr")
	}
}

// TestExplicitFile formats explicitly named files regardless of their
// extension, the gofmt contract for file arguments.
func TestExplicitFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unit.txt")
	if err := os.WriteFile(path, []byte(messy), 0o666); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := format(t, &formatter{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if stdout != canonical {
		t.Errorf("stdout:\n%s--- want ---\n%s", stdout, canonical)
	}
}
