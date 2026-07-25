package config

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// sample exercises the documented file syntax in one place: blank
// lines, a comment line, and whitespace around keys and values.
const sample = `
# server settings
name  =  app
port = 8080
timeout = 2s
retries = soon
`

// A whole-package scenario: load sample, then read it back through
// every exported lookup, covering the documented fallbacks for both
// unknown keys and values that do not parse.
func TestConfig(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf("String(missing) = %q, want %q", got, "fallback")
	}

	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf("Int(missing) = %d, want %d", got, 42)
	}
	if got := cfg.Int("retries", 42); got != 42 {
		t.Errorf("Int(retries) = %d, want fallback %d for an unparsable value", got, 42)
	}

	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 2*time.Second)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf("Duration(missing) = %v, want %v", got, time.Second)
	}
	if got := cfg.Duration("name", time.Second); got != time.Second {
		t.Errorf("Duration(name) = %v, want fallback %v for an unparsable value", got, time.Second)
	}

	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Lines that are neither blank, comments, nor "key = value" pairs
// make Load fail rather than return a partial Config.
func TestMalformedLines(t *testing.T) {
	for _, bad := range []string{
		"no equals sign",
		"= value without a key",
	} {
		if cfg, err := Load(strings.NewReader(bad)); err == nil {
			t.Errorf("Load(%q) = %v, want error", bad, cfg)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		Load(strings.NewReader(sample))
	}
}
