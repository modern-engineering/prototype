package application_test

import (
	"context"
	"flag"
	"testing"

	"github.com/modern-engineering/prototype/application"
)

// MakeFor's constraint (PT interface{ *T; Runner; Flags() ... }) is
// deliberately satisfiable by both receiver conventions, and the two
// service shapes below pin one each:
//
//   - gauge declares its methods on the pointer receiver — the common
//     shape, where Flags binds fields of the pointed-at instance, so
//     the factory must hand out the pointer for bound values to reach
//     the service's own state;
//   - beacon declares its methods on the value receiver — the method
//     set promotes to *beacon, so the same constraint admits it and
//     the factory still returns a pointer to a fresh zero value.
//
// Losing either side would break real citizens: pointer receivers are
// the norm for services with parameters, while value-receiver services
// must not be rejected by an over-tight constraint.

// A gauge is the pointer-receiver shape: Flags binds into the instance.
type gauge struct{ level int }

func (g *gauge) Run(context.Context) error { return nil }

func (g *gauge) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("gauge", flag.ContinueOnError)
	fs.IntVar(&g.level, "level", 0, "fill level")
	return fs
}

// A beacon is the value-receiver shape on a scalar underlying type.
type beacon bool

func (beacon) Run(context.Context) error { return nil }
func (beacon) Flags() *flag.FlagSet      { return nil }

// Every call to a MakeFor factory constructs a distinct service around
// a fresh zero value, and the returned pointer reaches the service's
// own state: a flag bound on one instance never bleeds into the next.
// MakeFunc, the adapter for runner+flags pairs, passes both through
// verbatim.
func TestFactoriesConstructFreshServices(t *testing.T) {
	makeGauge := application.MakeFor[gauge]()
	first, second := makeGauge().(*gauge), makeGauge().(*gauge)
	if first == second {
		t.Fatal("the factory returned the same instance twice; every call must construct afresh")
	}
	if first.level != 0 {
		t.Fatalf("fresh gauge level = %d, want the zero value", first.level)
	}
	if err := first.Flags().Set("level", "9"); err != nil {
		t.Fatalf("Set on the first gauge's flags: %v", err)
	}
	if first.level != 9 {
		t.Fatalf("first gauge level = %d after Set; flags must bind the pointed-at instance", first.level)
	}
	if second.level != 0 {
		t.Fatalf("second gauge level = %d, want 0; instances must not share state", second.level)
	}

	if lit := *application.MakeFor[beacon]()().(*beacon); lit {
		t.Fatalf("fresh beacon = %v, want the zero value", lit)
	}

	var ran bool
	pairFS := flag.NewFlagSet("pair", flag.ContinueOnError)
	svc := application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		return application.RunnerFunc(func(context.Context) error { ran = true; return nil }), pairFS
	})()
	if svc.Flags() != pairFS {
		t.Fatal("MakeFunc's service must expose the declared FlagSet verbatim")
	}
	if err := svc.Run(context.Background()); err != nil || !ran {
		t.Fatalf("Run() = %v, ran = %v; MakeFunc must delegate to the declared runner", err, ran)
	}
}
