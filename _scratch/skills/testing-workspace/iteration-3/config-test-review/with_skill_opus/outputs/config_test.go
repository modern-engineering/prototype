package config_test

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

// sample exercises every line kind the package documents: a plain pair, a
// numeric value, a duration, a '#' comment, and the surrounding blank lines.
// One fixture feeds every test, so a parser change surfaces in one place.
const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
`

// TestConfig loads the sample once and drives the whole public API the way a
// caller does, because users compose these accessors rather than call one in
// isolation. It checks each typed accessor for a present key, a missing key,
// and (for Int and Duration) a value that does not parse, since the docs
// promise a fall back to def in all three situations.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load(sample): %v", err)
	}

	// String: the stored value wins; an unknown key yields def.
	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf(`String("name") = %q, want "app"`, got)
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf(`String("missing") = %q, want "fallback"`, got)
	}

	// Int: parses a decimal; unknown and unparseable values yield def.
	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf(`Int("port") = %d, want 8080`, got)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf(`Int("missing") = %d, want 42`, got)
	}
	if got := cfg.Int("name", 42); got != 42 { // "app" is not a number
		t.Errorf(`Int("name") = %d, want 42 (fallback for unparseable value)`, got)
	}

	// Duration: parses via time.ParseDuration, with the same fallback rule.
	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf(`Duration("timeout") = %v, want 2s`, got)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf(`Duration("missing") = %v, want 1s`, got)
	}
	if got := cfg.Duration("name", time.Second); got != time.Second { // "app" is not a duration
		t.Errorf(`Duration("name") = %v, want 1s (fallback for unparseable value)`, got)
	}

	// Keys: every configured key, sorted; the comment line is not a key.
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestLoadRejectsMalformedLine pins the promise that a line which is not
// blank, a comment, or a "key = value" pair is reported as an error rather
// than silently dropped. We assert only that an error is returned: the
// documented "line N: ..." text is not exposed as a value, so matching the
// wording would pin incidental phrasing (see report.md).
func TestLoadRejectsMalformedLine(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"no separator", "name = app\ngarbage\n"},
		{"empty key", "name = app\n = value\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := config.Load(strings.NewReader(tc.input)); err == nil {
				t.Errorf("Load(%q) = nil error, want a positioned error", tc.input)
			}
		})
	}
}

// Example documents the everyday use: load a reader, then read typed values,
// falling back to a default for any key the file omits (here, "host").
func Example() {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.String("name", "unnamed"))
	fmt.Println(cfg.Int("port", 80))
	fmt.Println(cfg.Duration("timeout", time.Minute))
	fmt.Println(cfg.String("host", "localhost"))
	// Output:
	// app
	// 8080
	// 2s
	// localhost
}

// BenchmarkLoad tracks the cost of parsing the sample. The b.Loop form (Go
// 1.24) keeps the returned *Config live, so the parse cannot be optimized
// away the way a discarded call in a b.N loop can be.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
