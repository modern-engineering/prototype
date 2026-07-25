// Black-box tests for package truncate. They exercise only the
// exported API and assert the behavior promised by the doc comments,
// which are the contract callers will rely on once this ships.
package truncate_test

import (
	"testing"
	"unicode/utf8"

	"example.invalid/truncate"
)

func TestHead(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{name: "ascii shorter than limit", s: "hi", n: 5, want: "hi"},
		{name: "ascii exactly at limit", s: "hello", n: 5, want: "hello"},
		{name: "ascii cut", s: "hello", n: 3, want: "hel"},
		{name: "zero limit", s: "hello", n: 0, want: ""},
		{name: "empty input", s: "", n: 3, want: ""},
		{name: "empty input zero limit", s: "", n: 0, want: ""},
		{name: "doc example multibyte", s: "héllo", n: 2, want: "hé"},
		{name: "cut lands after multibyte rune", s: "héllo", n: 3, want: "hél"},
		{name: "emoji counted as single runes", s: "a👍b👍", n: 2, want: "a👍"},
		{name: "all multibyte exact", s: "日本語", n: 3, want: "日本語"},
		{name: "all multibyte cut", s: "日本語", n: 2, want: "日本"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate.Head(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("Head(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Head(%q, %d) = %q is not valid UTF-8", tt.s, tt.n, got)
			}
		})
	}
}

// TestHeadNegative pins the documented contract: "A negative n is
// treated as zero: Head returns the empty string."
//
// KNOWN FAILURE, left in deliberately. The current implementation
// panics instead: Head's only guard is n > len(r), which does not
// catch a negative n, so r[:n] slices out of range. Tail has the
// n <= 0 guard; Head is missing the same two lines. Since the package
// is about to ship, the test asserts the doc's promise rather than
// the buggy behavior, and should pass once Head is fixed (or the doc
// is amended, in which case update this test alongside it). The
// recover keeps the panic from aborting the rest of the suite.
func TestHeadNegative(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Head(\"hello\", -1) panicked: %v; doc promises it returns \"\"", r)
		}
	}()
	if got := truncate.Head("hello", -1); got != "" {
		t.Errorf(`Head("hello", -1) = %q, want ""`, got)
	}
}

func TestTail(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{name: "ascii shorter than limit", s: "hi", n: 5, want: "hi"},
		{name: "ascii exactly at limit", s: "hello", n: 5, want: "hello"},
		{name: "ascii cut", s: "hello", n: 3, want: "llo"},
		{name: "zero limit", s: "hello", n: 0, want: ""},
		{name: "negative limit", s: "hello", n: -1, want: ""},
		{name: "empty input", s: "", n: 3, want: ""},
		{name: "empty input negative limit", s: "", n: -2, want: ""},
		{name: "cut excludes multibyte rune cleanly", s: "héllo", n: 3, want: "llo"},
		{name: "cut lands before multibyte rune", s: "héllo", n: 4, want: "éllo"},
		{name: "emoji counted as single runes", s: "a👍b👍", n: 2, want: "b👍"},
		{name: "all multibyte exact", s: "日本語", n: 3, want: "日本語"},
		{name: "all multibyte cut", s: "日本語", n: 2, want: "本語"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate.Tail(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("Tail(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Tail(%q, %d) = %q is not valid UTF-8", tt.s, tt.n, got)
			}
		})
	}
}
