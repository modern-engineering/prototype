// Package truncate provides rune-safe truncation helpers for display
// purposes.
//
// Slicing a Go string by byte index can split a multi-byte UTF-8
// sequence, leaving a mangled character at the cut point. The
// functions in this package count runes instead of bytes, so the
// shortened strings remain valid UTF-8 and render cleanly in user
// interfaces, log lines, and terminal output.
//
// For example, Head("héllo", 2) returns "hé": two runes, three bytes.
package truncate

// Head returns the first n runes of s.
//
// If s has fewer than n runes, Head returns s unchanged, so callers
// may pass a generous limit without measuring the input first. A
// negative n is treated as zero: Head returns the empty string.
func Head(s string, n int) string {
	r := []rune(s)
	if n > len(r) {
		return s
	}
	return string(r[:n])
}

// Tail returns the last n runes of s.
//
// If s has fewer than n runes, Tail returns s unchanged, so callers
// may pass a generous limit without measuring the input first. A
// negative n is treated as zero: Tail returns the empty string.
func Tail(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if n > len(r) {
		return s
	}
	return string(r[len(r)-n:])
}
