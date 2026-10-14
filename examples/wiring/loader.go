package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/program"
)

var (
	errUnknownInput   = errors.New("unknown input")
	errMissingInput   = errors.New("required input has no binding")
	errUndecidedInput = errors.New("unbound input needs a resolution policy")
	errUnknownOutput  = errors.New("unknown output")
)

type loader struct {
	trace io.Writer
}

type invocation struct {
	runner  program.Runner
	inputs  program.Inputs
	outputs program.Outputs
}

// load constructs and configures an invocation without running it. This loader
// requires a binding for Required inputs and retains defaults for Optional
// inputs. It does not choose an absence policy for Unspecified inputs.
// Bindings provide one assignment per field, applied in declaration order.
// Binding names and absence policies are checked before reading any source or
// assigning any target.
func (l loader) load(factory program.Factory, bindings map[string]program.Getter) (*invocation, error) {
	runner, inputs, outputs := factory.Make()
	for name := range bindings {
		if _, ok := inputs.Lookup(name); !ok {
			return nil, fmt.Errorf("%w: %s", errUnknownInput, name)
		}
	}
	for input := range inputs.All() {
		field := input.Declaration()
		if _, bound := bindings[field.Name]; bound {
			continue
		}
		switch field.Requirement {
		case program.Optional:
		case program.Required:
			return nil, fmt.Errorf("%w: %s", errMissingInput, field.Name)
		default:
			return nil, fmt.Errorf("%w: %s", errUndecidedInput, field.Name)
		}
	}
	if err := l.describe(inputs); err != nil {
		return nil, err
	}
	for input := range inputs.All() {
		name := input.Declaration().Name
		if source, bound := bindings[name]; bound {
			if err := program.Wire(input.Value(), source); err != nil {
				return nil, &assignmentError{name: name, cause: err}
			}
		}
	}
	return &invocation{runner: runner, inputs: inputs, outputs: outputs}, nil
}

func (l loader) describe(inputs program.Inputs) error {
	if l.trace == nil {
		return nil
	}
	for input := range inputs.All() {
		field := input.Declaration()
		if _, err := fmt.Fprintf(l.trace, "input %s: %s\n", field.Name, field.Help); err != nil {
			return err
		}
		if suggestions, ok := input.(interface{ SuggestedValues() []string }); ok && !field.Sensitive {
			if _, err := fmt.Fprintln(l.trace, "  suggestions:", suggestions.SuggestedValues()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *invocation) output(name string) (program.Getter, error) {
	output, ok := i.outputs.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnknownOutput, name)
	}
	return output.Value(), nil
}

// assignmentError keeps setter diagnostics out of ordinary logging while
// retaining the cause for callers that explicitly inspect it.
type assignmentError struct {
	name  string
	cause error
}

func (e *assignmentError) Error() string { return fmt.Sprintf("input %s: assignment failed", e.name) }
func (e *assignmentError) Unwrap() error { return e.cause }

type literal string

func (value literal) Get() string { return string(value) }
