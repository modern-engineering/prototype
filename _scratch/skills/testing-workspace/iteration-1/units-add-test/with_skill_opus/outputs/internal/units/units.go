// Package units converts between human-readable size strings and byte
// counts, and offers lenient duration parsing for configuration values.
package units

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	kilobyte int64 = 1024
	megabyte       = kilobyte * 1024
	gigabyte       = megabyte * 1024
)

// ParseSize converts a size string such as "10K" or "3M" into a byte
// count. A bare number is a count of bytes; the suffixes K, M, and G
// (case-insensitive) denote binary multiples, so 1K is 1024 bytes.
func ParseSize(s string) (int64, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, fmt.Errorf("units: empty size")
	}
	mult := int64(1)
	switch trimmed[len(trimmed)-1] {
	case 'K', 'k':
		mult = kilobyte
		trimmed = trimmed[:len(trimmed)-1]
	case 'M', 'm':
		mult = megabyte
		trimmed = trimmed[:len(trimmed)-1]
	case 'G', 'g':
		mult = gigabyte
		trimmed = trimmed[:len(trimmed)-1]
	}
	n, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("units: invalid size %q", s)
	}
	if n < 0 {
		return 0, fmt.Errorf("units: negative size %q", s)
	}
	return n * mult, nil
}

// FormatSize renders a byte count using the largest binary suffix that
// divides it evenly, so FormatSize(3145728) returns "3M".
func FormatSize(n int64) string {
	switch {
	case n >= gigabyte && n%gigabyte == 0:
		return strconv.FormatInt(n/gigabyte, 10) + "G"
	case n >= megabyte && n%megabyte == 0:
		return strconv.FormatInt(n/megabyte, 10) + "M"
	case n >= kilobyte && n%kilobyte == 0:
		return strconv.FormatInt(n/kilobyte, 10) + "K"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// ParseDurationOrDefault parses s with time.ParseDuration. An empty
// string returns def. A string that does not parse also returns def:
// the leniency is deliberate, so a stray configuration value never
// turns into an error at this layer.
func ParseDurationOrDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
