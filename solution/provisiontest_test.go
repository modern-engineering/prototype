// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution"
)

// A stubProvisioner is the smallest Provisioner the rejection cases
// can shape: whatever FlagSet it is given back verbatim, a driver that
// never runs.
type stubProvisioner struct {
	flags *flag.FlagSet
}

func (p *stubProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *stubProvisioner) Attach(context.Context, *solution.OutputWriter) error { return nil }

func TestCheckProvisionTypeAcceptsCitizens(t *testing.T) {
	// The compile-test catalogue's citizens double as the harness's
	// positive cases: a flagged type, and a Make-less one (nil Make is
	// a complete dry citizen; only enactment refuses it).
	solution.CheckProvisionType(t, busProvisionType())
	solution.CheckProvisionType(t, storeProvisionType())

	// A flagless provisioner is legal too: nil Flags means no
	// parameters, uniform with a component's Main adapter.
	solution.CheckProvisionType(t, &solution.ProvisionType{
		Doc:   "flagless",
		Make:  func() solution.Provisioner { return &stubProvisioner{} },
		Kinds: solution.Attach,
	})
}

// failer satisfies testing.TB for asserting CheckProvisionType
// failures; Fatal* panics with a sentinel so the checked flow stops
// the way testing's Goexit would.
type failer struct {
	testing.TB
	failed string
}

type failerStop struct{}

func (f *failer) Helper() {}
func (f *failer) Fatal(args ...any) {
	f.failed = fmt.Sprint(args...)
	panic(failerStop{})
}
func (f *failer) Fatalf(format string, args ...any) {
	f.failed = fmt.Sprintf(format, args...)
	panic(failerStop{})
}

func checkFails(t *testing.T, pt *solution.ProvisionType) string {
	t.Helper()
	f := &failer{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(failerStop); !ok {
					panic(r)
				}
			}
		}()
		solution.CheckProvisionType(f, pt)
	}()
	if f.failed == "" {
		t.Fatal("CheckProvisionType accepted a provision type that violates the contract")
	}
	return f.failed
}

func TestCheckProvisionTypeRejectsViolations(t *testing.T) {
	sharedFS := flag.NewFlagSet("shared", flag.ContinueOnError)
	sharedFS.String("cluster", "", "shared across calls")
	singleton := &stubProvisioner{}

	tests := []struct {
		name string
		pt   *solution.ProvisionType
		want string
	}{
		{"nil provision type", nil, "provision type is nil"},
		{"no kinds", &solution.ProvisionType{}, "registers no kinds"},
		{"unknown kind bits", &solution.ProvisionType{Kinds: 1 << 5}, "unknown kinds"},
		{"invalid output name", &solution.ProvisionType{Kinds: solution.Attach,
			Outputs: []solution.Output{{Name: "not name"}}}, "not a valid Go identifier"},
		{"duplicate output", &solution.ProvisionType{Kinds: solution.Attach,
			Outputs: []solution.Output{{Name: "dsn"}, {Name: "dsn"}}}, "declares output dsn twice"},
		{"unknown output type", &solution.ProvisionType{Kinds: solution.Attach,
			Outputs: []solution.Output{{Name: "dsn", Type: "float"}}}, `unknown type "float"`},
		{"panicking make", &solution.ProvisionType{Kinds: solution.Attach,
			Make: func() solution.Provisioner { panic("factory exploded") }}, "Make panicked"},
		{"nil provisioner", &solution.ProvisionType{Kinds: solution.Attach,
			Make: func() solution.Provisioner { return nil }}, "returned nil"},
		{"singleton provisioner", &solution.ProvisionType{Kinds: solution.Attach,
			Make: func() solution.Provisioner { return singleton }}, "same provisioner"},
		{"shared flag set", &solution.ProvisionType{Kinds: solution.Attach,
			Make: func() solution.Provisioner { return &stubProvisioner{flags: sharedFS} }},
			"share one FlagSet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFails(t, tt.pt)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("failure message %q does not contain %q", got, tt.want)
			}
		})
	}
}
