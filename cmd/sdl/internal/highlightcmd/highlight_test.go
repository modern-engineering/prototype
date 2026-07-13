// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package highlightcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution"
)

var update = flag.Bool("update", false, "rewrite golden files from the observed output")

// goldens maps each target to its golden file; the tests iterate the
// real emitter table so a target cannot register without a golden.
var goldens = map[string]string{
	"vim":    "sdl.vim",
	"vscode": "sdl.tmLanguage.json",
}

// generate renders one target's output through the emitter table, the
// route runHighlight takes.
func generate(t *testing.T, target string) string {
	t.Helper()
	emit, ok := targets[target]
	if !ok {
		t.Fatalf("no emitter registered for target %q", target)
	}
	var buf bytes.Buffer
	if err := emit(&buf, collect()); err != nil {
		t.Fatalf("emit %s: %v", target, err)
	}
	return buf.String()
}

// TestGolden pins every target's output byte for byte. The goldens
// regenerate with -update; a diff there is a change to the language
// surface and reviews as one.
func TestGolden(t *testing.T) {
	if len(goldens) != len(targets) {
		t.Fatalf("%d goldens for %d targets; give every target a golden", len(goldens), len(targets))
	}
	for target, golden := range goldens {
		t.Run(target, func(t *testing.T) {
			got := generate(t, target)
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
				t.Errorf("%s output differs from %s; run go test -update and review the diff", target, path)
			}
		})
	}
}

// TestOutputsCarryGrammar is the anti-drift gate of the generated
// tooling: every keyword the scanner recognizes and every word of the
// linker's body vocabulary must appear, word-bounded, in every
// target's output. An emitter that misses a grammar addition fails
// here even when the golden was refreshed without review.
func TestOutputsCarryGrammar(t *testing.T) {
	var words []string
	for kw := range token.Keywords() {
		words = append(words, kw)
	}
	vocab := solution.Vocabulary()
	words = append(words, vocab.Sections...)
	for verb, fields := range vocab.RootFields {
		words = append(words, verb)
		words = append(words, fields...)
	}
	if len(words) <= 10 { // more than the ten keywords, or the vocabulary went missing
		t.Fatalf("collected only %d grammar words: %q", len(words), words)
	}
	for target := range targets {
		out := generate(t, target)
		for _, word := range words {
			bounded := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
			if !bounded.MatchString(out) {
				t.Errorf("%s output does not carry grammar word %q", target, word)
			}
		}
	}
}

// TestVSCodeOutputIsJSON holds the TextMate emission to well-formed
// JSON, the one property of that format checkable without an editor.
func TestVSCodeOutputIsJSON(t *testing.T) {
	if out := generate(t, "vscode"); !json.Valid([]byte(out)) {
		t.Errorf("vscode output is not valid JSON:\n%s", out)
	}
}

// TestRunHighlightUsage pins the usage contract: anything but exactly
// one known target is a usage fault, decided before a byte of output
// is written.
func TestRunHighlightUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"vim", "vscode"}, {"emacs"}} {
		err := runHighlight(context.Background(), CmdHighlight, args)
		var usage *base.UsageError
		if !errors.As(err, &usage) {
			t.Errorf("runHighlight(%q) = %v, want a UsageError", args, err)
		}
	}
}
