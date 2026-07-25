package config_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"example.invalid/config"
)

// A representative file exercises every corner of the format the package
// promises: a blank first line and a comment line that are skipped, an
// indented line and padding around '=' that are trimmed, and one value of
// each type the typed accessors parse.
const sample = `
# service configuration
name = app
port = 8080
timeout = 250ms
  retries = 3
`

// This example loads a small configuration and reads typed values, each
// with a default that stands in when the key is absent.
func Example() {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println(cfg.String("name", "anonymous"))
	fmt.Println(cfg.Int("port", 80))
	fmt.Println(cfg.Duration("timeout", time.Second))
	fmt.Println(cfg.Int("attempts", 3)) // absent key falls back to the default
	// Output:
	// app
	// 8080
	// 250ms
	// 3
}

// Loading a well-formed file and then reading it back is the whole point of
// the package, so one flow drives every accessor: stored values come back
// typed, and the caller's default stands in whenever the key is absent or
// the stored text will not parse as the requested type.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf("String(name) = %q, want %q", got, "app")
	}
	if got := cfg.String("absent", "fallback"); got != "fallback" {
		t.Errorf("String(absent) = %q, want the default %q", got, "fallback")
	}

	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(port) = %d, want %d", got, 8080)
	}
	if got := cfg.Int("retries", 0); got != 3 {
		t.Errorf("Int(retries) = %d, want %d for an indented line", got, 3)
	}
	if got := cfg.Int("name", -1); got != -1 {
		t.Errorf("Int(name) = %d, want the default -1 for a non-integer value", got)
	}
	if got := cfg.Int("absent", 42); got != 42 {
		t.Errorf("Int(absent) = %d, want the default %d", got, 42)
	}

	if got := cfg.Duration("timeout", 0); got != 250*time.Millisecond {
		t.Errorf("Duration(timeout) = %v, want %v", got, 250*time.Millisecond)
	}
	if got := cfg.Duration("name", time.Second); got != time.Second {
		t.Errorf("Duration(name) = %v, want the default for a non-duration value", got)
	}
	if got := cfg.Duration("absent", time.Minute); got != time.Minute {
		t.Errorf("Duration(absent) = %v, want the default %v", got, time.Minute)
	}

	// The comment and blank lines contribute no keys, and Keys sorts what
	// remains.
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// A line that is neither blank, a comment, nor a "key = value" pair is
// rejected, and a rejected file yields no Config for the caller to misuse.
func TestLoadRejectsMalformedLines(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "line without a separator", input: "name = app\nbroken\n"},
		{name: "separator but no key", input: "= app\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := config.Load(strings.NewReader(c.input))
			if err == nil {
				t.Fatalf("Load(%q) = nil error, want a parse error", c.input)
			}
			if cfg != nil {
				t.Errorf("Load(%q) returned a non-nil Config on error", c.input)
			}
		})
	}
}
