// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host_test

import (
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution/host"
)

// TestParseExterns pins the extern-file shape: name=value lines with
// blank lines and full-line comments skipped, names and values
// trimmed, a # after the = kept as value bytes, and the value running
// through any further = signs — connection strings carry both.
func TestParseExterns(t *testing.T) {
	got, err := host.ParseExterns(strings.NewReader(`
# the site's substrate handles
natsEndpoint = nats://user:p#ss@host:4222
dsn = postgres://h/db?sslmode=disable&opt=1

token=abc=def
`))
	if err != nil {
		t.Fatalf("ParseExterns: %v", err)
	}
	want := map[string]string{
		"natsEndpoint": "nats://user:p#ss@host:4222",
		"dsn":          "postgres://h/db?sslmode=disable&opt=1",
		"token":        "abc=def",
	}
	if len(got) != len(want) {
		t.Fatalf("ParseExterns = %v, want %v", got, want)
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("extern %s = %q, want %q", name, got[name], value)
		}
	}
}

// TestParseExternsRefusals pins the two refusals, each positioned by
// line: a line without = (or with an empty name) is not a binding,
// and a name bound twice is a mistake — fragments are disjoint by
// construction, so there is no override semantics to fall back on.
func TestParseExternsRefusals(t *testing.T) {
	if _, err := host.ParseExterns(strings.NewReader("natsEndpoint\n")); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("bare word: err = %v, want a line-positioned refusal", err)
	}
	if _, err := host.ParseExterns(strings.NewReader("=value\n")); err == nil {
		t.Fatalf("empty name: err = %v, want a refusal", err)
	}
	if _, err := host.ParseExterns(strings.NewReader("a=1\na=2\n")); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("duplicate: err = %v, want a line-positioned refusal naming the repeat", err)
	}
}
