package loaderflags

import (
	"testing"
)

// CheckParser verifies that a [Parser] does not call Set on a specific flag that
// a higher-priority parser already claimed. The named flag is registered and set
// before calling [Parser.ParseParameters].
func CheckParser(t *testing.T, p Parser, name string) {
	fs := NewFlagSet("parser-test")
	// Register a guarded flag that fails the test if Set is called more than once:
	// the first call comes from fs.Set below (simulating a higher-priority parser),
	// and any subsequent call from ParseParameters is the violation under test.
	fs.Var(&guardValue{t: t, flag: name}, name, "pre-registered by test")
	_ = fs.Set(name, "")

	err := p.ParseParameters(fs)
	if err != nil {
		t.Fatal("ParseParameters() failed:", err)
	}
}

// A guardValue is a [flag.Value] that fails the test if Set is called after the
// flag has already been claimed.
type guardValue struct {
	t       *testing.T
	flag    string
	claimed bool
}

func (g *guardValue) String() string {
	return "" // Irrelevant.
}

func (g *guardValue) Set(string) error {
	g.t.Helper()
	if g.claimed {
		g.t.Errorf("ParseParameters() called Set on already-assigned flag %q", g.flag)
	} else {
		g.claimed = true
	}
	return nil
}
