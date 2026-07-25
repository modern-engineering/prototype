package units_test

import (
	"fmt"
	"testing"
	"time"

	"example.invalid/units/internal/units"
)

// A size written with a binary suffix parses to its exact byte count,
// and a byte count that is an even multiple of a suffix formats back to
// the canonical string, so sizes survive a round trip through
// configuration files.
func TestSizeRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		bytes     int64
		formatted string
	}{
		{"bare bytes", "512", 512, "512"},
		{"kilobytes", "10K", 10 << 10, "10K"},
		{"megabytes", "3M", 3 << 20, "3M"},
		{"gigabytes", "2G", 2 << 30, "2G"},
		{"lowercase suffix normalizes", "4k", 4 << 10, "4K"},
		{"uneven count stays in bytes", "1500", 1500, "1500"},
		{"surrounding space is trimmed", " 8M ", 8 << 20, "8M"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := units.ParseSize(tt.in)
			if err != nil {
				t.Fatalf("ParseSize(%q): %v", tt.in, err)
			}
			if got != tt.bytes {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.in, got, tt.bytes)
			}
			if s := units.FormatSize(got); s != tt.formatted {
				t.Errorf("FormatSize(%d) = %q, want %q", got, s, tt.formatted)
			}
		})
	}
}

// A parseable duration string wins over the default, while an empty or
// malformed string falls back to it. The fallback on garbage is the
// documented leniency of the helper, not an error path.
func TestParseDurationOrDefault(t *testing.T) {
	def := 5 * time.Second
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"valid duration parses", "1500ms", 1500 * time.Millisecond},
		{"compound duration parses", "2m30s", 2*time.Minute + 30*time.Second},
		{"zero duration overrides default", "0s", 0},
		{"empty string falls back", "", def},
		{"garbage falls back", "soon", def},
		{"bare number falls back", "10", def},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := units.ParseDurationOrDefault(tt.in, def); got != tt.want {
				t.Errorf("ParseDurationOrDefault(%q, %v) = %v, want %v", tt.in, def, got, tt.want)
			}
		})
	}
}

func ExampleParseSize() {
	n, _ := units.ParseSize("3M")
	fmt.Println(n)
	// Output: 3145728
}
