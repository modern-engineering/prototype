package config_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

// sample uses every syntax rule the package doc names: a comment line,
// blank lines, and stray whitespace around a key and its value. The
// "retries" value is deliberately neither an integer nor a duration.
const sample = `
# server settings
  name   =   app
port = 8080
timeout = 2s
retries = many
`

// One scenario walks the whole documented lifecycle over one file:
// load it once, then read it back through every accessor. The
// accessors are covered together because that is how programs use the
// package: load once, look up many. Each accessor is asked for a
// present key, a missing key, and (where parsing is involved) a key
// whose value does not parse, since the docs promise the default in
// both failure shapes. The final Keys check also proves the comment
// and blank lines produced no keys.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// "  name   =   app" carries whitespace on every side of the pair.
	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf(`String("name") = %q, want "app" with whitespace trimmed`, got)
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf(`String("missing") = %q, want the "fallback" default`, got)
	}

	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf(`Int("port") = %d, want 8080`, got)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf(`Int("missing") = %d, want the default 42`, got)
	}
	// "retries" is configured, but "many" is not a decimal integer.
	if got := cfg.Int("retries", 7); got != 7 {
		t.Errorf(`Int("retries") = %d, want the default 7 for an unparseable value`, got)
	}

	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf(`Duration("timeout") = %v, want 2s`, got)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf(`Duration("missing") = %v, want the default 1s`, got)
	}
	// "many" is not a duration either.
	if got := cfg.Duration("retries", time.Minute); got != time.Minute {
		t.Errorf(`Duration("retries") = %v, want the default 1m for an unparseable value`, got)
	}

	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v (every configured key, sorted)", got, want)
	}
}

// The docs promise exactly one error behavior: a line that is neither
// blank, a comment, nor a "key = value" pair fails Load with an error
// positioned as "line N: ...". Only the position prefix is asserted;
// the wording after it is not part of the contract and is free to
// change.
func TestLoadErrorReportsLineNumber(t *testing.T) {
	// The bad line is third, after a pair and a comment, so a correct
	// position proves ignored lines still count toward N.
	const bad = "name = app\n# comment\nnot a pair\n"
	_, err := config.Load(strings.NewReader(bad))
	if err == nil {
		t.Fatal(`Load succeeded, want an error for a line with no "="`)
	}
	if !strings.HasPrefix(err.Error(), "line 3:") {
		t.Errorf(`Load error = %q, want a "line 3:" prefix`, err)
	}
}

// Tracks the cost of parsing a small, typical file.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
