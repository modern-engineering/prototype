package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Example_display shows the package's motivating scenario, straight from
// its doc comment: shortening a string for a narrow display column
// without splitting the multi-byte é in two, unlike a raw byte slice.
func Example_display() {
	// "héllo" is five runes but six bytes; Head counts runes, so the é
	// survives intact in the first two instead of being split mid-byte.
	fmt.Println(truncate.Head("héllo", 2))
	// The last two runes, not the last two bytes: a byte cut here would
	// land inside é and mangle it.
	fmt.Println(truncate.Tail("héllo", 2))
	// Output:
	// hé
	// lo
}

func TestHeadAndTailHonorLengthAndRuneBoundaries(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "héllo", n: 10, head: "héllo", tail: "héllo"}, // n exceeds a multi-byte string's rune count
		{s: "hello", n: 0, head: "", tail: ""},
		{s: "hello", n: 5, head: "hello", tail: "hello"},
		{s: "hello", n: 8, head: "hello", tail: "hello"},
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

// Both doc comments promise that a negative n is treated as zero,
// returning the empty string. Tail honors that promise; Head panics
// instead.
func TestHeadAndTailTreatNegativeNAsZero(t *testing.T) {
	const s = "hello"
	if got := callHead(t, s, -1); got != "" {
		t.Errorf("Head(%q, -1) = %q, want \"\" per doc comment", s, got)
	}
	if got := truncate.Tail(s, -1); got != "" {
		t.Errorf("Tail(%q, -1) = %q, want \"\"", s, got)
	}
}

// callHead calls Head and turns an unexpected panic into a t.Errorf
// naming the panic value, so a doc/implementation mismatch surfaces as
// an ordinary test failure.
func callHead(t *testing.T, s string, n int) (result string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, %d) panicked: %v", s, n, r)
		}
	}()
	return truncate.Head(s, n)
}
