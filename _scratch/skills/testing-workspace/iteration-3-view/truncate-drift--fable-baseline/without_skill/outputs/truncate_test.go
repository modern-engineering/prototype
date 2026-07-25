package truncate

import (
	"testing"
	"unicode/utf8"
)

// eval calls fn and reports whether it panicked. The documented
// contract never panics, so any recovered value is a test failure.
func eval(fn func() string) (result string, panicked any) {
	defer func() { panicked = recover() }()
	return fn(), nil
}

var headTests = []struct {
	name string
	s    string
	n    int
	want string
}{
	{"empty string", "", 3, ""},
	{"zero n", "hello", 0, ""},
	// Documented: "A negative n is treated as zero: Head returns the
	// empty string." Deliberately kept even though it currently fails:
	// Head has no n <= 0 guard, so r[:n] panics on negative n.
	{"negative n", "hello", -1, ""},
	{"negative n empty string", "", -1, ""},
	{"n less than length", "hello", 3, "hel"},
	{"n equals length", "hello", 5, "hello"},
	{"n exceeds length", "hello", 10, "hello"},
	// The package-doc example: two runes, three bytes.
	{"multi-byte rune kept whole", "héllo", 2, "hé"},
	{"cut after multi-byte rune", "héllo", 3, "hél"},
	{"emoji (4-byte runes)", "a😀b😀", 2, "a😀"},
	{"only multi-byte runes", "日本語", 2, "日本"},
}

func TestHead(t *testing.T) {
	for _, tt := range headTests {
		t.Run(tt.name, func(t *testing.T) {
			got, panicked := eval(func() string { return Head(tt.s, tt.n) })
			if panicked != nil {
				t.Fatalf("Head(%q, %d) panicked: %v; doc promises %q", tt.s, tt.n, panicked, tt.want)
			}
			if got != tt.want {
				t.Errorf("Head(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Head(%q, %d) = %q is not valid UTF-8", tt.s, tt.n, got)
			}
		})
	}
}

var tailTests = []struct {
	name string
	s    string
	n    int
	want string
}{
	{"empty string", "", 3, ""},
	{"zero n", "hello", 0, ""},
	{"negative n", "hello", -1, ""},
	{"negative n empty string", "", -1, ""},
	{"n less than length", "hello", 3, "llo"},
	{"n equals length", "hello", 5, "hello"},
	{"n exceeds length", "hello", 10, "hello"},
	{"multi-byte rune kept whole", "héllo", 4, "éllo"},
	{"emoji (4-byte runes)", "a😀b😀", 2, "b😀"},
	{"only multi-byte runes", "日本語", 2, "本語"},
}

func TestTail(t *testing.T) {
	for _, tt := range tailTests {
		t.Run(tt.name, func(t *testing.T) {
			got, panicked := eval(func() string { return Tail(tt.s, tt.n) })
			if panicked != nil {
				t.Fatalf("Tail(%q, %d) panicked: %v; doc promises %q", tt.s, tt.n, panicked, tt.want)
			}
			if got != tt.want {
				t.Errorf("Tail(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("Tail(%q, %d) = %q is not valid UTF-8", tt.s, tt.n, got)
			}
		})
	}
}
