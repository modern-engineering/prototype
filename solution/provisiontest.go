// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"flag"
	"go/token"
	"reflect"
	"testing"
)

// CheckProvisionType verifies that a provision type honours the
// catalogue citizenship contract tooling relies on (schema extraction,
// solution compilation, enactment), mirroring what
// [application.CheckDescriptor] is to components: explicit kinds, an
// output scheme of unique identifiers inside the type vocabulary, and
// a Make factory upholding the dry-instantiation invariant — callable
// repeatedly, a fresh provisioner with a fresh flag surface per call,
// an identical flag schema every time. Catalogue packages call it from
// their own tests, one call per exported provision type.
//
// A nil Make passes: it declares a parameterless, driverless type,
// which compiles fine and is refused only at enactment.
func CheckProvisionType(tb testing.TB, pt *ProvisionType) {
	tb.Helper()
	if pt == nil {
		tb.Fatal("CheckProvisionType: provision type is nil")
		return // unreachable: Fatal never returns, but tb is an interface and static analysis cannot see that
	}
	if pt.Kinds == 0 {
		tb.Fatal("provision type registers no kinds (declare Slice, Attach, or both)")
	}
	if pt.Kinds&^(Slice|Attach) != 0 {
		tb.Fatalf("provision type registers unknown kinds %#b", uint8(pt.Kinds))
	}
	seen := make(map[string]bool, len(pt.Outputs))
	for _, out := range pt.Outputs {
		if !token.IsIdentifier(out.Name) {
			tb.Fatalf("output name %q is not a valid Go identifier", out.Name)
		}
		if seen[out.Name] {
			tb.Fatalf("provision type declares output %s twice", out.Name)
		}
		seen[out.Name] = true
		if !out.Type.valid() {
			tb.Fatalf("output %s declares unknown type %q (string, int, bool, or duration; empty means string)", out.Name, out.Type)
		}
	}
	if pt.Make == nil {
		return // parameterless and driverless; nothing to instantiate
	}

	first := makeProvisioner(tb, pt, "first")
	second := makeProvisioner(tb, pt, "second")

	// Identity is checked only where it means something: two nonzero
	// pointer provisioners must differ, while value provisioners —
	// zero-size ones especially — are legitimately indistinguishable,
	// so their freshness is observable only through the flag surface.
	if t := reflect.TypeOf(first); t == reflect.TypeOf(second) &&
		t.Kind() == reflect.Pointer && t.Elem().Size() > 0 && first == second {
		tb.Fatal("two Make calls returned the same provisioner; each call must make a fresh instance")
	}

	f1, f2 := first.Flags(), second.Flags()
	if (f1 == nil) != (f2 == nil) {
		tb.Fatal("Flags() nil-ness differs across Make calls")
	}
	if f1 == nil {
		return // a flagless provision type; nothing further to compare
	}
	if f1 == f2 {
		tb.Fatal("two Make calls share one FlagSet; each call must declare a fresh surface")
	}

	schema := func(fs *flag.FlagSet) map[string][2]string {
		m := make(map[string][2]string)
		fs.VisitAll(func(f *flag.Flag) { m[f.Name] = [2]string{f.Usage, f.DefValue} })
		return m
	}
	s1, s2 := schema(f1), schema(f2)
	if len(s1) != len(s2) {
		tb.Fatalf("flag count differs across Make calls (%d vs %d)", len(s1), len(s2))
	}
	for name, meta := range s1 {
		if other, ok := s2[name]; !ok {
			tb.Fatalf("flag %q missing on the second Make call", name)
		} else if other != meta {
			tb.Fatalf("flag %q schema differs across Make calls (%q vs %q)", name, meta, other)
		}
	}
}

// makeProvisioner calls pt.Make once, converting a panic or a nil
// provisioner into a test failure. The recover guard wraps only the
// user's factory call, so the harness's own failure signals pass
// through untouched.
func makeProvisioner(tb testing.TB, pt *ProvisionType, ordinal string) Provisioner {
	tb.Helper()
	prov, panicked := func() (p Provisioner, r any) {
		defer func() { r = recover() }()
		return pt.Make(), nil
	}()
	if panicked != nil {
		tb.Fatalf("Make panicked on the %s call: %v", ordinal, panicked)
	}
	if prov == nil {
		tb.Fatalf("Make returned nil on the %s call", ordinal)
	}
	return prov
}
