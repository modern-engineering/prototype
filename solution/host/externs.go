// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ParseExterns reads extern bindings from r: one name=value per line,
// the exact shape the repeatable -extern argument takes, in file form
// — deliberately not a site format. Blank lines and full-line #
// comments are skipped; a # after the = belongs to the value, since
// endpoints and secrets may contain it; values run to end of line,
// trimmed. A name bound twice is refused: callers hand a host
// disjoint fragments (a plain file and a secret-mounted one, say), so
// a duplicate is always a mistake, never an override.
//
// The seam this fills is delivery, not semantics: a Kubernetes pod
// mounts its extern values as files where a shell passes arguments,
// and either way the bindings land in [Config].Externs and meet the
// same gate. A recorded site-binding document with classes beyond
// externs stays the open door it is (D-13).
func ParseExterns(r io.Reader) (map[string]string, error) {
	externs := make(map[string]string)
	sc := bufio.NewScanner(r)
	line := 0
	for sc.Scan() {
		line++
		text := strings.TrimSpace(sc.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		name, value, ok := strings.Cut(text, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !ok || name == "" {
			return nil, fmt.Errorf("externs: line %d: %q is not name=value", line, text)
		}
		if _, dup := externs[name]; dup {
			return nil, fmt.Errorf("externs: line %d: %s bound twice", line, name)
		}
		externs[name] = value
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("externs: %v", err)
	}
	return externs, nil
}
