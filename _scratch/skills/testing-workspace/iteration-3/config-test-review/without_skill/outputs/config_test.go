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

// TestLoad verifies that Load parses keys and values from a reader.
// It observes the result through the exported API so the test does not
// depend on the Config's internal representation.
func TestLoad(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.String("name", ""); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("port", ""); got != "8080" {
		t.Errorf("String(port) = %q, want %q", got, "8080")
	}
}

// TestLoadErrors verifies that malformed input produces positioned
// errors and no Config.
func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"missing equals", "name = app\nbogus line\n", `line 2: missing '=' in "bogus line"`},
		{"empty key", "= value\n", "line 1: empty key"},
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
			if cfg != nil {
				t.Errorf("Load(%q) config = %v, want nil", tt.input, cfg)
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
	if got := cfg.Int("name", 42); got != 42 {
		t.Errorf("Int(name) = %d, want fallback %d for non-numeric value", got, 42)
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
	if got := cfg.Duration("port", time.Second); got != time.Second {
		t.Errorf("Duration(port) = %v, want fallback %v for non-duration value", got, time.Second)
	}
}

// TestKeys verifies that Keys returns the keys in sorted order.
func TestKeys(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestConcurrentUse verifies that a loaded Config is safe for
// concurrent readers. Run with -race to make this meaningful.
func TestConcurrentUse(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var wg sync.WaitGroup
	for range 4 {
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
			b.Fatalf("Load: %v", err)
		}
	}
}
