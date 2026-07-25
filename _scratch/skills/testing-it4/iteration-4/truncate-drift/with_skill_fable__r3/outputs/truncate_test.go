package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A terminal column is eight characters wide. Byte slicing would cut
// "wörld" in the middle of the ö and leave a broken byte on screen;
// Head counts runes, so the cut always lands between characters.
func Example() {
	greeting := "héllo, wörld"
	fmt.Println(truncate.Head(greeting, 8))
	// Tail keeps the end instead, handy for the interesting suffix of
	// a long identifier.
	fmt.Println(truncate.Tail(greeting, 5))
	// A generous limit is safe: shorter inputs come back unchanged, so
	// there is no need to measure first.
	fmt.Println(truncate.Head(greeting, 80))
	// Output:
	// héllo, w
	// wörld
	// héllo, wörld
}

// Head and Tail share one contract, differing only in which end of the
// string survives, so every case drives both from the same inputs.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "hello", n: 5, head: "hello", tail: "hello"},
		{s: "hello", n: 10, head: "hello", tail: "hello"}, // a generous limit returns s unchanged
		{s: "hello", n: 0, head: "", tail: ""},
		{s: "", n: 4, head: "", tail: ""},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},                          // the package doc's own example: two runes, three bytes
		{s: "日本語のテキスト", n: 3, head: "日本語", tail: "キスト"},                     // three-byte runes
		{s: "a\U0001F642b", n: 2, head: "a\U0001F642", tail: "\U0001F642b"}, // a four-byte rune survives the cut
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

// Both doc comments promise that a negative count is treated as zero,
// and Tail delivers by guarding n <= 0 before slicing. Head has no such
// guard, so a negative count slices out of range and panics. The prose
// and Tail's parallel guard read as one intent, so this test holds Head
// to the documented behavior and fails, loudly, until Head gains the
// guard or the owner rules that the docs should promise less.
func TestNegativeCountMeansEmptyString(t *testing.T) {
	if got := truncate.Tail("hello", -1); got != "" {
		t.Errorf("Tail(%q, -1) = %q, want %q", "hello", got, "")
	}
	if got := truncate.Head("hello", -1); got != "" {
		t.Errorf("Head(%q, -1) = %q, want %q", "hello", got, "")
	}
}
