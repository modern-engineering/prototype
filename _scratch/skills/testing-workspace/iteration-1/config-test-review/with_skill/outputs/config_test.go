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

// sample exercises every documented line form: key/value pairs, a
// blank line, a comment, and whitespace around keys and values.
const sample = `
name = app
port = 8080
timeout = 2s
# retries controls how often we try again
retries = 3
`

// A loaded Config answers every typed lookup from one parse. String,
// Int, and Duration return the parsed value for a known key. Unknown
// keys and values that do not parse fall back to the caller's default.
// Keys lists the configured keys in sorted order.
func TestLoadAndLookups(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.String("name", "fallback"), "app"; got != want {
		t.Errorf(`String("name") = %q, want %q`, got, want)
	}
	if got, want := cfg.String("missing", "fallback"), "fallback"; got != want {
		t.Errorf(`String("missing") = %q, want %q`, got, want)
	}
	if got, want := cfg.Int("port", 0), 8080; got != want {
		t.Errorf(`Int("port") = %d, want %d`, got, want)
	}
	if got, want := cfg.Int("missing", 42), 42; got != want {
		t.Errorf(`Int("missing") = %d, want %d`, got, want)
	}
	if got, want := cfg.Int("name", 42), 42; got != want {
		t.Errorf(`Int("name") = %d, want default %d for unparsable value`, got, want)
	}
	if got, want := cfg.Duration("timeout", 0), 2*time.Second; got != want {
		t.Errorf(`Duration("timeout") = %v, want %v`, got, want)
	}
	if got, want := cfg.Duration("missing", time.Second), time.Second; got != want {
		t.Errorf(`Duration("missing") = %v, want %v`, got, want)
	}
	if got, want := cfg.Duration("port", time.Second), time.Second; got != want {
		t.Errorf(`Duration("port") = %v, want default %v for unparsable value`, got, want)
	}
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Load rejects a line that is neither blank, a comment, nor a
// "key = value" pair with an error positioned as "line N: ...".
// Blank and comment lines still count toward the position.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		prefix string
	}{
		{"missing equals", "name = app\nbroken\n", "line 2:"},
		{"empty key", "= value\n", "line 1:"},
		{"position counts blanks and comments", "# comment\n\nbroken\n", "line 3:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(strings.NewReader(tt.input))
			if err == nil {
				t.Fatalf("Load(%q) succeeded, want error", tt.input)
			}
			if !strings.HasPrefix(err.Error(), tt.prefix) {
				t.Errorf("Load(%q) error = %q, want %q prefix", tt.input, err, tt.prefix)
			}
		})
	}
}

func ExampleLoad() {
	cfg, err := config.Load(strings.NewReader("port = 8080\ntimeout = 2s\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.Int("port", 0))
	fmt.Println(cfg.Duration("timeout", time.Second))
	// Output:
	// 8080
	// 2s
}

func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
