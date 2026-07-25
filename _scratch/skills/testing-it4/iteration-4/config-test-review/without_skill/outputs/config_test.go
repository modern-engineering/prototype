package config_test

import (
	"reflect"
	"strings"
	"sync"
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

// load parses src or fails the test.
func load(t *testing.T, src string) *config.Config {
	t.Helper()
	cfg, err := config.Load(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

// TestLoad verifies that Load parses pairs, trims whitespace, and
// skips blank lines and comments.
func TestLoad(t *testing.T) {
	cfg := load(t, sample)
	if got := cfg.String("name", ""); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("port", ""); got != "8080" {
		t.Errorf("String(port) = %q, want %q", got, "8080")
	}
	// The key after the comment line is still parsed; the comment
	// itself is not.
	if got := cfg.String("retries", ""); got != "3" {
		t.Errorf("String(retries) = %q, want %q", got, "3")
	}
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestLoadErrors verifies that malformed input produces positioned
// "line N: ..." errors.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{"missing equals", "name = app\nnot a pair\n", `line 2: missing '=' in "not a pair"`},
		{"empty key", "= value\n", "line 1: empty key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(strings.NewReader(tt.src))
			if err == nil {
				t.Fatalf("Load(%q) succeeded, want error %q", tt.src, tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("Load(%q) error = %q, want %q", tt.src, err.Error(), tt.wantErr)
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
	if got := cfg.Int("name", 42); got != 42 { // "app" does not parse
		t.Errorf("Int(name) = %d, want %d", got, 42)
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
	if got := cfg.Duration("port", time.Second); got != time.Second { // "8080" has no unit
		t.Errorf("Duration(port) = %v, want %v", got, time.Second)
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

// TestConcurrentReads verifies that a loaded Config is safe for
// concurrent readers. Run with -race to make this meaningful.
func TestConcurrentReads(t *testing.T) {
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
			cfg.Keys()
		}()
	}
	wg.Wait()
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}
