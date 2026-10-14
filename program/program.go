// Package program defines invocation and bound input/output contracts shared by
// catalogue types and loaders.
//
// A factory constructs fresh state without performing I/O. A loader configures
// that state through its input handles before invoking the runner, then reads
// the output handles after a successful finite invocation. Input mutation and
// execution are separate phases; these handles provide no synchronization.
package program

import (
	"context"
	"iter"
)

// Runner executes an invocation using the state bound to its input handles.
// It may validate that state and return an error independently of loader checks.
type Runner interface {
	Run(context.Context) error
}

// RunnerFunc adapts a function to Runner.
type RunnerFunc func(context.Context) error

func (f RunnerFunc) Run(ctx context.Context) error { return f(ctx) }

// Factory constructs a runner and the handles bound to the same fresh state.
// Make is pure and infallible: it neither acquires external resources nor runs
// the program. Repeated calls must create independent invocation state.
type Factory interface {
	Make() (Runner, Inputs, Outputs)
}

// MakeFunc adapts a tuple constructor to Factory.
type MakeFunc func() (Runner, Inputs, Outputs)

func (f MakeFunc) Make() (Runner, Inputs, Outputs) { return f() }

// Adapt creates a factory from a constructor whose runner optionally implements
// Inputs() Inputs and Outputs() Outputs. Each accessor is called once per Make;
// missing accessors produce empty surfaces. Construction follows Factory's
// purity and fresh-state contract.
func Adapt[T Runner](newRunner func() T) MakeFunc {
	return func() (Runner, Inputs, Outputs) {
		runner := newRunner()
		var inputs Inputs = InputFields(nil)
		var outputs Outputs = OutputFields(nil)
		if r, ok := any(runner).(interface{ Inputs() Inputs }); ok {
			inputs = r.Inputs()
		}
		if r, ok := any(runner).(interface{ Outputs() Outputs }); ok {
			outputs = r.Outputs()
		}
		return runner, inputs, outputs
	}
}

// Requirement records an explicit requiredness declaration. Its zero value
// leaves the unmarked case undecided. How a loader determines that a required
// value is available belongs to its input/output resolution policy.
type Requirement uint8

const (
	// Unspecified leaves requiredness to an explicitly chosen loader policy.
	Unspecified Requirement = iota
	// Required declares that the field's value is required.
	Required
	// Optional declares that the field's value is optional.
	Optional
)

// Field describes a named input or output. Requirement and Sensitive communicate
// declaration semantics to loaders. Their enforcement is loader policy; this
// package does not track assignment, output production, or sensitivity flow.
type Field struct {
	Name        string
	Help        string
	Requirement Requirement
	Sensitive   bool
}

// Setter receives a string into the Go state used by a runner. If Set is never
// called, the constructor's default is retained. Repeated calls permit replacement,
// accumulation, or rejection determined by the implementation.
// Implementations may expose additional transfer capabilities.
type Setter interface {
	Set(string) error
}

// SetterFunc adapts a function to Setter.
type SetterFunc func(string) error

func (f SetterFunc) Set(value string) error { return f(value) }

// Getter provides the current output value as a string. It does not distinguish
// an unproduced value from an empty value. Implementations may expose additional
// transfer capabilities.
type Getter interface {
	Get() string
}

// GetterFunc adapts a function to Getter.
type GetterFunc func() string

func (f GetterFunc) Get() string { return f() }

// Input combines declaration metadata with a bound setter. Adopter-specific
// declaration capabilities may be implemented on Input; transfer capabilities
// belong to its Value.
type Input interface {
	Declaration() Field
	Value() Setter
}

// Output combines declaration metadata with a bound getter. Adopter-specific
// declaration capabilities may be implemented on Output; transfer capabilities
// belong to its Value.
type Output interface {
	Declaration() Field
	Value() Getter
}

// Inputs enumerates and looks up the uniquely named inputs of an invocation.
// All and Lookup expose the same bound fields.
type Inputs interface {
	All() iter.Seq[Input]
	Lookup(name string) (Input, bool)
}

// Outputs enumerates and looks up the uniquely named outputs of an invocation.
// All and Lookup expose the same bound fields.
type Outputs interface {
	All() iter.Seq[Output]
	Lookup(name string) (Output, bool)
}
