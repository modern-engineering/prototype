package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Fixed-width displays cut long values at a character budget: keep the
// start of a title, keep the end of a path, and never split a rune.
func Example_display() {
	// The budget counts runes, not bytes: five runes of "héllo wörld"
	// survive the cut even though é occupies two bytes.
	fmt.Println(truncate.Head("héllo wörld", 5))
	// Keep the end of a path, where the interesting part lives.
	fmt.Println(truncate.Tail("logs/2026-07-23/access.log", 10))
	// A value already within budget passes through unchanged, so a
	// generous limit is safe without measuring the input first.
	fmt.Println(truncate.Head("héllo", 40))
	// Output:
	// héllo
	// access.log
	// héllo
}

// An implementation that slices by byte index fails the multi-byte
// cases outright instead of silently mangling the cut.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		// Plain ASCII, where runes and bytes coincide.
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		// The documented example: two runes, three bytes.
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		// The cut lands beside the two-byte é on each side.
		{s: "héllo", n: 4, head: "héll", tail: "éllo"},
		// Three-byte runes only: eight runes, twenty-four bytes.
		{s: "日本語のテキスト", n: 3, head: "日本語", tail: "キスト"},
		// A four-byte emoji straddling the cut point.
		{s: "a👍b", n: 2, head: "a👍", tail: "👍b"},
		// A generous limit returns the input unchanged.
		{s: "hi", n: 5, head: "hi", tail: "hi"},
		// Exactly n runes is also unchanged.
		{s: "hi", n: 2, head: "hi", tail: "hi"},
		// Zero keeps nothing.
		{s: "hello", n: 0, head: "", tail: ""},
		// The empty string survives any limit.
		{s: "", n: 3, head: "", tail: ""},
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

// A negative budget is documented to act as zero, and callers get one
// naturally from width arithmetic ("width minus used" dips below zero),
// so a panic here would take the caller down. Today Tail keeps that
// promise but Head panics on the negative slice bound, which reads as
// an accidental omission rather than a contract change, so this test
// sides with the docs and fails until the owner either gives Head the
// guard Tail has or amends Head's doc comment. The recover turns the
// panic into a readable failure instead of an aborted run.
func TestNegativeCountActsAsZero(t *testing.T) {
	t.Run("Head", func(t *testing.T) {
		defer func() {
			if p := recover(); p != nil {
				t.Errorf(`Head("héllo", -1) panicked: %v, want ""`, p)
			}
		}()
		if got := truncate.Head("héllo", -1); got != "" {
			t.Errorf(`Head("héllo", -1) = %q, want ""`, got)
		}
	})
	t.Run("Tail", func(t *testing.T) {
		defer func() {
			if p := recover(); p != nil {
				t.Errorf(`Tail("héllo", -1) panicked: %v, want ""`, p)
			}
		}()
		if got := truncate.Tail("héllo", -1); got != "" {
			t.Errorf(`Tail("héllo", -1) = %q, want ""`, got)
		}
	})
}
