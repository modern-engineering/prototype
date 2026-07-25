package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Head keeps the first n runes, counting characters rather than bytes, so a
// multi-byte rune is never split into a mangled half. A limit larger than
// the input is not an error: the whole string comes back unchanged.
func ExampleHead() {
	// "héllo" is five runes but six bytes; asking for two runes keeps "hé"
	// whole, where the byte slice "héllo"[:2] would cut é in half.
	fmt.Println(truncate.Head("héllo", 2))

	// A generous limit needs no measuring first: given fewer runes than
	// asked for, Head returns the input as it stands.
	fmt.Println(truncate.Head("café", 10))
	// Output:
	// hé
	// café
}

// Tail keeps the last n runes, counting characters rather than bytes, so a
// multi-byte rune is never split into a mangled half. A limit larger than
// the input is not an error: the whole string comes back unchanged.
func ExampleTail() {
	// "café" is four runes but five bytes; asking for three runes keeps the
	// trailing "afé" whole rather than splitting é.
	fmt.Println(truncate.Tail("café", 3))

	// A generous limit needs no measuring first: given fewer runes than
	// asked for, Tail returns the input as it stands.
	fmt.Println(truncate.Tail("hi", 10))
	// Output:
	// afé
	// hi
}

// Head and Tail mirror each other, so one row states both halves of a
// cut. The cases show that a multi-byte rune is counted once and never
// split, and that a limit at or beyond the input's rune count returns
// the input unchanged, letting a caller pass a generous limit without
// measuring first.
func TestTruncatesToRuneCount(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "café", n: 3, head: "caf", tail: "afé"},
		{s: "café", n: 4, head: "café", tail: "café"},
		{s: "hi", n: 9, head: "hi", tail: "hi"},
		{s: "héllo", n: 0, head: "", tail: ""},
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

// The doc promises both helpers treat a negative n as zero and return the
// empty string, so a caller need not clamp the length first. Tail honors
// that; Head panics instead, so this fails until Head grows the guard Tail
// already has. Siding with the prose, the golden contract, over the code.
func TestNegativeLimitYieldsEmptyString(t *testing.T) {
	if got := truncate.Tail("héllo", -1); got != "" {
		t.Errorf(`Tail("héllo", -1) = %q, want ""`, got)
	}

	// Head panics on a negative n today, though the doc promises "".
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(`Head("héllo", -1) panicked (%v), want ""`, r)
		}
	}()
	if got := truncate.Head("héllo", -1); got != "" {
		t.Errorf(`Head("héllo", -1) = %q, want ""`, got)
	}
}
