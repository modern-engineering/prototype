package config_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
`

// Loading a configuration file makes its accessors return the parsed
// values, or fall back to the caller's default for a key that is
// missing or holds a value the accessor can't parse.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.String("name", "fallback"), "app"; got != want {
		t.Errorf("String(name) = %q, want %q", got, want)
	}
	if got, want := cfg.String("missing", "fallback"), "fallback"; got != want {
		t.Errorf("String(missing) = %q, want %q", got, want)
	}

	if got, want := cfg.Int("port", 0), 8080; got != want {
		t.Errorf("Int(port) = %d, want %d", got, want)
	}
	if got, want := cfg.Int("missing", 42), 42; got != want {
		t.Errorf("Int(missing) = %d, want %d", got, want)
	}
	if got, want := cfg.Int("name", 7), 7; got != want {
		t.Errorf("Int(name) = %d, want %d", got, want) // "app" doesn't parse as an integer
	}

	if got, want := cfg.Duration("timeout", 0), 2*time.Second; got != want {
		t.Errorf("Duration(timeout) = %v, want %v", got, want)
	}
	if got, want := cfg.Duration("missing", time.Second), time.Second; got != want {
		t.Errorf("Duration(missing) = %v, want %v", got, want)
	}
	if got, want := cfg.Duration("name", time.Minute), time.Minute; got != want {
		t.Errorf("Duration(name) = %v, want %v", got, want) // "app" doesn't parse as a duration
	}

	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Load reports an error for a line that is neither blank, a comment,
// nor a "key = value" pair.
func TestMalformedLines(t *testing.T) {
	tests := []string{
		"name\n",             // missing '='
		"= value\n",          // empty key
		"key = ok\nbroken\n", // second line malformed
	}
	for _, input := range tests {
		if _, err := config.Load(strings.NewReader(input)); err == nil {
			t.Errorf("Load(%q) succeeded, want error", input)
		}
	}
}

// Load repeatedly parses the sample configuration to measure the cost
// of the common case: a small, well-formed file.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		config.Load(strings.NewReader(sample))
	}
}
