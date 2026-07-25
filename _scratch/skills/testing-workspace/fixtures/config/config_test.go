package config

import (
	"reflect"
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

// TestLoad verifies that Load parses keys and values from a reader.
func TestLoad(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.values["name"] != "app" {
		t.Errorf("values[name] = %q, want %q", cfg.values["name"], "app")
	}
	if cfg.values["port"] != "8080" {
		t.Errorf("values[port] = %q, want %q", cfg.values["port"], "8080")
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
	if got := cfg.Keys(); !reflect.DeepEqual(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// TestDebugDump dumps the parsed state so it can be checked manually.
func TestDebugDump(t *testing.T) {
	// t.Skip("only needed for local debugging")
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for k, v := range cfg.values {
		t.Logf("values[%q] = %q", k, v)
	}
	// check the dump manually when parsing looks off
}

// TestConcurrentUse verifies that a Config loaded in a goroutine works.
func TestConcurrentUse(t *testing.T) {
	var cfg *Config
	go func() {
		cfg, _ = Load(strings.NewReader(sample))
	}()
	time.Sleep(100 * time.Millisecond)
	if cfg == nil {
		t.Fatal("config was not loaded")
	}
	if got := cfg.String("name", ""); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Load(strings.NewReader(sample))
	}
}
