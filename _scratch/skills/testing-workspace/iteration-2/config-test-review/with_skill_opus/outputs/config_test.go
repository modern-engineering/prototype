package config

import (
	"errors"
	"fmt"
	"log"
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

func Example() {
	cfg, err := Load(strings.NewReader("name = app\nport = 8080\ntimeout = 2s\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.String("name", "default"))
	fmt.Println(cfg.Int("port", 80))
	fmt.Println(cfg.Duration("timeout", time.Second))
	fmt.Println(cfg.Keys())
	// Output:
	// app
	// 8080
	// 2s
	// [name port timeout]
}

func TestConfig(t *testing.T) {
	cfg, err := Load(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := cfg.String("name", "fallback"); got != "app" {
		t.Errorf("String(\"name\") = %q, want %q", got, "app")
	}
	if got := cfg.String("missing", "fallback"); got != "fallback" {
		t.Errorf("String(\"missing\") = %q, want %q", got, "fallback")
	}

	if got := cfg.Int("port", 0); got != 8080 {
		t.Errorf("Int(\"port\") = %d, want %d", got, 8080)
	}
	if got := cfg.Int("missing", 42); got != 42 {
		t.Errorf("Int(\"missing\") = %d, want %d", got, 42)
	}
	// "app" is not an integer, so an unparsable value falls back to def.
	if got := cfg.Int("name", 7); got != 7 {
		t.Errorf("Int(\"name\") = %d, want %d", got, 7)
	}

	if got := cfg.Duration("timeout", 0); got != 2*time.Second {
		t.Errorf("Duration(\"timeout\") = %v, want %v", got, 2*time.Second)
	}
	if got := cfg.Duration("missing", time.Second); got != time.Second {
		t.Errorf("Duration(\"missing\") = %v, want %v", got, time.Second)
	}
	// "app" is not a duration, so an unparsable value falls back to def.
	if got := cfg.Duration("name", time.Minute); got != time.Minute {
		t.Errorf("Duration(\"name\") = %v, want %v", got, time.Minute)
	}

	// Comments and blank lines contribute no keys; the rest are sorted.
	want := []string{"name", "port", "retries", "timeout"}
	if got := cfg.Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

// A malformed line is reported as a *SyntaxError carrying its 1-based line.
func TestMalformedLines(t *testing.T) {
	for _, tt := range []struct {
		input string
		line  int
	}{
		{"good = 1\nbad line\n", 2},
		{"= value\n", 1},
		{"name = app\n\t= oops\n", 2},
	} {
		_, err := Load(strings.NewReader(tt.input))
		var se *SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("Load(%q) error = %v, want *SyntaxError", tt.input, err)
			continue
		}
		if se.Line != tt.line {
			t.Errorf("Load(%q) reported line %d, want %d", tt.input, se.Line, tt.line)
		}
	}
}

func BenchmarkLoad(b *testing.B) {
	for b.Loop() {
		Load(strings.NewReader(sample))
	}
}
