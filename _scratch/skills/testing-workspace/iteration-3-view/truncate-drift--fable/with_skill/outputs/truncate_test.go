package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// call invokes fn(s, n) and turns a panic into an ordinary test
// failure naming the call. The package documents both functions as
// safe for any n, including negative ones, so no input may panic; a
// bare panic would abort the whole test binary and hide the results
// of the remaining tests.
func call(t *testing.T, name string, fn func(string, int) string, s string, n int) string {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("%s(%q, %d) panicked: %v; the doc promises the empty string for a negative n", name, s, n, p)
		}
	}()
	return fn(s, n)
}

// A negative n is documented to be treated as zero, yielding the
// empty string. Callers are invited to pass limits without measuring
// or validating them first, so a negative value from arithmetic on
// user input must not crash the program. This special case comes
// first: the rune-counting tests below assume both functions are
// total.
func TestNegativeCountIsTreatedAsZero(t *testing.T) {
	if got := call(t, "Head", truncate.Head, "héllo", -1); got != "" {
		t.Errorf(`Head("héllo", -1) = %q, want ""`, got)
	}
	if got := call(t, "Tail", truncate.Tail, "héllo", -1); got != "" {
		t.Errorf(`Tail("héllo", -1) = %q, want ""`, got)
	}
}

// The package's central promise for Head is that the cut counts
// runes, so multi-byte characters are kept whole and the result stays
// valid UTF-8. This also pins the generous-limit clause, that an n at
// or above the rune count returns s unchanged.
func TestHeadCountsRunesNotBytes(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"héllo", 2, "hé"},     // the documented example: two runes, three bytes
		{"hello", 3, "hel"},    // plain ASCII, where runes and bytes coincide
		{"日本語abc", 2, "日本"},    // three-byte runes; a byte cut at 2 would mangle 日
		{"héllo", 5, "héllo"},  // n equals the rune count: unchanged
		{"héllo", 10, "héllo"}, // generous limit: unchanged
		{"héllo", 0, ""},
		{"", 4, ""},
	}
	for _, tt := range tests {
		if got := truncate.Head(tt.s, tt.n); got != tt.want {
			t.Errorf("Head(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

// Tail makes the same rune-counting promise as Head, measured from
// the end of the string instead: a cut landing just before a
// multi-byte character keeps it whole, and an n at or above the rune
// count returns s unchanged.
func TestTailCountsRunesNotBytes(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"héllo", 2, "lo"},
		{"héllo", 4, "éllo"},   // the cut lands on the two-byte é; a byte cut would split it
		{"日本語abc", 4, "語abc"},  // three-byte runes counted from the end
		{"hello", 3, "llo"},    // plain ASCII, where runes and bytes coincide
		{"héllo", 5, "héllo"},  // n equals the rune count: unchanged
		{"héllo", 10, "héllo"}, // generous limit: unchanged
		{"héllo", 0, ""},
		{"", 4, ""},
	}
	for _, tt := range tests {
		if got := truncate.Tail(tt.s, tt.n); got != tt.want {
			t.Errorf("Tail(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

// The package doc's own string shows the promised rune counting:
// Head keeps the two-byte é whole, and Tail counts the same way from
// the other end.
func Example() {
	fmt.Println(truncate.Head("héllo", 2))
	fmt.Println(truncate.Tail("héllo", 4))
	// Output:
	// hé
	// éllo
}
