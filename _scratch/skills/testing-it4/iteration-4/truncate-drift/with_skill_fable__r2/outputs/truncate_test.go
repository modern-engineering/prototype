package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A column too narrow for a full identifier keeps its head and tail
// around an ellipsis. Both cut points land on character boundaries
// because the functions count runes, so accented text never splits
// mid-character.
func Example() {
	title := "Ünïcödé-heavy dashboard title that overflows its column"
	fmt.Println(truncate.Head(title, 12) + "…" + truncate.Tail(title, 6))
	// Output:
	// Ünïcödé-heav…column
}

// Head and Tail are mirror images, so one table carries both
// expectations per input. Cut points deliberately land next to
// multi-byte runes, where byte-indexed slicing would mangle a
// character.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hello", n: 3, head: "hel", tail: "llo"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "héllo", n: 4, head: "héll", tail: "éllo"},
		{s: "日本語テキスト", n: 3, head: "日本語", tail: "キスト"},
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 10, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 0, head: "", tail: ""},
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

// Both doc comments promise that a negative n is treated as zero. Tail
// honors it; Head currently panics on the r[:n] slice because it lacks
// Tail's n <= 0 guard, which reads as an oversight given the two are
// otherwise mirror images. This test states the documented contract
// and fails until the owner adds the guard or amends the prose; the
// recover turns the panic into a plain failure so the rest of the run
// survives.
func TestNegativeCountMeansEmpty(t *testing.T) {
	check := func(name string, fn func(string, int) string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s(%q, -1) panicked: %v, want \"\" per the doc comment", name, "héllo", r)
			}
		}()
		if got := fn("héllo", -1); got != "" {
			t.Errorf("%s(%q, -1) = %q, want \"\"", name, "héllo", got)
		}
	}
	check("Head", truncate.Head)
	check("Tail", truncate.Tail)
}
