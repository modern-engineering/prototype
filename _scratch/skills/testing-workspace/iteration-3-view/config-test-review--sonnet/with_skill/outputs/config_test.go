package config_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

// sample is a small configuration file exercising every accessor's value
// type (string, int, duration), a comment, a blank line, and two values
// that fail to parse so their accessors must fall back to the default.
const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
bad_port = notanumber
bad_timeout = notaduration
`

// TestConfig walks the exported API the way a caller does: load a file,
// then read its values back through each typed accessor. One scenario
// covers the whole contract instead of one test per method, since callers
// always compose Load with the accessors rather than call them in
// isolation.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// a present key returns its value; an unknown key falls back to def
	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf("String(missing) = %q, want %q", got, "fallback")
	}

	// a decimal value parses as int; an unparseable or unknown value falls
	// back to def
	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Int("bad_port", 99); got != 99 {
		t.Errorf("Int(bad_port) = %d, want %d", got, 99)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf("Int(missing) = %d, want %d", got, 42)
	}

	// a duration value parses with time.ParseDuration; an unparseable or
	// unknown value falls back to def
	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 2*time.Second)
	}
	if got := cfg.Duration("bad_timeout", time.Minute); got != time.Minute {
		t.Errorf("Duration(bad_timeout) = %v, want %v", got, time.Minute)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf("Duration(missing) = %v, want %v", got, time.Second)
	}

	// comments and blank lines never become keys; Keys is sorted
	want := []string{"bad_port", "bad_timeout", "name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestLoadRejectsMalformedInput verifies that a line which is neither
// blank, a comment, nor a "key = value" pair produces an error rather
// than silently dropping or misreading the line, matching the package
// doc's documented failure mode.
func TestLoadRejectsMalformedInput(t *testing.T) {
	cases := []struct {
		reason string
		input  string
	}{
		{"no '=' separator", "just some text"},
		{"empty key before '='", " = value"},
	}
	for _, tc := range cases {
		if _, err := config.Load(strings.NewReader(tc.input)); err == nil {
			t.Errorf("Load(%q) succeeded, want error for %s", tc.input, tc.reason)
		}
	}
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		config.Load(strings.NewReader(sample))
	}
}
