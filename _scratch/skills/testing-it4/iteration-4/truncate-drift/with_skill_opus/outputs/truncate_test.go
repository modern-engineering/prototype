package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Truncating by runes rather than bytes keeps multi-byte characters whole, so
// a label clipped to fit a fixed width stays valid UTF-8 on both ends.
func Example() {
	const label = "héllo, wörld"
	fmt.Println(truncate.Head(label, 5))
	fmt.Println(truncate.Tail(label, 5))
	// Output:
	// héllo
	// wörld
}

// Head and Tail are mirror images, so one table pins both: each cuts on a rune
// boundary rather than a byte, hands back the input untouched when the limit
// outruns its length, and empties at a zero limit.
func TestHeadAndTail(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "héllö", n: 2, head: "hé", tail: "lö"},
		{s: "héllö", n: 5, head: "héllö", tail: "héllö"},
		{s: "héllö", n: 0, head: "", tail: ""},
		{s: "hi", n: 5, head: "hi", tail: "hi"},
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

// The package doc promises a negative count is treated as zero and yields the
// empty string, so a caller may pass an unclamped length without a bounds
// check of their own. Tail honors that; Head panics on a negative count
// instead, so this stays red until Head agrees with the documented contract.
func TestNegativeCountReturnsEmpty(t *testing.T) {
	const s = "héllo"
	wantEmpty(t, "Head", truncate.Head, s, -1)
	wantEmpty(t, "Tail", truncate.Tail, s, -1)
}

// wantEmpty fails unless trunc(s, n) returns the empty string the doc promises
// for a non-positive count; a panic is reported as that same failure so one
// broken function does not abort the whole run.
func wantEmpty(t *testing.T, name string, trunc func(string, int) string, s string, n int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s(%q, %d) panicked (%v), want %q", name, s, n, r, "")
		}
	}()
	if got := trunc(s, n); got != "" {
		t.Errorf("%s(%q, %d) = %q, want %q", name, s, n, got, "")
	}
}
