package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// ExampleHead shows the behavior a caller relies on: the cut lands on a
// rune boundary (so the multi-byte "é" survives whole), and a limit past
// the end of the string returns the string untouched. It doubles as the
// example printed in the package docs.
func ExampleHead() {
	fmt.Println(truncate.Head("héllo", 2))  // first two runes, not two bytes
	fmt.Println(truncate.Head("héllo", 10)) // limit past the end: unchanged
	// Output:
	// hé
	// héllo
}

// ExampleTail is the trailing-edge twin of ExampleHead, documenting that
// Tail counts runes from the end and likewise leaves an over-long limit
// alone.
func ExampleTail() {
	fmt.Println(truncate.Tail("héllo", 2))  // last two runes
	fmt.Println(truncate.Tail("héllo", 10)) // limit past the end: unchanged
	// Output:
	// lo
	// héllo
}

// TestHeadKeepsLeadingRunes pins Head's whole contract for non-negative n
// in one table. The CJK row is the load-bearing case: those characters are
// three bytes each, so a byte-index cut would mangle them, and only a
// rune-counting implementation returns the first three intact. The rest of
// the table pins the documented edges: the n == rune-count boundary, the
// generous-limit passthrough, zero, and empty input.
func TestHeadKeepsLeadingRunes(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"日本語テスト", 3, "日本語"}, // runes, not the three-byte-per-char encoding
		{"hello", 3, "hel"},
		{"héllo", 5, "héllo"},  // n equals the rune count: the whole string
		{"héllo", 10, "héllo"}, // generous limit: returned unchanged
		{"anything", 0, ""},    // zero runes requested
		{"", 4, ""},            // empty input, generous limit
	}
	for _, tt := range tests {
		if got := truncate.Head(tt.s, tt.n); got != tt.want {
			t.Errorf("Head(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

// TestTailKeepsTrailingRunes mirrors TestHeadKeepsLeadingRunes for Tail and
// carries one extra row Head cannot: the negative-n case. Tail's docs make
// the same "negative n is treated as zero" promise Head's do, and Tail
// keeps it, so the promise is pinned here where it holds. The Head side of
// that promise lives in TestHeadTreatsNegativeCountAsZero.
func TestTailKeepsTrailingRunes(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"日本語テスト", 3, "テスト"}, // trailing runes, not trailing bytes
		{"hello", 3, "llo"},
		{"héllo", 5, "héllo"},  // n equals the rune count: the whole string
		{"héllo", 10, "héllo"}, // generous limit: returned unchanged
		{"anything", 0, ""},    // zero runes requested
		{"héllo", -1, ""},      // negative n treated as zero
		{"", 4, ""},            // empty input, generous limit
	}
	for _, tt := range tests {
		if got := truncate.Tail(tt.s, tt.n); got != tt.want {
			t.Errorf("Tail(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

// TestHeadTreatsNegativeCountAsZero pins the exact sentence in Head's doc:
// "A negative n is treated as zero: Head returns the empty string." This is
// the one promise the shipping implementation breaks; Tail keeps the same
// promise (see TestTailKeepsTrailingRunes), so the two functions ought to
// agree, and this test states the agreement the docs require.
//
// The deferred recover is not testing for a panic; the contract forbids one.
// It converts the current panic into a legible failure that names the input,
// instead of letting it crash the whole test binary and hide which other
// contracts still hold.
func TestHeadTreatsNegativeCountAsZero(t *testing.T) {
	const s = "héllo"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Head(%q, -1) panicked (%v); the docs promise the empty string", s, r)
		}
	}()

	if got := truncate.Head(s, -1); got != "" {
		t.Errorf("Head(%q, -1) = %q, want %q", s, got, "")
	}
}
