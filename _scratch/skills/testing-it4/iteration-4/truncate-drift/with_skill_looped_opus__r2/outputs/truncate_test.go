package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

func Example() {
	// Shortening a long path for display keeps the leading and trailing
	// segments and elides the middle with an ellipsis. Both kept segments
	// carry multi-byte characters, so counting runes rather than bytes leaves
	// every glyph intact.
	const path = "café-münchen-señor"
	// Four runes spans the first segment "café"; five spans the last "señor".
	fmt.Println(truncate.Head(path, 4) + "…" + truncate.Tail(path, 5))
	// Output:
	// café…señor
}

func ExampleHead() {
	// A two-rune limit on "héllo" keeps the accented "é" whole rather than
	// splitting its two-byte UTF-8 encoding: Head counts runes, not bytes.
	fmt.Println(truncate.Head("héllo", 2))
	// A limit past the end returns the input unchanged, so a caller may pass a
	// generous bound without measuring the string first.
	fmt.Println(truncate.Head("café", 10))
	// Output:
	// hé
	// café
}

func ExampleTail() {
	// Tail counts from the other end: the last two runes of "héllo".
	fmt.Println(truncate.Tail("héllo", 2))
	// Like Head, a limit past the end returns the input unchanged.
	fmt.Println(truncate.Tail("café", 10))
	// Output:
	// lo
	// café
}

// The package earns its name by counting runes, not bytes: a multi-byte glyph
// must survive the cut whole, an over-generous limit must return the input
// untouched so callers need not measure first, and the exact-length, empty,
// and zero edges must not fall off by one. Head and Tail mirror each other, so
// one table states both expectations per input.
//
// The final row exercises the shared promise that a negative limit behaves
// like zero and yields the empty string. Tail enforces it with an n <= 0
// guard; Head has none and reaches a negative slice bound, so Head(s, n<0)
// panics. That row's Head assertion therefore fails today, holding Head to its
// published contract rather than ratifying the panic.
func TestHeadAndTailCountRunes(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{"hello", 3, "hel", "llo"},
		{"héllo", 2, "hé", "lo"},
		{"café", 3, "caf", "afé"},
		{"a😀b", 2, "a😀", "😀b"},
		{"café", 10, "café", "café"},
		{"café", 4, "café", "café"},
		{"", 3, "", ""},
		{"café", 0, "", ""},
		{"café", -5, "", ""}, // negative n: Tail guards it; Head panics without the guard.
	}
	for _, c := range cases {
		if got := headRecovered(t, c.s, c.n); got != c.head {
			t.Errorf("Head(%q, %d) = %q, want %q", c.s, c.n, got, c.head)
		}
		if got := truncate.Tail(c.s, c.n); got != c.tail {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.s, c.n, got, c.tail)
		}
	}
}

// headRecovered calls Head and turns a panic into a reported failure instead of
// unwinding the test binary. Head's doc promises a return value for every n, so
// a panic is itself the drift under test, not a reason to abandon the remaining
// rows.
func headRecovered(t *testing.T, s string, n int) (result string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, %d) panicked (%v), want %q", s, n, r, "")
		}
	}()
	return truncate.Head(s, n)
}
