package application

import (
	"flag"
	"go/token"
	"testing"
)

// CheckDescriptor verifies that a descriptor honours the catalogue
// citizenship contract tooling relies on (help rendering, schema
// extraction, solution compilation): a well-formed name and a Make factory
// that upholds the dry-instantiation invariant — callable repeatedly, a
// fresh service per call, an identical flag schema every time. Catalogue
// packages call it from their own tests, one call per exported descriptor.
func CheckDescriptor(tb testing.TB, d *Descriptor) {
	tb.Helper()
	if d == nil {
		tb.Fatal("CheckDescriptor: descriptor is nil")
	}
	if !token.IsIdentifier(d.Name) {
		tb.Fatalf("descriptor name %q is not a valid Go identifier", d.Name)
	}
	if d.Make == nil {
		tb.Fatalf("descriptor %q has no Make factory; catalogue citizens must be constructible dry", d.Name)
	}

	first := makeDry(tb, d, "first")
	second := makeDry(tb, d, "second")

	f1, f2 := first.Flags(), second.Flags()
	if (f1 == nil) != (f2 == nil) {
		tb.Fatalf("descriptor %q: Flags() nil-ness differs across Make calls", d.Name)
	}
	if f1 == nil {
		return // a flagless application; nothing further to compare
	}
	if f1 == f2 {
		tb.Fatalf("descriptor %q: two Make calls share one FlagSet; each call must declare a fresh surface", d.Name)
	}

	schema := func(fs *flag.FlagSet) map[string][2]string {
		m := make(map[string][2]string)
		fs.VisitAll(func(f *flag.Flag) { m[f.Name] = [2]string{f.Usage, f.DefValue} })
		return m
	}
	s1, s2 := schema(f1), schema(f2)
	if len(s1) != len(s2) {
		tb.Fatalf("descriptor %q: flag count differs across Make calls (%d vs %d)", d.Name, len(s1), len(s2))
	}
	for name, meta := range s1 {
		if other, ok := s2[name]; !ok {
			tb.Fatalf("descriptor %q: flag %q missing on the second Make call", d.Name, name)
		} else if other != meta {
			tb.Fatalf("descriptor %q: flag %q schema differs across Make calls (%q vs %q)", d.Name, name, meta, other)
		}
	}
}

// makeDry calls d.Make once, converting a panic or a nil service into a
// test failure attributed to the descriptor. The recover guard wraps only
// the user's factory call, so the harness's own failure signals pass
// through untouched.
func makeDry(tb testing.TB, d *Descriptor, ordinal string) Service {
	tb.Helper()
	svc, panicked := func() (s Service, p any) {
		defer func() { p = recover() }()
		return d.Make(), nil
	}()
	if panicked != nil {
		tb.Fatalf("descriptor %q: Make panicked on the %s call: %v", d.Name, ordinal, panicked)
	}
	if svc == nil {
		tb.Fatalf("descriptor %q: Make returned nil on the %s call", d.Name, ordinal)
	}
	return svc
}
