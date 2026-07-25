package config_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

// A service reads its settings once at startup: Load parses the file,
// then each accessor supplies a typed default for anything the file
// leaves out or gets wrong.
func Example_serverSettings() {
	cfg, err := config.Load(strings.NewReader(`
listen = :8080
timeout = 500ms
# workers is tuned per deployment; unset here, so the default applies.
`))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(cfg.String("listen", ":80"))
	// An unknown key falls back to the caller's default.
	fmt.Println(cfg.Int("workers", 4))
	fmt.Println(cfg.Duration("timeout", time.Second))

	// Output:
	// :8080
	// 4
	// 500ms
}

// sample exercises the whole documented format in one fixture: a
// comment, a blank line, and spacing that varies around '=' so the
// promised trimming is actually load-bearing. TestConfig's expectations
// mirror these pairs.
const sample = `# Display name, reported by the health endpoint.
name = app
port=8080

	timeout   =   2s
retries = 3
`

// One Load serves every accessor, the way a caller reads several
// settings out of a single file: present keys come back typed, missing
// keys fall back, and a value of the wrong shape falls back too rather
// than erroring.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load(sample) = %v, want nil", err)
	}

	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf(`String("name") = %q, want "app"`, got)
	}
	if got := cfg.String("absent", "fallback"); got != "fallback" {
		t.Errorf(`String("absent") = %q, want the fallback`, got)
	}

	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf(`Int("port") = %d, want 8080`, got)
	}
	if got := cfg.Int("absent", 42); got != 42 {
		t.Errorf(`Int("absent") = %d, want the default 42`, got)
	}
	// "app" is not a number: reading an existing key through the wrong
	// accessor is the documented misuse, and the default must win.
	if got := cfg.Int("name", 42); got != 42 {
		t.Errorf(`Int("name") = %d, want the default 42 for a non-numeric value`, got)
	}

	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf(`Duration("timeout") = %v, want 2s`, got)
	}
	if got := cfg.Duration("absent", time.Second); got != time.Second {
		t.Errorf(`Duration("absent") = %v, want the default 1s`, got)
	}
	// "8080" carries no unit, so time.ParseDuration rejects it.
	if got := cfg.Duration("port", time.Second); got != time.Second {
		t.Errorf(`Duration("port") = %v, want the default 1s for a unitless value`, got)
	}

	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// Both malformed lines hide on line 2 behind a line Load ignores, so
// the position in the error proves that skipped lines still count.
// The "line N:" prefix is the form the package doc promises; a typed
// error carrying the line number would let this check drop the string
// match (flagged with the owner).
func TestLoadRejectsMalformedLines(t *testing.T) {
	cases := []struct {
		input      string
		invalidity string
	}{
		{"# tuning knobs\nport 8080\n", "a line missing '='"},
		{"\n = app\n", "an empty key"},
	}
	for _, c := range cases {
		_, err := config.Load(strings.NewReader(c.input))
		if err == nil {
			t.Errorf("Load accepted %s, want an error", c.invalidity)
			continue
		}
		if !strings.HasPrefix(err.Error(), "line 2:") {
			t.Errorf("Load error for %s = %q, want a \"line 2:\" prefix", c.invalidity, err)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		if _, err := config.Load(strings.NewReader(sample)); err != nil {
			b.Fatal(err)
		}
	}
}
