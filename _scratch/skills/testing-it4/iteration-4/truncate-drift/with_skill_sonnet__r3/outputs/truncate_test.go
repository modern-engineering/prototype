package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Shortening a UI label to a display budget is the anticipated call:
// Head or Tail cuts by rune count, so a multi-byte character sitting at
// the cut point comes through whole instead of as a mangled trailing byte.
func Example() {
	fmt.Println(truncate.Head("héllo", 2))
	fmt.Println(truncate.Tail("héllo", 2))
	// Output:
	// hé
	// lo
}

// Head and Tail mirror each other over every n that both docs promise
// the same thing for: a cut that lands mid-rune still returns whole
// runes, and a limit at or beyond the input's rune count passes it
// through unmeasured.
func TestHeadAndTailStayRuneSafe(t *testing.T) {
	cases := []struct {
		s, head, tail string
		n             int
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "héllo", n: 3, head: "hél", tail: "llo"},
		{s: "hi", n: 10, head: "hi", tail: "hi"},
		{s: "hello", n: 0, head: "", tail: ""},
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

// Head's and Tail's doc comments promise the same negative-n contract in
// the same words, "A negative n is treated as zero", so both are held to
// it here rather than to Tail's code, which is the only one of the two
// that actually special-cases n <= 0.
func TestNegativeNIsTreatedAsZero(t *testing.T) {
	if got := truncate.Tail("hello", -3); got != "" {
		t.Errorf("Tail(%q, %d) = %q, want %q", "hello", -3, got, "")
	}
	if got := truncate.Head("hello", -3); got != "" {
		t.Errorf("Head(%q, %d) = %q, want %q", "hello", -3, got, "")
	}
}
