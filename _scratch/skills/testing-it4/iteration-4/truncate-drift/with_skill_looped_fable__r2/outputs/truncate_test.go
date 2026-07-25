package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Display surfaces trim text to fit: a preview column keeps the start
// of a message, a breadcrumb keeps the end of a path. Head and Tail
// count runes, so the cut always lands on a character boundary and the
// shortened text stays valid UTF-8.
func Example() {
	// A preview column keeps the first few characters of a message.
	fmt.Println(truncate.Head("héllo, wörld", 6))

	// The interesting end of a long path is its file name.
	fmt.Println(truncate.Tail("/var/log/app/tödäy.log", 9))

	// Short values pass through unchanged, so a generous limit is safe
	// without measuring the input first.
	fmt.Println(truncate.Head("ok", 40))

	// Output:
	// héllo,
	// tödäy.log
	// ok
}

// Both doc comments promise that a negative n is treated as zero.
// Tail delivers on that promise; Head panics today. The negative-n row
// fails until the owner adds Head's missing guard or amends the prose.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		// Cuts count runes, never bytes, on multi-byte text.
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "héllo", n: 4, head: "héll", tail: "éllo"},
		{s: "日本語のログ", n: 3, head: "日本語", tail: "のログ"},
		{s: "a🙂b", n: 2, head: "a🙂", tail: "🙂b"},

		// Generous limits pass the string through unchanged.
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 40, head: "héllo", tail: "héllo"},
		{s: "", n: 3, head: "", tail: ""},

		// Non-positive limits mean the empty string.
		{s: "héllo", n: 0, head: "", tail: ""},
		{s: "héllo", n: -1, head: "", tail: ""},
	}
	for _, c := range cases {
		if got := headRecovered(t, c.s, c.n, c.head); got != c.head {
			t.Errorf("Head(%q, %d) = %q, want %q", c.s, c.n, got, c.head)
		}
		if got := truncate.Tail(c.s, c.n); got != c.tail {
			t.Errorf("Tail(%q, %d) = %q, want %q", c.s, c.n, got, c.tail)
		}
	}
}

// headRecovered reports a panic from Head as an ordinary row failure
// instead of letting it kill the run.
func headRecovered(t *testing.T, s string, n int, want string) (got string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(%q, %d) = panic(%v), want %q", s, n, r, want)
		}
	}()
	return truncate.Head(s, n)
}
