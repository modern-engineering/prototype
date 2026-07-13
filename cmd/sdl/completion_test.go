// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"regexp"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// TestCompletionCoversCommandTree is the anti-drift gate of the
// generated completion: every command path and every flag of the
// registered tree, plus the dispatcher's help verb, must appear in
// every shell's script. The gate walks base.Commands itself, so a
// verb registered here is demanded of the emitters automatically;
// the e2e goldens pin the exact bytes, this test the coverage.
func TestCompletionCoversCommandTree(t *testing.T) {
	type want struct {
		word string
		re   *regexp.Regexp
	}
	wants := []want{{"help", regexp.MustCompile(`\bhelp\b`)}}
	var walk func(cmds []*base.Command)
	walk = func(cmds []*base.Command) {
		for _, cmd := range cmds {
			name := cmd.Name()
			wants = append(wants, want{name, regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)})
			cmd.Flag.VisitAll(func(f *flag.Flag) {
				wants = append(wants, want{"-" + f.Name, regexp.MustCompile(`-` + regexp.QuoteMeta(f.Name) + `\b`)})
			})
			walk(cmd.Commands)
		}
	}
	walk(base.Commands)
	if len(wants) < 10 { // more than a few verbs, or the registered tree went missing
		t.Fatalf("collected only %d dispatch words from base.Commands", len(wants))
	}
	for _, shell := range []string{"bash", "zsh"} {
		out := completionScript(t, shell)
		for _, w := range wants {
			if !w.re.MatchString(out) {
				t.Errorf("%s script does not carry dispatch word %q", shell, w.word)
			}
		}
	}
}

// completionScript renders one shell's script through the real
// dispatch route, reading the verb's payload off the invocation
// streams.
func completionScript(t *testing.T, shell string) string {
	t.Helper()
	var stdout, stderr strings.Builder
	s := base.Streams{Stdout: &stdout, Stderr: &stderr}
	if code := invoke(context.Background(), s, []string{"completion", shell}); code != 0 {
		t.Fatalf("invoke(completion %s) = %d, want 0\n%s", shell, code, stderr.String())
	}
	return stdout.String()
}
