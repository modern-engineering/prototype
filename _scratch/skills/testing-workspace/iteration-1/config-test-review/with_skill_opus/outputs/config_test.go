// The tests live in config_test so they exercise the package exactly as
// its users do: through the exported API, never the unexported value map.
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

const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
`

// Loading the documented sample and reading it back covers the whole
// happy path in one pass: comment and blank lines are dropped, surrounding
// whitespace is trimmed, and each typed accessor returns either the stored
// value or the caller's default for an unknown key.
func TestLoadAndLookup(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load(sample): %v", err)
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

	if got, want := cfg.Duration("timeout", 0), 2*time.Second; got != want {
		t.Errorf("Duration(timeout) = %v, want %v", got, want)
	}
	if got, want := cfg.Duration("missing", time.Second), time.Second; got != want {
		t.Errorf("Duration(missing) = %v, want %v", got, want)
	}

	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Int and Duration promise to fall back to the default when the stored
// value is present but does not parse, distinct from an absent key.
func TestTypedLookupRejectsUnparsableValues(t *testing.T) {
	cfg, err := config.Load(strings.NewReader("port = wat\ntimeout = later\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.Int("port", 42), 42; got != want {
		t.Errorf("Int(port=wat) = %d, want %d", got, want)
	}
	if got, want := cfg.Duration("timeout", time.Second), time.Second; got != want {
		t.Errorf("Duration(timeout=later) = %v, want %v", got, want)
	}
}

// A line that is neither blank, a comment, nor a key = value pair aborts
// Load with a positioned "line N: ..." error carrying the offending line.
func TestLoadReportsPositionedErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSub string
	}{
		{"missing equals", "name = app\nbroken\n", "line 2:"},
		{"empty key", "= value\n", "line 1:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(strings.NewReader(tt.input))
			if err == nil {
				t.Fatalf("Load(%q) = nil error, want one containing %q", tt.input, tt.wantSub)
			}
			if got := err.Error(); !strings.Contains(got, tt.wantSub) {
				t.Errorf("Load(%q) error = %q, want it to contain %q", tt.input, got, tt.wantSub)
			}
		})
	}
}

// A runnable example doubles as the package's usage documentation.
func Example() {
	cfg, err := config.Load(strings.NewReader("name = app\nport = 8080\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.String("name", "default"))
	fmt.Println(cfg.Int("port", 80))
	// Output:
	// app
	// 8080
}

func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
