package config

import (
	"reflect"
	"strings"
	"sync"
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

// load is a test helper that parses src and fails the test on error.
func load(t *testing.T, src string) *Config {
	t.Helper()
	cfg, err := Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// TestLoad verifies that Load parses keys and values from a reader,
// skipping blank lines and comments and trimming whitespace.
func TestLoad(t *testing.T) {
	cfg := load(t, sample)
	for key, want := range map[string]string{
		"name":    "app",
		"port":    "8080",
		"timeout": "2s",
		"retries": "3",
	} {
		if got := cfg.String(key, ""); got != want {
			t.Errorf("String(%q) = %q, want %q", key, got, want)
		}
	}
	if got, want := len(cfg.Keys()), 4; got != want {
		t.Errorf("len(Keys()) = %d, want %d", got, want)
	}
}

// TestLoadTrimsWhitespace verifies that whitespace around keys and
// values is trimmed and that values may contain '='.
func TestLoadTrimsWhitespace(t *testing.T) {
	cfg := load(t, "  spaced   =   v  \nexpr = a=b\n")
	if got := cfg.String("spaced", ""); got != "v" {
		t.Errorf("String(spaced) = %q, want %q", got, "v")
	}
	if got := cfg.String("expr", ""); got != "a=b" {
		t.Errorf("String(expr) = %q, want %q", got, "a=b")
	}
}

// TestLoadErrors verifies that malformed input produces positioned
// errors that count blank and comment lines.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"missing equals", "name app\n", `line 1: missing '=' in "name app"`},
		{"empty key", " = value\n", "line 1: empty key"},
		{"positioned after comments", "# c\n\nbroken\n", `line 3: missing '=' in "broken"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(strings.NewReader(tt.input))
			if err == nil {
				t.Fatalf("Load(%q) = %v, want error", tt.input, cfg)
			}
			if got := err.Error(); got != tt.wantErr {
				t.Errorf("Load(%q) error = %q, want %q", tt.input, got, tt.wantErr)
			}
		})
	}
}

// TestString verifies that String returns values and defaults.
func TestString(t *testing.T) {
	cfg := load(t, sample)
	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf("String(missing) = %q, want %q", got, "fallback")
	}
}

// TestInt verifies that Int parses integer values and falls back to
// the default for unknown keys and unparseable values.
func TestInt(t *testing.T) {
	cfg := load(t, sample)
	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf("Int(missing) = %d, want %d", got, 42)
	}
	if got := cfg.Int("name", 42); got != 42 {
		t.Errorf("Int(name) = %d, want fallback %d for non-integer value", got, 42)
	}
}

// TestDuration verifies that Duration parses duration values and falls
// back to the default for unknown keys and unparseable values.
func TestDuration(t *testing.T) {
	cfg := load(t, sample)
	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(timeout) = %v, want %v", got, 2*time.Second)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf("Duration(missing) = %v, want %v", got, time.Second)
	}
	if got := cfg.Duration("port", time.Second); got != time.Second {
		t.Errorf("Duration(port) = %v, want fallback %v for unitless value", got, time.Second)
	}
}

// TestKeys verifies that Keys returns the keys in sorted order.
func TestKeys(t *testing.T) {
	cfg := load(t, sample)
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestConcurrentUse verifies that concurrent lookups on a shared
// Config are safe. Run with -race to catch violations.
func TestConcurrentUse(t *testing.T) {
	cfg := load(t, sample)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := cfg.String("name", ""); got != "app" {
				t.Errorf("String(name) = %q, want %q", got, "app")
			}
			if got := cfg.Int("port", 0); got != 8080 {
				t.Errorf("Int(port) = %d, want %d", got, 8080)
			}
		}()
	}
	wg.Wait()
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
