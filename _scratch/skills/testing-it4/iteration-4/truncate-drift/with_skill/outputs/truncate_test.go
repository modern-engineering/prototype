package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// Generated identifiers are commonly shortened for display by keeping
// both ends and marking the elision, so neither the human-chosen
// prefix nor the distinguishing suffix is lost. Counting runes keeps
// the accented characters intact at either cut.
func Example_ellipsis() {
	id := "déploiement-frontend-7c9f8d6b54-x2vql"
	fmt.Println(truncate.Head(id, 12) + "…" + truncate.Tail(id, 5))
	// Output:
	// déploiement-…x2vql
}

// Head and Tail mirror one contract from opposite ends, so every case
// pins both sides of the same cut: runes are counted rather than
// bytes, and a limit at or beyond the rune count returns the input
// unchanged.
func TestTruncate(t *testing.T) {
	cases := []struct {
		s          string
		n          int
		head, tail string
	}{
		{s: "hi", n: 5, head: "hi", tail: "hi"},
		{s: "héllo", n: 5, head: "héllo", tail: "héllo"},
		{s: "héllo", n: 2, head: "hé", tail: "lo"},
		{s: "héllo", n: 4, head: "héll", tail: "éllo"},
		{s: "a🙂b", n: 2, head: "a🙂", tail: "🙂b"},
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

// Both doc comments promise that a negative n is treated as zero.
// Tail keeps the promise; Head slices r[:n] without the guard and
// panics. Tail's guard and the matching prose on both functions read
// as a deliberate promise, and Head's panic as an oversight, so this
// test sides with the prose: it fails until Head gains the guard or
// the owner amends the docs. The recover turns the panic into an
// ordinary failure so the rest of the suite still reports.
func TestNegativeCountKeepsNothing(t *testing.T) {
	if got := truncate.Tail("héllo", -1); got != "" {
		t.Errorf(`Tail("héllo", -1) = %q, want ""`, got)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf(`Head("héllo", -1) panicked (%v), want ""`, r)
		}
	}()
	if got := truncate.Head("héllo", -1); got != "" {
		t.Errorf(`Head("héllo", -1) = %q, want ""`, got)
	}
}
