package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A caller shortening a label for a narrow column: keep the leading
// runes and mark the cut with an ellipsis. "café" stays intact because
// Head counts runes, not bytes.
func Example_preview() {
	name := "café society"
	// Head returns name unchanged once it already has 4 runes or fewer,
	// so skip the ellipsis rather than appending one to an untruncated
	// string.
	if short := truncate.Head(name, 4); short != name {
		fmt.Println(short + "…")
	}
	// Output:
	// café…
}

// The doc comments for Head and Tail describe the same three behaviors
// in the same words, so one table drives both: a cut in range, a limit
// beyond the string's length passed through unchanged, and a cut that
// lands on either side of a multi-byte rune.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "abc", n: 3, head: "abc", tail: "abc"}, // n equals the rune count exactly
		{s: "hi", n: 5, head: "hi", tail: "hi"},    // n exceeds the rune count
		{s: "hello", n: 0, head: "", tail: ""},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},     // cut through the multi-byte rune from the head
		{s: "héllo", n: 4, head: "héll", tail: "éllo"}, // cut through the multi-byte rune from the tail
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

// Tail's doc comment promises that a negative n behaves like zero, and
// the guard backs the promise up.
func TestTailTreatsNegativeNAsZero(t *testing.T) {
	if got := truncate.Tail("hello", -1); got != "" {
		t.Errorf("Tail(%q, %d) = %q, want %q per doc", "hello", -1, got, "")
	}
}

// Head's doc comment makes the same promise as Tail's, but Head has no
// matching guard: the call below panics rather than returning "". Left
// failing rather than papered over with a recover, so the drift stays
// visible until Head gets Tail's guard or the doc comment is corrected.
func TestHeadPanicsWhereDocPromisesZero(t *testing.T) {
	if got := truncate.Head("hello", -1); got != "" {
		t.Errorf("Head(%q, %d) = %q, want %q per doc", "hello", -1, got, "")
	}
}
