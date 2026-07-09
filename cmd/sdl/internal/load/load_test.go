// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package load

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// writeUnits lays the given files down in a fresh solution directory.
func writeUnits(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestDirDiagnostics drives the fault paths of Dir: every case must come
// back as one *base.DiagnosticsError carrying exactly the expected
// positioned lines — never a panic, never a partial success.
func TestDirDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  []string // exact diagnostic lines, in report order
	}{
		{
			// Regression: "import ff" with no path leaves a partial
			// ImportSpec (Path == nil); walking it used to panic. The
			// healthy unit's import fault must still be collected in
			// the same report.
			name: "malformed import does not panic",
			files: map[string]string{
				"bad.sdl": "solution s\n\nimport ff\n",
				"ok.sdl":  "solution s\n\nimport x \"example.com/x/...\"\n",
			},
			want: []string{
				"bad.sdl:3:10: expected import path string, found newline",
				`ok.sdl:3:8: import "example.com/x/...": import path must not contain wildcards`,
			},
		},
		{
			name:  "no sdl files",
			files: map[string]string{"README.md": "not a unit\n"},
			want:  []string{"no .sdl files in %DIR%"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeUnits(t, tt.files)
			_, err := Dir(dir)
			var diags *base.DiagnosticsError
			if !errors.As(err, &diags) {
				t.Fatalf("Dir error = %v (%T), want *base.DiagnosticsError", err, err)
			}
			want := make([]string, len(tt.want))
			for i, line := range tt.want {
				want[i] = strings.ReplaceAll(line, "%DIR%", dir)
			}
			if got := diags.Lines; !equal(got, want) {
				t.Errorf("diagnostics mismatch\n--- got ---\n%s\n--- want ---\n%s",
					strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		})
	}
}

// TestDirAliasScope pins that reference names are none of load's
// business: one alias naming two different paths in two units is legal
// (import scope is the unit, the linker's per-file tables resolve it),
// and load's union is only the deduplicated path set for discovery.
func TestDirAliasScope(t *testing.T) {
	dir := writeUnits(t, map[string]string{
		"a.sdl": "solution s\n\nimport ff \"example.com/one\"\n",
		"b.sdl": "solution s\n\nimport ff \"example.com/two\"\n",
		"c.sdl": "solution s\n\nimport pp \"example.com/one\"\n",
	})
	sol, err := Dir(dir)
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	var paths []string
	for _, imp := range sol.Imports {
		paths = append(paths, imp.Path)
	}
	if want := []string{"example.com/one", "example.com/two"}; !equal(paths, want) {
		t.Errorf("import set = %v, want %v (deduplicated by path)", paths, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestCheckImportPath drives the load-pattern gate: strings that cannot
// name a single package must be rejected before they reach the Go
// toolchain as package patterns or the shell as arguments.
func TestCheckImportPath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr string // empty means accepted
	}{
		{"example.com/ok", ""},
		{"example.com/v2", ""},
		{"", "empty import path"},
		{"all", "not a package path"},
		{"std", "not a package path"},
		{"cmd", "not a package path"},
		{"example.com/x/...", "import path must not contain wildcards"},
		{"-flag", "import path must not begin with -"},
		{"example.com/a b", "import path contains invalid characters"},
		{"example.com/a\"b", "import path contains invalid characters"},
		{"example.com/a`b", "import path contains invalid characters"},
		{"example.com/a\nb", "import path contains invalid characters"},
	}
	for _, tt := range tests {
		err := checkImportPath(tt.path)
		if tt.wantErr == "" {
			if err != nil {
				t.Errorf("checkImportPath(%q) = %v, want nil", tt.path, err)
			}
			continue
		}
		if err == nil || err.Error() != tt.wantErr {
			t.Errorf("checkImportPath(%q) = %v, want %q", tt.path, err, tt.wantErr)
		}
	}
}

// TestDirOrdering pins the deterministic load order: units sorted by
// filename (which also selects the solution name and the first import
// spec of a duplicated path) and imports sorted by path.
func TestDirOrdering(t *testing.T) {
	dir := writeUnits(t, map[string]string{
		"c.sdl": "solution gamma\n\nimport zz \"example.com/zz\"\n",
		"a.sdl": "solution alpha\n\nimport mm \"example.com/mm\"\n",
		"b.sdl": "solution beta\n\nimport aa \"example.com/aa\"\nimport mm \"example.com/mm\"\n",
	})
	sol, err := Dir(dir)
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if sol.Name != "alpha" {
		t.Errorf("solution name = %q, want %q (the first unit's clause)", sol.Name, "alpha")
	}
	var units []string
	for _, u := range sol.Units {
		units = append(units, u.Name)
	}
	if want := []string{"a.sdl", "b.sdl", "c.sdl"}; !equal(units, want) {
		t.Errorf("unit order = %v, want %v", units, want)
	}
	var paths []string
	for _, imp := range sol.Imports {
		paths = append(paths, imp.Path)
	}
	if want := []string{"example.com/aa", "example.com/mm", "example.com/zz"}; !equal(paths, want) {
		t.Errorf("import order = %v, want %v", paths, want)
	}
	// example.com/mm is imported by a.sdl and b.sdl; the recorded
	// position is the first spec in unit order.
	if got := sol.Imports[1].Pos.Filename; got != "a.sdl" {
		t.Errorf("duplicated path recorded at %s, want a.sdl", got)
	}
}
