package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Example demonstrates the package's anticipated use: shortening a long,
// accented display name for a narrow UI while keeping both ends
// recognizable and every rune intact.
func Example() {
	name := "Bjørn Åström"
	fmt.Println(truncate.Head(name, 5) + "…" + truncate.Tail(name, 6))
	// Output:
	// Bjørn…Åström
}

// TestHeadAndTailCountRunes holds Head and Tail to the contract they
// share: both count runes rather than bytes, both pass a short input
// through unchanged, and both clamp to the empty string at n=0.
func TestHeadAndTailCountRunes(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"}, // cut lands right after a two-byte rune
		{s: "hello", n: 0, head: "", tail: ""},
		{s: "hello", n: 5, head: "hello", tail: "hello"},
		{s: "hello", n: 9, head: "hello", tail: "hello"}, // n beyond len(s): unchanged
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

// TestNegativeNIsZero checks the doc's promise that a negative n is
// treated as zero. Tail keeps that promise; Head does not (see
// report.md), so the Head half is wrapped in a recover to turn the
// drift into a reported test failure instead of a crashed test binary.
func TestNegativeNIsZero(t *testing.T) {
	if got := truncate.Tail("hello", -1); got != "" {
		t.Errorf("Tail(%q, -1) = %q, want \"\" per doc", "hello", got)
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Head(%q, -1) panicked (%v), want \"\" per doc", "hello", r)
			}
		}()
		if got := truncate.Head("hello", -1); got != "" {
			t.Errorf("Head(%q, -1) = %q, want \"\" per doc", "hello", got)
		}
	}()
}
