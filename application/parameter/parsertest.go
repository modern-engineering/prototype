package parameter

import (
	"flag"
	"testing"
)

// TestParser verifies that a [Parser] does not call Set on flags that were
// already assigned by a higher-priority parser.
//
// It calls [Parser.ParseParameters] twice on the same [flag.FlagSet]: the first
// pass populates flags normally, and the second pass asserts that the parser
// skips every flag it already set.
func TestParser(t *testing.T, p Parser) {
	fs := NewFlagSet("parser-test")

	// We cannot pre-set specific flags by name because each Parser decides
	// which flags it sets. Calling the same parser twice sidesteps this:
	// the first pass populates exactly the flags the parser cares about,
	// and the second pass reveals whether it respects flags that are
	// already claimed.
	err := p.ParseParameters(fs)
	if err != nil {
		t.Fatal("first pass of ParseParameters() failed:", err)
	}

	// At least one flag must have been set, otherwise the test cannot
	// distinguish a well-behaved parser from one that ignores the flag-set
	// entirely.
	fs.Visit(func(*flag.Flag) {
		// Though called for each flag, which has been set, Fatal panics on the first one
		// being encountered.
		t.Fatal("parser did not set any flags; test scenario is flawed")
	})

	// Intercept every flag's Value so that any call to Set fails the test.
	fs.VisitAll(func(f *flag.Flag) {
		f.Value = guardValue{t: t, name: f.Name}
	})

	err = p.ParseParameters(fs)
	if err != nil {
		t.Fatal("second pass of ParseParameters() failed:", err)
	}
}

// guardValue wraps a [flag.Value] and fails the test if Set is called.
// Interface implementations like [BoolFlag] and [RequiredFlag] are preserved
// through the embedded field's promoted method set.
type guardValue struct {
	t    *testing.T
	name string
}

func (g guardValue) String() string {
	return ""
}

func (g guardValue) Set(string) error {
	g.t.Helper()
	g.t.Errorf("ParseParameters() called Set on already-assigned flag %q", g.name)
	return nil
}
