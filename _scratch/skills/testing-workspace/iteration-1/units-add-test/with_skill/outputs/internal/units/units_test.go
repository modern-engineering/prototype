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

func ExampleParseSize() {
	n, _ := units.ParseSize("3M")
	fmt.Println(n)
	// Output: 3145728
}

// A duration string that time.ParseDuration accepts wins over the
// default; an empty string or one that does not parse falls back to the
// default, because the documented leniency keeps a stray configuration
// value from becoming an error at this layer.
func TestParseDurationOrDefault(t *testing.T) {
	const def = 30 * time.Second
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"valid duration wins over default", "1m30s", 90 * time.Second},
		{"empty string falls back", "", def},
		{"garbage falls back", "not-a-duration", def},
		{"bare number without unit falls back", "15", def},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := units.ParseDurationOrDefault(tt.in, def); got != tt.want {
				t.Errorf("ParseDurationOrDefault(%q, %v) = %v, want %v", tt.in, def, got, tt.want)
			}
		})
	}
}
