// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"flag"
	"testing"
)

// CheckSchemeType verifies that a stanza scheme type honours the
// catalogue citizenship contract tooling relies on (registration,
// stanza checking, schema pinning), the scheme sibling of
// [CheckProvisionType]: a qualifier stanzas can spell — identifier
// segments joined by dots, the same decider registration validation
// runs — and a Params function that declares the identical key
// schema on every fresh dry surface it is handed. The linker makes a
// fresh throwaway surface per checked stanza, so Params must be
// repeatable and must keep its declarations on the surface it is
// given, panic-free. Catalogue packages call it from their own
// tests, one call per exported scheme type.
//
// A nil Params passes: it declares a keyless scheme, which claims
// its qualifier and rejects every stanza key as unknown.
func CheckSchemeType(tb testing.TB, st *SchemeType) {
	tb.Helper()
	if st == nil {
		tb.Fatal("CheckSchemeType: scheme type is nil")
		return // unreachable: Fatal never returns, but tb is an interface and static analysis cannot see that
	}
	if !validQualifier(st.Qualifier) {
		tb.Fatalf("scheme qualifier %q is not identifier segments joined by dots (e.g. k8s.pod)", st.Qualifier)
	}
	if st.Params == nil {
		return // a keyless scheme; nothing to instantiate
	}

	first := schemeSurface(tb, st, "first")
	second := schemeSurface(tb, st, "second")

	schema := func(fs *flag.FlagSet) map[string][2]string {
		m := make(map[string][2]string)
		fs.VisitAll(func(f *flag.Flag) { m[f.Name] = [2]string{f.Usage, f.DefValue} })
		return m
	}
	s1, s2 := schema(first), schema(second)
	if len(s1) != len(s2) {
		tb.Fatalf("key count differs across Params calls (%d vs %d)", len(s1), len(s2))
	}
	for name, meta := range s1 {
		if other, ok := s2[name]; !ok {
			tb.Fatalf("key %q missing on the second Params call", name)
		} else if other != meta {
			tb.Fatalf("key %q schema differs across Params calls (%q vs %q)", name, meta, other)
		}
	}
}

// schemeSurface hands Params one fresh dry surface, converting a
// panic into a test failure. The recover guard wraps only the user's
// declaration call, so the harness's own failure signals pass through
// untouched.
func schemeSurface(tb testing.TB, st *SchemeType, ordinal string) *flag.FlagSet {
	tb.Helper()
	fs := flag.NewFlagSet("scheme", flag.ContinueOnError)
	panicked := func() (r any) {
		defer func() { r = recover() }()
		st.Params(fs)
		return nil
	}()
	if panicked != nil {
		tb.Fatalf("Params panicked on the %s surface: %v", ordinal, panicked)
	}
	return fs
}
