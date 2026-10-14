// Package program defines invocation and bound input/output contracts shared by
// catalogue types and loaders.
//
// A descriptor constructs fresh state without performing I/O. A loader configures
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

// Descriptor constructs a runner and the handles bound to the same fresh state.
// Make is pure and infallible: it neither acquires external resources nor runs
// the program. Repeated calls must create independent invocation state.
type Descriptor interface {
	Make() (Runner, Inputs, Outputs)
}

// MakeFunc adapts a tuple constructor to Descriptor.
type MakeFunc func() (Runner, Inputs, Outputs)

func (f MakeFunc) Make() (Runner, Inputs, Outputs) { return f() }

// Field describes a named input or output. Sensitive marks values that loaders
// must keep out of ordinary diagnostics. This package does not track assignment,
// output production, or sensitivity flow.
type Field struct {
	Name      string
	Help      string
	Sensitive bool
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

// Input binds declaration metadata to the original setter, preserving any
// additional transfer capabilities exposed by Value.
type Input struct {
	Field
	Value Setter
}

// Inputs enumerates and looks up the uniquely named inputs of an invocation.
// All and Lookup expose the same bound fields.
type Inputs interface {
	All() iter.Seq[Input]
	Lookup(name string) (Input, bool)
}

// InputList is an input surface in declaration order. Field names must be
// unique. A nil slice is an empty surface.
type InputList []Input

func (fields InputList) All() iter.Seq[Input] {
	return func(yield func(Input) bool) {
		for _, field := range fields {
			if !yield(field) {
				return
			}
		}
	}
}

func (fields InputList) Lookup(name string) (Input, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return Input{}, false
}

// Getter provides the current output value as a string. It does not distinguish
// an unproduced value from an empty value. Implementations may expose additional
// transfer capabilities.
type Getter interface {
	Get() string
}

// GetterFunc adapts a function to Getter.
type GetterFunc func() string

func (f GetterFunc) Get() string { return f() }

// Output binds declaration metadata to the original getter, preserving any
// additional transfer capabilities exposed by Value.
type Output struct {
	Field
	Value Getter
}

// Outputs enumerates and looks up the uniquely named outputs of an invocation.
// All and Lookup expose the same bound fields.
type Outputs interface {
	All() iter.Seq[Output]
	Lookup(name string) (Output, bool)
}

// OutputList is an output surface in declaration order. Field names must be
// unique. A nil slice is an empty surface.
type OutputList []Output

func (fields OutputList) All() iter.Seq[Output] {
	return func(yield func(Output) bool) {
		for _, field := range fields {
			if !yield(field) {
				return
			}
		}
	}
}

func (fields OutputList) Lookup(name string) (Output, bool) {
	for _, field := range fields {
		if field.Name == name {
			return field, true
		}
	}
	return Output{}, false
}
