package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/program"
)

var (
	errUnknownInput  = errors.New("unknown input")
	errUnknownOutput = errors.New("unknown output")
)

type loader struct {
	trace io.Writer
}

type invocation struct {
	runner   program.Runner
	inputs   program.Inputs
	outputs  program.Outputs
	borrowed []io.Closer
}

// load constructs and configures an invocation without running it. Unbound
// inputs retain their constructor values. Bindings provide one assignment per
// field, applied in declaration order. Binding names are checked before reading
// any source or assigning any target. A failed configuration releases all readers
// borrowed so far. The caller must Run or Close a successfully loaded invocation.
func (l loader) load(ctx context.Context, factory program.Factory, bindings map[string]program.Getter) (*invocation, error) {
	runner, inputs, outputs := factory.Make()
	for name := range bindings {
		if _, ok := inputs.Lookup(name); !ok {
			return nil, fmt.Errorf("%w: %s", errUnknownInput, name)
		}
	}
	if err := l.describe(inputs); err != nil {
		return nil, err
	}
	i := &invocation{runner: runner, inputs: inputs, outputs: outputs}
	for input := range inputs.All() {
		name := input.Name
		if source, bound := bindings[name]; bound {
			closer, err := program.Wire(ctx, input.Value, source)
			if err != nil {
				return nil, &assignmentError{name: name, cause: errors.Join(err, i.Close())}
			}
			if closer != nil {
				i.borrowed = append(i.borrowed, closer)
			}
		}
	}
	return i, nil
}

// Run executes a configured invocation once. It keeps borrowed readers alive
// until the runner and its background work have stopped, then releases them even
// when execution fails or is canceled. Run must not follow Close.
func (i *invocation) Run(ctx context.Context) (err error) {
	defer func() { err = errors.Join(err, i.Close()) }()
	return i.runner.Run(ctx)
}

// Close abandons a configured invocation or releases its completed execution.
// A loader that decides not to Run must call Close. Repeated calls do nothing.
func (i *invocation) Close() error {
	var err error
	for index := len(i.borrowed) - 1; index >= 0; index-- {
		err = errors.Join(err, i.borrowed[index].Close())
	}
	i.borrowed = nil
	return err
}

func (l loader) describe(inputs program.Inputs) error {
	if l.trace == nil {
		return nil
	}
	for input := range inputs.All() {
		field := input.Field
		if _, err := fmt.Fprintf(l.trace, "input %s: %s\n", field.Name, field.Help); err != nil {
			return err
		}
	}
	return nil
}

func (i *invocation) output(name string) (program.Getter, error) {
	output, ok := i.outputs.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnknownOutput, name)
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

type literal string

func (value literal) Get() string { return string(value) }
