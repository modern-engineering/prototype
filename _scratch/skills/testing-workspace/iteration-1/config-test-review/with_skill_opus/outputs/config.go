// Package config loads simple key/value configuration files.
//
// A configuration file holds one "key = value" pair per line. Blank
// lines and lines beginning with '#' are ignored. Whitespace around
// keys and values is trimmed.
package config

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Config holds the key/value pairs read from a configuration file.
// Lookups for unknown keys fall back to caller-supplied defaults.
type Config struct {
	values map[string]string
}

// Load reads key/value pairs from r. A line that is neither blank, a
// comment, nor a "key = value" pair produces a positioned error of the
// form "line N: ...".
func Load(r io.Reader) (*Config, error) {
	c := &Config{values: make(map[string]string)}
	scanner := bufio.NewScanner(r)
	n := 0
	for scanner.Scan() {
		n++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: missing '=' in %q", n, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", n)
		}
		c.values[key] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return c, nil
}

// String returns the value for key. An unknown key falls back to def.
func (c *Config) String(key, def string) string {
	if v, ok := c.values[key]; ok {
		return v
	}
	return def
}

// Int returns the value for key parsed as a decimal integer. An
// unknown key, or a value that does not parse, falls back to def.
func (c *Config) Int(key string, def int) int {
	v, ok := c.values[key]
	if !ok {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// Duration returns the value for key parsed with time.ParseDuration.
// An unknown key, or a value that does not parse, falls back to def.
func (c *Config) Duration(key string, def time.Duration) time.Duration {
	v, ok := c.values[key]
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// Keys returns every configured key in sorted order.
func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
