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

// sample exercises every line shape Load must handle: a leading blank
// line, a comment, and both spaced and unspaced "key = value" pairs, so
// the promised whitespace-trimming is actually proven.
const sample = `
name=app
  port = 8080
timeout=2s
# retries controls how often we try again
retries = 3
`

// A typical caller loads a config once, then reads every setting back
// through its typed accessor with a per-field default, so a config file
// that omits a setting never stops the program.
func Example_typicalUsage() {
	cfg, err := config.Load(strings.NewReader(`
name = payments
port = 8080
timeout = 2s
`))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.String("name", "unnamed"))
	fmt.Println(cfg.Int("port", 80))
	fmt.Println(cfg.Duration("timeout", time.Second))
	fmt.Println(cfg.Int("workers", 4)) // absent from the file: falls back to the default

	// Output:
	// payments
	// 8080
	// 2s
	// 4
}

// The whole-package scenario: parse a config, then read every field back
// through its typed accessor, with defaults standing in for whatever the
// file leaves out or a value that fails to parse. "port" doubles as an
// invalid duration and "name" as an invalid integer, so the parse-failure
// fallback is proven without inventing dedicated bad values.
func TestConfig(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load(sample) = %v, want nil", err)
	}

	if got, want := cfg.Keys(), []string{"name", "port", "retries", "timeout"}; !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}

	stringCases := []struct {
		key, def, want string
	}{
		{key: "name", def: "unnamed", want: "app"},
		{key: "missing", def: "unnamed", want: "unnamed"},
	}
	for _, c := range stringCases {
		if got := cfg.String(c.key, c.def); got != c.want {
			t.Errorf("String(%q, %q) = %q, want %q", c.key, c.def, got, c.want)
		}
	}

	intCases := []struct {
		key  string
		def  int
		want int
	}{
		{key: "port", def: 0, want: 8080},
		{key: "missing", def: 42, want: 42},
		{key: "name", def: 7, want: 7}, // "app" does not parse as an integer
	}
	for _, c := range intCases {
		if got := cfg.Int(c.key, c.def); got != c.want {
			t.Errorf("Int(%q, %d) = %d, want %d", c.key, c.def, got, c.want)
		}
	}

	durationCases := []struct {
		key  string
		def  time.Duration
		want time.Duration
	}{
		{key: "timeout", def: 0, want: 2 * time.Second},
		{key: "missing", def: time.Second, want: time.Second},
		{key: "port", def: time.Minute, want: time.Minute}, // "8080" carries no unit
	}
	for _, c := range durationCases {
		if got := cfg.Duration(c.key, c.def); got != c.want {
			t.Errorf("Duration(%q, %v) = %v, want %v", c.key, c.def, got, c.want)
		}
	}
}

// Every way a line can fail to be a "key = value" pair: the
// positioned-error half of Load's contract that the rest of the suite
// never exercises. Load's errors are plain fmt.Errorf values with no
// sentinel or typed line field, so this can only confirm an error
// occurs, not pin its wording; that gap is worth closing in the package
// itself, not papering over with string matching.
func TestLoadRejectsMalformedLines(t *testing.T) {
	cases := []struct {
		line, invalidity string
	}{
		{line: "just text", invalidity: "no '=' separator"},
		{line: "= value", invalidity: "an empty key"},
	}
	for _, c := range cases {
		if _, err := config.Load(strings.NewReader(c.line)); err == nil {
			t.Errorf("Load(%q) = nil error, want an error for %s", c.line, c.invalidity)
		}
	}
}

// BenchmarkLoad measures how fast Load parses the sample.
func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		config.Load(strings.NewReader(sample))
	}
}
