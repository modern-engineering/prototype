// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package vetcmd

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// writeUnits materializes one unit set in a fresh directory.
func writeUnits(t *testing.T, units map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range units {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// vet runs one invocation over the given paths, returning the printed
// stderr lines and the error the dispatch layer would translate.
func vet(t *testing.T, paths ...string) ([]string, error) {
	t.Helper()
	var buf strings.Builder
	v := &vetter{stderr: &buf}
	err := v.run(paths)
	out := strings.TrimSuffix(buf.String(), "\n")
	if out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), err
}

// Every v0 check holds its whole finding: the wording (the linker's,
// verbatim, where the check shadows one; the flag-style unused wording
// where it does not), the anchoring position, and the clean cases that
// must stay quiet. ${DIR} in a want line stands for the unit set's
// directory.
func TestVetFindings(t *testing.T) {
	tests := []struct {
		name  string
		units map[string]string
		want  []string
	}{
		{
			name: "unknown section word",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	bogus {
	}
}
`},
			want: []string{"${DIR}/a.sdl:4:2: unknown section bogus: sections are metadata, params, and with"},
		},
		{
			name: "retired on keeps its migration message",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	on {
	}
}
`},
			want: []string{"${DIR}/a.sdl:4:2: unknown section on: deployment intent moved to top-level fields, controller schemes to with <qualifier> stanzas"},
		},
		{
			name: "params refuses a qualifier",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	params k8s.pod {
	}
}
`},
			want: []string{"${DIR}/a.sdl:4:9: params takes no qualifier"},
		},
		{
			name: "metadata refuses a qualifier",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	metadata k8s.pod {
	}
}
`},
			want: []string{"${DIR}/a.sdl:4:11: metadata takes no qualifier"},
		},
		{
			name: "with requires a qualifier",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	with {
	}
}
`},
			want: []string{"${DIR}/a.sdl:4:2: with requires a qualifier, e.g. with k8s.pod"},
		},
		{
			name: "duplicate params section",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	params {
	}
	params {
	}
}
`},
			want: []string{"${DIR}/a.sdl:6:2: duplicate params section (first declared at ${DIR}/a.sdl:4:2)"},
		},
		{
			name: "with stanzas dedup per qualifier",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	with k8s.pod {
	}
	with k8s.pod {
	}
	with k8s.workload {
	}
}
`},
			want: []string{"${DIR}/a.sdl:6:2: duplicate with k8s.pod section (first declared at ${DIR}/a.sdl:4:2)"},
		},
		{
			name: "unknown top-level field on deploy",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	count: 3
	location: here
}
`},
			want: []string{"${DIR}/a.sdl:4:2: unknown top-level field count: deploy takes location"},
		},
		{
			name: "provision takes no top-level fields",
			units: map[string]string{"a.sdl": `solution s

provision p.T attach as a {
	location: here
}
`},
			want: []string{"${DIR}/a.sdl:4:2: unknown top-level field location: provision takes no top-level fields"},
		},
		{
			name: "verb default checks its top-level scheme",
			units: map[string]string{"a.sdl": `solution s

default deploy {
	locus: here
}
`},
			want: []string{"${DIR}/a.sdl:4:2: unknown top-level field locus: deploy takes location"},
		},
		{
			name: "type default root fields pass unjudged",
			units: map[string]string{"a.sdl": `solution s

default p.T {
	anything: 1
}
`},
			want: nil,
		},
		{
			name: "metadata values must be strings",
			units: map[string]string{"a.sdl": `solution s

deploy p.T as a {
	metadata {
		team: "search"
		size: 3
	}
}
`},
			want: []string{"${DIR}/a.sdl:6:9: metadata values must be string literals"},
		},
		{
			name: "duplicate symbol across the unit set",
			units: map[string]string{
				"a.sdl": `solution s

var x: 1

deploy p.T as b {
	params {
		k: x
	}
}
`,
				"b.sdl": `solution s

deploy p.T as x {
	params {
		k: x
	}
}
`,
			},
			want: []string{"${DIR}/b.sdl:3:15: duplicate symbol x (first declared at ${DIR}/a.sdl:3:5)"},
		},
		{
			name: "unused symbols, and a stanza token marks nothing used",
			units: map[string]string{"a.sdl": `solution s

extern dbServer p.T

var (
	color: "blue"
	subject: "s"
)

deploy p.T as a {
	params {
		target: subject
	}
	with acme.x {
		hue: color
	}
}
`},
			want: []string{
				"${DIR}/a.sdl:3:8: symbol dbServer declared and not used",
				"${DIR}/a.sdl:6:2: symbol color declared and not used",
			},
		},
		{
			name: "a dotted reference head counts as a use",
			units: map[string]string{"a.sdl": `solution s

var q: "v"

deploy p.T as a {
	params {
		k: q.out
	}
}
`},
			want: nil,
		},
		{
			name: "a default's params section counts uses",
			units: map[string]string{"a.sdl": `solution s

var x: "v"

default deploy {
	params {
		k: x
	}
}

deploy p.T as a {
}
`},
			want: nil,
		},
		{
			name: "clean multi-unit solution",
			units: map[string]string{
				"a.sdl": `solution s

extern e p.T

var x: "v"

provision p.S attach as prov {
	params {
		endpoint: e
	}
}
`,
				"b.sdl": `solution s

deploy p.T as a {
	location: there

	params {
		k: x
		nats: prov.config
	}

	with k8s.pod {
		replicas: 3
	}

	metadata {
		team: "t"
	}
}
`,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeUnits(t, tt.units)
			want := make([]string, len(tt.want))
			for i, w := range tt.want {
				want[i] = strings.ReplaceAll(w, "${DIR}", dir)
			}
			got, err := vet(t, dir)
			if !slices.Equal(got, want) {
				t.Errorf("findings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if len(want) == 0 {
				if err != nil {
					t.Errorf("run = %v, want nil", err)
				}
				return
			}
			var diags *base.DiagnosticsError
			if !errors.As(err, &diags) {
				t.Errorf("run = %v, want a DiagnosticsError selecting exit 1", err)
			}
		})
	}
}

// Syntax gates the set checks: a unit set with a
// broken member reports the syntax findings and the clean members'
// unit findings, but never judges duplicates or unusedness over half a
// namespace.
func TestSyntaxGatesSetChecks(t *testing.T) {
	dir := writeUnits(t, map[string]string{
		"a.sdl": `solution s

var lonely: 1

deploy p.T as a {
	bogus {
	}
}
`,
		"b.sdl": `solution s

deploy p.T as {
`,
	})
	got, err := vet(t, dir)
	var diags *base.DiagnosticsError
	if !errors.As(err, &diags) {
		t.Errorf("run = %v, want a DiagnosticsError selecting exit 1", err)
	}
	all := strings.Join(got, "\n")
	if want := dir + "/a.sdl:6:2: unknown section bogus"; !strings.Contains(all, want) {
		t.Errorf("findings lack the clean unit's own check %q:\n%s", want, all)
	}
	if want := dir + "/b.sdl:3:"; !strings.Contains(all, want) {
		t.Errorf("findings lack a syntax error anchored at %q:\n%s", want, all)
	}
	if strings.Contains(all, "lonely") {
		t.Errorf("set-scope unused check ran over a gated set:\n%s", all)
	}
}

// The unit set's boundary is the directory: sibling
// solution directories under one walked root never share a namespace,
// so a name reused across solutions is not a duplicate.
func TestVetGroupsByDirectory(t *testing.T) {
	root := t.TempDir()
	for _, sub := range []string{"one", "two"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o777); err != nil {
			t.Fatal(err)
		}
		src := "solution " + sub + "\n\ndeploy p.T as a {\n}\n"
		if err := os.WriteFile(filepath.Join(root, sub, "u.sdl"), []byte(src), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	got, err := vet(t, root)
	if err != nil || got != nil {
		t.Errorf("run = %v with findings %q, want clean", err, got)
	}
}

// Overlapping arguments dedup: a file named both
// directly and through its directory is checked once, so its findings
// do not double.
func TestVetOverlappingArguments(t *testing.T) {
	dir := writeUnits(t, map[string]string{"a.sdl": `solution s

var lonely: 1
`})
	got, _ := vet(t, dir, filepath.Join(dir, "a.sdl"))
	want := []string{dir + "/a.sdl:3:5: symbol lonely declared and not used"}
	if !slices.Equal(got, want) {
		t.Errorf("findings = %q, want %q exactly once", got, want)
	}
}

// Unreadable paths are plain errors, not findings: faults print, win
// the exit code (2), and still do not silence the findings
// of the paths that were readable.
func TestVetPathFaults(t *testing.T) {
	dir := writeUnits(t, map[string]string{"a.sdl": `solution s

var lonely: 1
`})
	got, err := vet(t, filepath.Join(dir, "missing.sdl"), dir)
	if err == nil || err.Error() != "1 error" {
		t.Errorf("run = %v, want the 1-error fault", err)
	}
	var diags *base.DiagnosticsError
	if errors.As(err, &diags) {
		t.Errorf("run = %v, a DiagnosticsError; faults must select exit 2", err)
	}
	all := strings.Join(got, "\n")
	if !strings.Contains(all, "missing.sdl") {
		t.Errorf("fault not reported:\n%s", all)
	}
	if !strings.Contains(all, "symbol lonely declared and not used") {
		t.Errorf("findings silenced by the fault:\n%s", all)
	}
}

// CheckSource is the single-file contract the language
// server builds on: syntax and unit-scope findings come back sorted,
// within-file duplicates included, and the unused check — whole-set
// knowledge — never runs.
func TestCheckSource(t *testing.T) {
	if got := CheckSource("u.sdl", []byte("solution s\n\nvar lonely: 1\n")); len(got) != 0 {
		t.Errorf("unused check ran at single-file scope: %v", got)
	}

	got := CheckSource("u.sdl", []byte("solution s\n\nvar x: 1\n\ndeploy p.T as x {\n}\n"))
	want := "u.sdl:5:15: duplicate symbol x (first declared at u.sdl:3:5)"
	if len(got) != 1 || got[0].Error() != want {
		t.Errorf("findings = %v, want exactly %q", got, want)
	}

	got = CheckSource("u.sdl", []byte("solution s\n\ndeploy p.T as a {\n\ton {\n\t}\n}\n"))
	if len(got) != 1 || !strings.Contains(got[0].Error(), "unknown section on") {
		t.Errorf("findings = %v, want the on migration message", got)
	}

	if got := CheckSource("u.sdl", []byte("solution s\n\ndeploy p.T as {\n")); len(got) == 0 {
		t.Error("syntax errors did not surface as findings")
	}
}
