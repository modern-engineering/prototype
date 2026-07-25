package config

import (
	"slices"
	"strings"
	"testing"
	"time"
)

const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
`

// TestLoad verifies that Load parses "key = value" lines, skipping
// blank lines and comments, and that the result is queryable through
// the accessor methods.
func TestLoad(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.String("name", ""); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 2*time.Second)
	}
}

// TestLoadRejectsMalformedLines verifies the positioned error that
// Load documents for a line that is neither blank, a comment, nor a
// "key = value" pair.
func TestLoadRejectsMalformedLines(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"missing equals", "just-a-word\n"},
		{"empty key", " = value\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(tt.in))
			if err == nil {
				t.Fatal("Load: want error, got nil")
			}
			if got, want := err.Error(), "line 1:"; !strings.HasPrefix(got, want) {
				t.Errorf("Load: err = %q, want prefix %q", got, want)
			}
		})
	}
}

// TestString verifies that String returns values and defaults.
func TestString(t *testing.T) {
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
}

// TestInt verifies that Int parses integer values.
func TestInt(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf("Int(missing) = %d, want %d", got, 42)
	}
}

// TestDuration verifies that Duration parses duration values.
func TestDuration(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 2*time.Second)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf("Duration(missing) = %v, want %v", got, time.Second)
	}
}

// TestKeys verifies that Keys returns the keys in sorted order.
func TestKeys(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Load(strings.NewReader(sample))
	}
}
