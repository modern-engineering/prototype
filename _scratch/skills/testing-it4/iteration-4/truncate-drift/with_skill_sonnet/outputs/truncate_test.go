package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A narrow status line keeps a recognizable prefix and suffix of a long
// identifier instead of measuring runes by hand at the call site.
func Example() {
	id := "request-8f3ce1a9-payload-checksum-mismatch"
	fmt.Printf("%s...%s\n", truncate.Head(id, 12), truncate.Tail(id, 8))
	// Output:
	// request-8f3c...mismatch
}

// Both functions promise the same cut: exactly n runes, counted rune by
// rune rather than byte by byte, so a multi-byte case sits beside the
// plain-ASCII ones.
func TestCutsExactlyNRunes(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 0, head: "", tail: ""},
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
	}
	for _, c := range cases {
		if got := truncate.Head(c.s, c.n); got != c.head {
			t.Errorf("Head(%q, %d) = %q, want %q", c.s, c.n, got, c.head)
		}
		if got := truncate.Tail(c.s, c.n); got != c.tail {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.s, c.n, got, c.tail)
		}
	}
}

// The documented shortcut: a caller may pass n larger than the string's
// rune count without measuring it first, and get the whole string back.
func TestGenerousLimitReturnsInputUnchanged(t *testing.T) {
	cases := []struct {
		s string
		n int
	}{
		{s: "hello", n: 5},
		{s: "hello", n: 8},
		{s: "héllo", n: 100},
	}
	for _, c := range cases {
		if got := truncate.Head(c.s, c.n); got != c.s {
			t.Errorf("Head(%q, %d) = %q, want %q unchanged", c.s, c.n, got, c.s)
		}
		if got := truncate.Tail(c.s, c.n); got != c.s {
			t.Errorf("Tail(%q, %d) = %q, want %q unchanged", c.s, c.n, got, c.s)
		}
	}
}

// The doc for both functions promises a negative n is treated as zero.
// Tail special-cases n <= 0 and honors that; Head slices with the raw n
// and panics on a negative index instead. Siding with the documented
// contract per the maintainer's drift policy: this holds Head to the
// promise and recovers so the gap surfaces as a clean failure rather than
// a crashed test binary.
func TestNegativeNIsTreatedAsZero(t *testing.T) {
	if got := truncate.Tail("hello", -1); got != "" {
		t.Errorf("Tail(%q, %d) = %q, want %q", "hello", -1, got, "")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, %d) panicked (%v), want %q per package doc", "hello", -1, r, "")
		}
	}()
	if got := truncate.Head("hello", -1); got != "" {
		t.Errorf("Head(%q, %d) = %q, want %q", "hello", -1, got, "")
	}
}
