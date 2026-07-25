package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Head trims a label to a column width, counting runes so a multi-byte
// character is never split at the cut. A width past the end of the string
// returns it unchanged, so a caller may pass a generous limit without
// measuring the input first.
func ExampleHead() {
	fmt.Println(truncate.Head("café-münchen", 4))    // first 4 runes: "café", which is 5 bytes
	fmt.Println(truncate.Head("café-münchen", 1000)) // width past the end: unchanged
	// Output:
	// café
	// café-münchen
}

// Tail keeps a label's trailing runes, the mirror of Head, again counting
// runes rather than bytes so the kept text stays valid UTF-8. A width past
// the end returns the string unchanged.
func ExampleTail() {
	fmt.Println(truncate.Tail("café-münchen", 7))    // last 7 runes: "münchen"
	fmt.Println(truncate.Tail("café-münchen", 1000)) // width past the end: unchanged
	// Output:
	// münchen
	// café-münchen
}

// Head and Tail mirror each other, so a single table of shared inputs
// exercises both: a cut inside the string, a clamp back to the whole string
// when the count overshoots, and the empty string at zero. The multi-byte
// rows carry the package's headline promise that a count is runes, not
// bytes: "héllo" is five runes across six bytes.
func TestTruncatesOnRuneBoundaries(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 9, head: "héllo", tail: "héllo"},
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

// Both functions document that a negative count is treated as zero and
// returns the empty string, so a caller may forward an unmeasured,
// possibly negative width without a guard of its own. Tail honors that
// promise; Head panics on the negative slice bound instead. The Head case
// is left red to surface that divergence from the documented contract
// rather than bless it.
func TestNegativeCountYieldsEmptyString(t *testing.T) {
	const s = "héllo"

	t.Run("Tail", func(t *testing.T) {
		if got := truncate.Tail(s, -1); got != "" {
			t.Errorf("Tail(%q, -1) = %q, want the empty string", s, got)
		}
	})

	t.Run("Head", func(t *testing.T) {
		// A panic here is the documented contract going unmet, not a crashed
		// binary, so recover and report it as an ordinary failure.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Head(%q, -1) panicked (%v), want the empty string", s, r)
			}
		}()
		if got := truncate.Head(s, -1); got != "" {
			t.Errorf("Head(%q, -1) = %q, want the empty string", s, got)
		}
	})
}
