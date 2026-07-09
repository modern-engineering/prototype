package application_test

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
)

func TestCheckDescriptorAcceptsCitizens(t *testing.T) {
	application.CheckDescriptor(t, pingLike())
	application.CheckDescriptor(t, named("flagless"))
}

// failer satisfies testing.TB for asserting CheckDescriptor failures; Fatal*
// panics with a sentinel so the checked flow stops the way testing's Goexit
// would.
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

func checkFails(t *testing.T, d *application.Descriptor) string {
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
		application.CheckDescriptor(f, d)
	}()
	if f.failed == "" {
		t.Fatal("CheckDescriptor accepted a descriptor that violates the contract")
	}
	return f.failed
}

func TestCheckDescriptorRejectsViolations(t *testing.T) {
	sharedFS := flag.NewFlagSet("shared", flag.ContinueOnError)
	sharedFS.Int("count", 0, "shared across calls")
	noop := application.Main(func(ctx context.Context) error { return nil })

	tests := []struct {
		name string
		d    *application.Descriptor
		want string
	}{
		{"nil descriptor", nil, "descriptor is nil"},
		{"bad name", &application.Descriptor{Name: "no-good"}, "not a valid Go identifier"},
		{"missing make", &application.Descriptor{Name: "bare"}, "no Make factory"},
		{"panicking make", &application.Descriptor{Name: "boom",
			Make: func() application.Service { panic("constructor exploded") }}, "Make panicked"},
		{"nil service", &application.Descriptor{Name: "empty",
			Make: func() application.Service { return nil }}, "returned nil"},
		{"shared flag set", &application.Descriptor{Name: "leaky",
			Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) { return noop, sharedFS })},
			"share one FlagSet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFails(t, tt.d)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("failure message %q does not contain %q", got, tt.want)
			}
		})
	}
}
