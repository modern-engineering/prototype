// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution"
)

// The exported vocabulary carries the closed word lists sorted — the
// generator-facing shape — and hands every caller an independent
// copy, so corruption cannot reach the linker's own checks.
func TestVocabulary(t *testing.T) {
	v := solution.Vocabulary()
	if want := []string{"metadata", "params", "with"}; !slices.Equal(v.Sections, want) {
		t.Errorf("Sections = %q, want %q", v.Sections, want)
	}
	if want := []string{"location"}; !slices.Equal(v.RootFields["deploy"], want) {
		t.Errorf("RootFields[deploy] = %q, want %q", v.RootFields["deploy"], want)
	}
	if fields, ok := v.RootFields["provision"]; !ok || len(fields) != 0 {
		t.Errorf("RootFields[provision] = %q (present %v), want an empty list", fields, ok)
	}
	for verb, fields := range v.RootFields {
		if !slices.IsSorted(fields) {
			t.Errorf("RootFields[%s] = %q is not sorted", verb, fields)
		}
	}
	if !slices.IsSorted(v.Sections) {
		t.Errorf("Sections = %q is not sorted", v.Sections)
	}

	// The value is the caller's copy: corrupting it must not reach
	// the vocabulary the linker checks against.
	v.Sections[0] = "corrupted"
	v.RootFields["deploy"][0] = "corrupted"
	fresh := solution.Vocabulary()
	if fresh.Sections[0] != "metadata" || fresh.RootFields["deploy"][0] != "location" {
		t.Error("mutating a returned Vocab reached the vocabulary itself")
	}
}

// The linker holds to the export's promise: the diagnostics that
// teach the closed vocabularies enumerate exactly the exported words,
// so tooling generated from [solution.Vocabulary] cannot drift from
// the checks. The expected lines are rebuilt from the export, never
// hard-coded, so growing the vocabulary keeps the wiring pinned
// rather than the words.
func TestVocabularyWiresLinker(t *testing.T) {
	v := solution.Vocabulary()

	sections := strings.Join(v.Sections[:len(v.Sections)-1], ", ") + ", and " + v.Sections[len(v.Sections)-1]
	deployFields := "deploy takes " + strings.Join(v.RootFields["deploy"], ", ")
	provisionFields := "provision takes no top-level fields"
	if len(v.RootFields["provision"]) > 0 {
		provisionFields = "provision takes " + strings.Join(v.RootFields["provision"], ", ")
	}

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "unknown section enumerates the section words",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tmount {\n" +
				"\t}\n" +
				"}\n",
			want: fmt.Sprintf("u.sdl:4:2: unknown section mount: sections are %s", sections),
		},
		{
			name: "unknown deploy field enumerates the verb's list",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\tzone: here\n" +
				"}\n",
			want: fmt.Sprintf("u.sdl:4:2: unknown top-level field zone: %s", deployFields),
		},
		{
			name: "unknown provision field enumerates the verb's list",
			source: "solution sample\n" +
				"import sub \"example.com/acme/substrate\"\n" +
				"provision sub.Store as legacy {\n" +
				"\tzone: here\n" +
				"}\n",
			want: fmt.Sprintf("u.sdl:4:2: unknown top-level field zone: %s", provisionFields),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := compile(t, solution.CompileConfig{
				Solution:  "sample",
				Units:     []solution.Unit{{Name: "u.sdl", Source: tt.source}},
				Catalogue: testCatalogue(),
			})
			if code != 1 {
				t.Fatalf("exit code = %d, want 1; stderr:\n%s", code, stderr)
			}
			if stderr != tt.want+"\n" {
				t.Errorf("stderr = %q, want %q", stderr, tt.want+"\n")
			}
		})
	}
}
