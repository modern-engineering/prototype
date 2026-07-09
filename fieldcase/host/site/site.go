// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package site reads the flat site-configuration format the anomaly
// deployment artifacts consume: the stage-(c) input that binds a
// compiled image's extern symbols, optionally rebinds its vars, and
// feeds the provision drivers. Both the generator (fieldcase-gen) and
// the generated host read the same format, so a site file validated at
// generation time is byte-for-byte the file the host loads at startup.
//
// The format is deliberately primitive — one "key = value" pair per
// line — so that no configuration-language dependency enters the
// deployment path (the public contract has no site format yet; this is
// the field-case-local minimal shape). Keys are namespaced by their
// first dotted segment:
//
//	# comment lines and blank lines are skipped
//	extern.kafkaCluster = localhost:9092   # binds an extern (MUST bind every reachable one)
//	var.networkController = http://127.0.0.1:8081  # rebinds a var (MAY)
//	driver.kafka.brokers = localhost:9092  # feeds a provision driver's outputs
//
// Values run from the first '=' to the end of the line, trimmed;
// inline comments are NOT supported (a '#' after the '=' belongs to
// the value, since endpoints and passwords may contain it). A key may
// be bound at most once across all files of one load: the generator
// splits sites into disjoint fragments (ConfigMap and Secret halves),
// so a duplicate is always a mistake, never an override.
package site

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// The key namespaces, the first dotted segment of every site key.
const (
	prefixExtern = "extern"
	prefixVar    = "var"
	prefixDriver = "driver"
)

// A Site is one loaded site configuration: extern bindings, var
// overrides, and per-provision driver sections.
type Site struct {
	externs map[string]string            // symbol name -> value
	vars    map[string]string            // symbol name -> override
	drivers map[string]map[string]string // provision instance -> output -> value

	origin map[string]string // fully qualified key -> "file:line", for duplicate reports
}

// Load reads and merges the named site files in order. Every key must
// be bound exactly once across all files; a duplicate — within one
// file or across files — is an error naming both origins.
func Load(paths ...string) (*Site, error) {
	s := &Site{
		externs: make(map[string]string),
		vars:    make(map[string]string),
		drivers: make(map[string]map[string]string),
		origin:  make(map[string]string),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("site: %v", err)
		}
		if err := s.parse(path, string(data)); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// parse folds one file's lines into the site.
func (s *Site) parse(path, data string) error {
	for i, line := range strings.Split(data, "\n") {
		lineno := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			return fmt.Errorf("site: %s:%d: not a key = value line: %q", path, lineno, trimmed)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("site: %s:%d: empty key", path, lineno)
		}
		origin := fmt.Sprintf("%s:%d", path, lineno)
		if first, dup := s.origin[key]; dup {
			return fmt.Errorf("site: %s: duplicate key %s (first bound at %s)", origin, key, first)
		}
		s.origin[key] = origin

		prefix, rest, _ := strings.Cut(key, ".")
		switch prefix {
		case prefixExtern:
			if rest == "" {
				return fmt.Errorf("site: %s: extern key names no symbol", origin)
			}
			s.externs[rest] = value
		case prefixVar:
			if rest == "" {
				return fmt.Errorf("site: %s: var key names no symbol", origin)
			}
			s.vars[rest] = value
		case prefixDriver:
			instance, output, ok := strings.Cut(rest, ".")
			if !ok || instance == "" || output == "" {
				return fmt.Errorf("site: %s: driver key must be driver.<instance>.<output>: %q", origin, key)
			}
			sec := s.drivers[instance]
			if sec == nil {
				sec = make(map[string]string)
				s.drivers[instance] = sec
			}
			sec[output] = value
		default:
			return fmt.Errorf("site: %s: unknown key namespace %q (want extern, var, or driver): %q", origin, prefix, key)
		}
	}
	return nil
}

// Extern returns the site's binding for the named extern symbol.
func (s *Site) Extern(name string) (string, bool) {
	v, ok := s.externs[name]
	return v, ok
}

// Var returns the site's override for the named var symbol.
func (s *Site) Var(name string) (string, bool) {
	v, ok := s.vars[name]
	return v, ok
}

// Driver returns the site's driver section for the named provision
// instance; nil when the site carries none. The returned map is the
// site's own — callers must not mutate it.
func (s *Site) Driver(instance string) map[string]string {
	return s.drivers[instance]
}

// Externs returns the bound extern names, sorted.
func (s *Site) Externs() []string { return sortedKeys(s.externs) }

// Vars returns the overridden var names, sorted.
func (s *Site) Vars() []string { return sortedKeys(s.vars) }

// Drivers returns the provision instances with driver sections, sorted.
func (s *Site) Drivers() []string { return sortedKeys(s.drivers) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
