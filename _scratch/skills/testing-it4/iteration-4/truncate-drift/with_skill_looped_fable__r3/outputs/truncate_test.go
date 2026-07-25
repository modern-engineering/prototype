package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Head shortens text to its first runes. Counting runes rather than
// bytes keeps the cut clean in accented text.
func ExampleHead() {
	fmt.Println(truncate.Head("héllo", 2)) // "hé": two runes, three bytes

	// Input shorter than the limit comes back unchanged, so a
	// generous limit needs no measuring first.
	fmt.Println(truncate.Head("héllo", 80))

	// Output:
	// hé
	// héllo
}

// Tail keeps the last runes, where the distinctive part of a path or
// identifier usually lives.
func ExampleTail() {
	fmt.Println(truncate.Tail("héllo", 2)) // "lo": the last two runes

	// Input shorter than the limit comes back unchanged.
	fmt.Println(truncate.Tail("héllo", 80))

	// Output:
	// lo
	// héllo
}

// Display fields have room for only so many characters, and slicing by
// byte index would mangle accented or CJK text at the cut. Counting
// runes keeps the shortened text valid UTF-8: a preview keeps the head
// of a message, and a path is recognized by its tail.
func Example_display() {
	// "héllo, wörld" is 12 runes but 14 bytes; the é spans bytes 1-2,
	// so a byte-based cut at index 2 would split it. Head counts
	// runes, so every cut lands between characters.
	fmt.Println(truncate.Head("héllo, wörld", 5))

	// The end of a long path identifies the file better than its start.
	fmt.Println(truncate.Tail("/var/log/app/2026-07-23/server.log", 10))

	// A generous limit is fine: input shorter than the limit comes
	// back unchanged, no need to measure it first.
	fmt.Println(truncate.Head("short", 80))

	// Output:
	// héllo
	// server.log
	// short
}

func TestRunesNotBytes(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "héllo", n: 2, head: "hé", tail: "lo"}, // the package doc's own example
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "日本語のテスト", n: 3, head: "日本語", tail: "テスト"},   // three-byte runes
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"}, // exactly n runes
		{s: "hi", n: 10, head: "hi", tail: "hi"},         // generous limit, unchanged
		{s: "", n: 3, head: "", tail: ""},
		{s: "hello", n: 0, head: "", tail: ""},
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

// The docs promise that a negative n is treated as zero, returning the
// empty string. Tail keeps that promise; Head panics instead. This
// test asserts the documented contract, so it fails until Head honors
// it. Owner call: make Head keep the promise, or amend the prose?
func TestNegativeCountMeansEmpty(t *testing.T) {
	if got := truncate.Tail("héllo", -1); got != "" {
		t.Errorf(`Tail("héllo", -1) = %q, want ""`, got)
	}
	if got := truncate.Head("héllo", -1); got != "" {
		t.Errorf(`Head("héllo", -1) = %q, want ""`, got)
	}
}
