// Package truncate_test pins the contract package truncate documents:
// rune-safe Head and Tail truncation that never splits a multi-byte
// character, with negative n treated as zero.
package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// ExampleHead documents the package's headline promise, taken directly
// from the package doc: cutting by rune count instead of byte count
// keeps a multi-byte character out of the cut, unlike a byte slice.
func ExampleHead() {
	fmt.Println(truncate.Head("héllo", 2))
	// Output: hé
}

// ExampleTail mirrors ExampleHead at the other end of the string: the
// multi-byte é lands inside the kept suffix and survives intact.
func ExampleTail() {
	fmt.Println(truncate.Tail("héllo", 4))
	// Output: éllo
}

// TestHead pins Head's documented contract: n at or past the rune
// count returns s unchanged, a negative n is documented to come back
// as the empty string, and otherwise the first n runes are kept
// without splitting a multi-byte character.
func TestHead(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		// Foundational special cases the rest of the table builds on.
		{s: "héllo", n: -1, want: ""},     // doc: negative n is treated as zero
		{s: "héllo", n: 0, want: ""},      // n=0 keeps nothing
		{s: "", n: 3, want: ""},           // empty input has fewer than n runes
		{s: "héllo", n: 5, want: "héllo"}, // n equal to the rune count
		{s: "héllo", n: 9, want: "héllo"}, // n well past the rune count
		// The documented rune-safe cut, including the package doc's own example.
		{s: "héllo", n: 2, want: "hé"},
		{s: "héllo", n: 1, want: "h"},
	}
	for _, c := range cases {
		got := headRecovered(t, c.s, c.n)
		if got != c.want {
			t.Errorf("Head(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

// headRecovered calls Head and turns any panic into a table-row test
// failure instead of crashing the whole run, so a single bad row
// (n=-1 currently panics rather than returning "" as documented)
// still lets the rest of the table report its results.
func headRecovered(t *testing.T, s string, n int) (got string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, %d) panicked: %v", s, n, r)
		}
	}()
	return truncate.Head(s, n)
}

// TestTail pins Tail's documented contract: n at or past the rune
// count returns s unchanged, a negative n comes back as the empty
// string, and otherwise the last n runes are kept without splitting a
// multi-byte character.
func TestTail(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		// Foundational special cases the rest of the table builds on.
		{s: "héllo", n: -1, want: ""},     // doc: negative n is treated as zero
		{s: "héllo", n: 0, want: ""},      // n=0 keeps nothing
		{s: "", n: 3, want: ""},           // empty input has fewer than n runes
		{s: "héllo", n: 5, want: "héllo"}, // n equal to the rune count
		{s: "héllo", n: 9, want: "héllo"}, // n well past the rune count
		// The documented rune-safe cut, matching ExampleTail above.
		{s: "héllo", n: 2, want: "lo"},
		{s: "héllo", n: 4, want: "éllo"},
	}
	for _, c := range cases {
		got := truncate.Tail(c.s, c.n)
		if got != c.want {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}
