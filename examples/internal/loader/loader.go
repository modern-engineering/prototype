// Package loader supplies invocation loading and resource custody for the example commands.
package loader

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/program"
)

var (
	// ErrUnknownInput reports a binding for an undeclared input.
	ErrUnknownInput = errors.New("unknown input")
	// ErrUnknownOutput reports a requested output that was not declared.
	ErrUnknownOutput = errors.New("unknown output")
)

// Loader configures invocations and retains their borrowed resources.
type Loader struct {
	// Trace receives field names and help, without values. A nil Trace disables it.
	Trace io.Writer
}

// Invocation holds configured state and owns the readers borrowed by its runner.
// The caller must run or close each invocation.
type Invocation struct {
	runner  program.Runner
	outputs program.Outputs
	wiring  program.Wiring
}

// Load constructs and configures an invocation without running it. Unbound
// inputs retain their constructor values. Bindings provide one assignment per
// field, applied in declaration order. Binding names are checked before reading
// any source or assigning any target. A failed configuration releases all readers
// borrowed so far. The caller must Run or Close a successfully loaded invocation.
func (l Loader) Load(ctx context.Context, factory program.Factory, bindings map[string]program.Getter) (*Invocation, error) {
	runner, inputs, outputs := factory.Make()
	for name := range bindings {
		if _, ok := inputs.Lookup(name); !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownInput, name)
		}
	}
	if err := l.describe(inputs); err != nil {
		return nil, err
	}
	i := &Invocation{runner: runner, outputs: outputs}
	for input := range inputs.All() {
		name := input.Name
		if source, bound := bindings[name]; bound {
			if err := i.wiring.Wire(ctx, input.Value, source); err != nil {
				return nil, &assignmentError{name: name, cause: errors.Join(err, i.Close())}
			}
		}
	}
	return i, nil
}

// Run executes a configured invocation once. It keeps borrowed readers alive
// until the runner and its background work have stopped, then releases them even
// when execution fails or is canceled. Run must not follow Close.
func (i *Invocation) Run(ctx context.Context) (err error) {
	defer func() { err = errors.Join(err, i.Close()) }()
	return i.runner.Run(ctx)
}

// Close abandons a configured invocation or releases its completed execution.
// A loader that decides not to Run must call Close. Repeated calls do nothing.
func (i *Invocation) Close() error {
	return i.wiring.Close()
}

func (l Loader) describe(inputs program.Inputs) error {
	if l.Trace == nil {
		return nil
	}
	for input := range inputs.All() {
		field := input.Field
		if _, err := fmt.Fprintf(l.Trace, "input %s: %s\n", field.Name, field.Help); err != nil {
			return err
		}
	}
	return nil
}

// Output returns a declared output handle. Read it after a successful finite Run.
func (i *Invocation) Output(name string) (program.Getter, error) {
	output, ok := i.outputs.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownOutput, name)
	}
	return output.Value, nil
}

// assignmentError keeps setter diagnostics out of ordinary logging while
// retaining the cause for callers that explicitly inspect it.
type assignmentError struct {
	name  string
	cause error
}

func (e *assignmentError) Error() string { return fmt.Sprintf("input %s: assignment failed", e.name) }
func (e *assignmentError) Unwrap() error { return e.cause }
