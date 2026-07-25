package truncate_test

import (
	"fmt"
	"testing"

	"example.invalid/truncate"
)

// A display layer trims a label to fit a narrow column. Counting runes
// rather than bytes keeps the trimmed label valid UTF-8, so a multi-byte
// character is never cut in half, and a label already within budget comes
// back untouched.
func Example_fitColumn() {
	const label = "héllo" // five runes, six bytes: é takes two bytes

	// Keep the first two runes. A byte slice label[:2] would split é and
	// leave a mangled character; Head counts runes and stops cleanly.
	head := truncate.Head(label, 2)
	fmt.Printf("%s is %d bytes\n", head, len(head))

	// Keep the last two runes.
	fmt.Println(truncate.Tail(label, 2))

	// A label already within budget is returned unchanged, so callers can
	// pass a generous limit without measuring the input first.
	fmt.Println(truncate.Head(label, 40))

	// Output:
	// hé is 3 bytes
	// lo
	// héllo
}

// Truncating from either end obeys one contract: take n runes off the
// chosen side, hand the input back untouched once n reaches its length,
// and keep every multi-byte character whole. One table drives both
// helpers so the front and the back cannot drift apart.
func TestHeadAndTail(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string, int) string
		s    string
		n    int
		want string
	}{
		{"Head keeps a whole multi-byte rune", truncate.Head, "héllo", 2, "hé"},
		{"Head at exact length returns the input", truncate.Head, "héllo", 5, "héllo"},
		{"Head over the length returns the input", truncate.Head, "héllo", 40, "héllo"},
		{"Head with zero count returns empty", truncate.Head, "héllo", 0, ""},
		{"Tail keeps a whole multi-byte rune", truncate.Tail, "héllo", 4, "éllo"},
		{"Tail at exact length returns the input", truncate.Tail, "héllo", 5, "héllo"},
		{"Tail over the length returns the input", truncate.Tail, "héllo", 40, "héllo"},
		{"Tail with zero count returns empty", truncate.Tail, "héllo", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.fn(c.s, c.n); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// Both docs say a negative count is treated as zero and yields the empty
// string, which lets a caller forward an untrusted count without a sign
// check. Tail guards n <= 0 up front; Head has no such guard and slices
// r[:n] with a negative bound, so it panics instead. This pins the
// documented promise: a failure here flags that Head's code and its doc
// disagree, to be resolved by adding Head's guard or dropping the promise.
func TestNegativeCountYieldsEmptyString(t *testing.T) {
	emptyOnNegative(t, "Head", truncate.Head)
	emptyOnNegative(t, "Tail", truncate.Tail)
}

// emptyOnNegative asserts the documented promise that a negative count
// yields the empty string. It recovers so a contract-breaking panic is
// reported as a plain failure instead of crashing the test binary.
func emptyOnNegative(t *testing.T, name string, fn func(string, int) string) {
	t.Helper()
	const s = "héllo"
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s(%q, -1) panicked (%v), want %q", name, s, r, "")
		}
	}()
	if got := fn(s, -1); got != "" {
		t.Errorf("%s(%q, -1) = %q, want %q", name, s, got, "")
	}
}
