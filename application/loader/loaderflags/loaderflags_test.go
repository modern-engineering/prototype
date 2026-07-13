package loaderflags_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application/loader/loaderflags"
)

// A mapSource is the canonical Parser shape: it populates every
// registered flag it has a value for and honors the priority contract
// by checking IsSet before writing (solution/host's recordSource is
// the production twin).
type mapSource struct {
	name   string
	values map[string]string
}

func (m mapSource) ParseParameters(fs *flag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}
		v, ok := m.values[f.Name]
		if !ok || loaderflags.IsSet(fs, f.Name) {
			return
		}
		err = fs.Set(f.Name, v)
	})
	return err
}

func (m mapSource) String() string { return m.name }

// Parse populates a parameter set from its sources in priority order:
// an earlier source's claim survives a later source's value, each
// source fills what remains, flags no source mentions keep their
// defaults, and the set comes back Parsed. IsSet answers claimed-ness
// per flag and PendingParams walks exactly the unclaimed remainder —
// the surface a host uses to refuse required-but-unbound parameters.
func TestEarlierSourcesWin(t *testing.T) {
	fs := loaderflags.NewFlagSet("ping")
	endpoint := fs.String("endpoint", "", "where to dial")
	count := fs.Int("count", 3, "how many pings")
	subject := fs.String("subject", "hello", "what to say")
	fs.String("token", "", "credential nobody binds")

	record := mapSource{"record", map[string]string{"endpoint": "nats://one", "count": "7"}}
	fallback := mapSource{"fallback", map[string]string{"endpoint": "nats://two", "subject": "fallback"}}
	if err := loaderflags.Parse(fs, record, fallback); err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if *endpoint != "nats://one" {
		t.Errorf("endpoint = %q; the record source came first and must win", *endpoint)
	}
	if *count != 7 || *subject != "fallback" {
		t.Errorf("count, subject = %d, %q; want 7, %q — each source fills what remains", *count, *subject, "fallback")
	}
	if !fs.Parsed() {
		t.Error("Parse left the set unparsed")
	}

	if !loaderflags.IsSet(fs, "endpoint") || loaderflags.IsSet(fs, "token") {
		t.Error("IsSet must report exactly the bound flags")
	}
	var pending []string
	for f := range loaderflags.PendingParams(fs) {
		pending = append(pending, f.Name)
	}
	if len(pending) != 1 || pending[0] != "token" {
		t.Errorf("PendingParams = %v, want [token]", pending)
	}
}

// Parse guards its own preconditions: a flag set born outside
// NewFlagSet panics (a foreign set may ExitOnError mid-parse), a
// second Parse refuses with an error, and a source's failure surfaces
// wrapped in that source's own name.
func TestParseRefusals(t *testing.T) {
	t.Run("foreign set", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("Parse accepted a flag set born outside NewFlagSet")
			}
		}()
		_ = loaderflags.Parse(flag.NewFlagSet("foreign", flag.ContinueOnError))
	})

	t.Run("already parsed", func(t *testing.T) {
		fs := loaderflags.NewFlagSet("once")
		if err := loaderflags.Parse(fs); err != nil {
			t.Fatalf("first Parse: %v", err)
		}
		if err := loaderflags.Parse(fs); err == nil {
			t.Fatal("second Parse must refuse an already-parsed set")
		}
	})

	t.Run("failing source", func(t *testing.T) {
		fs := loaderflags.NewFlagSet("faulty")
		fs.Int("count", 0, "an int")
		err := loaderflags.Parse(fs, mapSource{"badsource", map[string]string{"count": "many"}})
		if err == nil || !strings.Contains(err.Error(), "badsource") {
			t.Fatalf("Parse = %v, want the failing source named", err)
		}
	})
}

// Defaults renders the PrintDefaults listing as a string and puts the
// set's output sink back untouched; NewFlagSet labels the usage output
// with the display name rather than the shared internal set name.
func TestUsageSurface(t *testing.T) {
	fs := loaderflags.NewFlagSet("ping")
	fs.String("endpoint", "nats://localhost", "where to dial")

	var sink strings.Builder
	fs.SetOutput(&sink)

	defaults := loaderflags.Defaults(fs)
	if !strings.Contains(defaults, "-endpoint") || !strings.Contains(defaults, "where to dial") {
		t.Errorf("Defaults() = %q, want the endpoint flag listed", defaults)
	}
	if sink.Len() != 0 {
		t.Errorf("Defaults leaked %q into the set's own sink", sink.String())
	}

	fs.Usage()
	if got := sink.String(); !strings.Contains(got, "Parameterization of ping:") {
		t.Errorf("Usage wrote %q, want it labeled with the display name", got)
	}
}
