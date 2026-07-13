package loaderflags

import (
	"testing"
)

// CheckParser verifies that a [Parser] does not call Set on a specific flag that
// a higher-priority parser already claimed. The named flag is registered and set
// before calling [Parser.ParseParameters]; any Set the parser then issues on it
// fails the test. Parser authors call it from their own tests, once per source.
func CheckParser(tb testing.TB, p Parser, name string) {
	tb.Helper()
	fs := NewFlagSet("parser-test")
	// Register a guarded flag that fails the test if Set is called more than once:
	// the first call comes from fs.Set below (simulating a higher-priority parser),
	// and any subsequent call from ParseParameters is the violation under test.
	fs.Var(&guardValue{tb: tb, flag: name}, name, "pre-registered by test")
	_ = fs.Set(name, "")

	err := p.ParseParameters(fs)
	if err != nil {
		tb.Fatal("ParseParameters() failed:", err)
	}
}

// A guardValue is a [flag.Value] that fails the test if Set is called after the
// flag has already been claimed.
type guardValue struct {
	tb      testing.TB
	flag    string
	claimed bool
}

func (g *guardValue) String() string {
	return "" // Irrelevant.
}

func (g *guardValue) Set(string) error {
	g.tb.Helper()
	if g.claimed {
		g.tb.Errorf("ParseParameters() called Set on already-assigned flag %q", g.flag)
	} else {
		g.claimed = true
	}
	return nil
}
