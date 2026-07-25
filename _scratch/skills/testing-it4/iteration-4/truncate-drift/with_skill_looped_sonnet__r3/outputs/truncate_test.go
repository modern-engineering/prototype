package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A UI abbreviates a long display name to fit a narrow column, keeping
// both ends and dropping the middle. Names carry accented letters like ë,
// so the cut has to land on a whole rune rather than a raw byte offset.
func Example_abbreviateName() {
	const name = "Zoë Bergström"
	// Keep 3 runes from each end: "Zoë" ends right after the accented
	// third rune instead of splitting its two-byte encoding.
	fmt.Println(truncate.Head(name, 3) + "…" + truncate.Tail(name, 3))
	// Output:
	// Zoë…röm
}

// Both functions promise the same shape of contract: keep the requested
// runes from one end, and hand the string back whole once there's
// nothing left to trim. The é in "héllo" holds two bytes, so slicing by
// byte index would land inside the character; that case would catch the
// mistake.
func TestHeadAndTailKeepWholeRunes(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "banana", n: 2, head: "ba", tail: "na"},
		{s: "hi", n: 5, head: "hi", tail: "hi"},
		{s: "abc", n: 3, head: "abc", tail: "abc"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "banana", n: 0, head: "", tail: ""},
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

// Both doc comments promise a negative n is treated as zero. Tail
// honors that; Head instead slices with a negative index and panics.
func TestNegativeNIsTreatedAsZero(t *testing.T) {
	const s = "hello"
	t.Run("Tail", func(t *testing.T) {
		if got := truncate.Tail(s, -1); got != "" {
			t.Errorf("Tail(%q, -1) = %q, want %q", s, got, "")
		}
	})
	t.Run("Head", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Head(%q, -1) panicked: %v, want %q per doc", s, r, "")
			}
		}()
		if got := truncate.Head(s, -1); got != "" {
			t.Errorf("Head(%q, -1) = %q, want %q", s, got, "")
		}
	})
}
