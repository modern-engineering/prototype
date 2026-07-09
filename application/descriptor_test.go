package application_test

import (
	"context"
	"flag"
	"testing"

	"github.com/modern-engineering/prototype/application"
)

// pingLike builds a descriptor the way application authors do: a pure Make
// that declares flags on a fresh service per call.
func pingLike() *application.Descriptor {
	return &application.Descriptor{
		Name: "ping",
		Doc:  "ping emits a payload at a fixed interval",
		Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
			fs := flag.NewFlagSet("ping", flag.ContinueOnError)
			fs.Int("count", 0, "number of pings; 0 pings forever")
			fs.Duration("interval", 0, "delay between pings")
			return application.Main(func(ctx context.Context) error { return nil }), fs
		}),
	}
}

func TestDescriptorFlagsDryInstantiation(t *testing.T) {
	d := pingLike()

	first, second := d.Flags(), d.Flags()
	if first == nil || second == nil {
		t.Fatal("Flags returned nil for a descriptor that declares parameters")
	}
	if first == second {
		t.Fatal("Flags returned the same FlagSet twice; dry instantiation must yield a fresh surface per call")
	}

	names := func(fs *flag.FlagSet) []string {
		var out []string
		fs.VisitAll(func(f *flag.Flag) { out = append(out, f.Name) })
		return out
	}
	got, want := names(first), []string{"count", "interval"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("declared flags = %v, want %v", got, want)
	}

	// Mutating one dry surface must not leak into another.
	if err := first.Set("count", "3"); err != nil {
		t.Fatalf("Set on a dry surface: %v", err)
	}
	if v := second.Lookup("count").Value.String(); v != "0" {
		t.Fatalf("fresh surface saw %q for count, want the default 0", v)
	}
}

func TestDescriptorFlagsNilForFlagless(t *testing.T) {
	d := &application.Descriptor{
		Name: "noop",
		Make: func() application.Service {
			return application.Main(func(ctx context.Context) error { return nil })
		},
	}
	if fs := d.Flags(); fs != nil {
		t.Fatalf("Flags = %v, want nil for a flagless application", fs)
	}
}

func TestDescriptorFlagsPanicsWithoutMake(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Flags did not panic for a descriptor without Make")
		}
	}()
	(&application.Descriptor{Name: "broken"}).Flags()
}
