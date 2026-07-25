package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A UI trims a display name to fit a fixed-width badge. Because the
// helpers count runes rather than bytes, a multi-byte character is never
// split, so the badge always renders as valid UTF-8.
func Example() {
	// "Zoë Saldaña" is eleven runes; Head keeps the first eight.
	fmt.Println(truncate.Head("Zoë Saldaña", 8))

	// A limit longer than the input leaves it untouched, so callers need
	// not measure the string first.
	fmt.Println(truncate.Head("OK", 8))

	// Tail keeps the end instead: the last three runes of a word.
	fmt.Println(truncate.Tail("café", 3))

	// Output:
	// Zoë Sald
	// OK
	// afé
}

// The documented boundaries shared by Head and Tail: an empty input, a
// limit past the end that returns the input whole, a limit equal to the
// rune count, a zero limit, and a real cut that must land on a rune
// boundary rather than mid-sequence. The two functions mirror each other,
// so one table carries both expectations per case.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "", n: 3, head: "", tail: ""},
		{s: "héllo", n: 10, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 0, head: "", tail: ""},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "a🌍b", n: 2, head: "a🌍", tail: "🌍b"},
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

// Both doc comments promise that a negative count is treated as zero and
// yields the empty string. Tail guards for this; Head does not, so a
// negative bound reaches its slice expression and panics. Stating the
// shared contract makes that gap a failing test rather than a latent
// crash: restore Head's guard, or amend its doc if the panic is intended.
func TestNegativeCountYieldsEmptyString(t *testing.T) {
	const s = "héllo"

	if got := truncate.Tail(s, -1); got != "" {
		t.Errorf("Tail(%q, -1) = %q, want the empty string", s, got)
	}

	// Recover turns Head's missing-guard panic into a clean failure so the
	// rest of the suite still reports.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, -1) panicked (%v), want the empty string per its doc", s, r)
		}
	}()
	if got := truncate.Head(s, -1); got != "" {
		t.Errorf("Head(%q, -1) = %q, want the empty string", s, got)
	}
}
