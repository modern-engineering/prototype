// Package loader supplies invocation loading and resource custody for the example commands.
package loader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/modern-engineering/prototype/program"
)

var (
	// ErrUnknownInput reports a requested input that was not declared.
	ErrUnknownInput = errors.New("unknown input")
	// ErrUnknownOutput reports a requested output that was not declared.
	ErrUnknownOutput = errors.New("unknown output")
)

// Loader constructs invocations and supplies files for their configuration.
// The zero value uses the operating system's filesystem and emits no diagnostics.
type Loader struct {
	// FS supplies files to ReadFile. A nil FS uses os.Open.
	FS fs.FS

	// Logger enables debug diagnostics for assignment and execution boundaries.
	// A nil Logger disables diagnostics, including use of slog.Default.
	// Diagnostics contain field names and failure status, never values or causes.
	Logger *slog.Logger
}

// Invocation holds configured state and owns the readers borrowed by its runner.
// The caller must run or close each invocation.
type Invocation struct {
	runner   program.Runner
	inputs   program.Inputs
	outputs  program.Outputs
	fs       fs.FS
	borrowed []io.Closer
	logger   *slog.Logger
}

// Make constructs a fresh invocation without acquiring resources or running it.
// Unbound inputs retain their constructor values.
// The caller must Run or Close the invocation after configuring it.
func (l Loader) Make(descriptor program.Descriptor) *Invocation {
	runner, inputs, outputs := descriptor.Make()
	return &Invocation{runner: runner, inputs: inputs, outputs: outputs, fs: l.FS, logger: l.Logger}
}

// Input returns a declared input's setter for explicit assignment.
func (i *Invocation) Input(name string) (program.Setter, error) {
	input, ok := i.inputs.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownInput, name)
	}
	return input.Value, nil
}

// ReadFile opens a file and assigns its contents to an input.
// An io.ReaderFrom consumes eagerly; otherwise BorrowReader(io.Reader) error
// borrows through Run. Other setters receive the file contents as a string.
// Readers expose no closing capability: this invocation closes eager readers
// immediately and retains successful borrows until Run returns or Close is called.
// A failed assignment closes its file; earlier borrows remain usable.
// Cancellation is checked before opening the file, but does not interrupt file I/O.
func (i *Invocation) ReadFile(ctx context.Context, name, path string) (err error) {
	defer func() {
		if i.logger != nil {
			i.logger.DebugContext(ctx, "input assignment completed", "input", name, "failed", err != nil)
		}
	}()
	destination, err := i.Input(name)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var file fs.File
	if i.fs == nil {
		file, err = os.Open(path)
	} else {
		file, err = i.fs.Open(path)
	}
	if err != nil {
		return &assignmentError{name: name, cause: err}
	}
	retained := false
	defer func() {
		if !retained {
			err = errors.Join(err, file.Close())
		}
		if err != nil {
			err = &assignmentError{name: name, cause: err}
		}
	}()
	reader := struct{ io.Reader }{file}
	switch destination := destination.(type) {
	case io.ReaderFrom:
		_, err = destination.ReadFrom(reader)
	case interface{ BorrowReader(io.Reader) error }:
		err = destination.BorrowReader(reader)
		if err == nil {
			i.borrowed = append(i.borrowed, file)
			retained = true
		}
	default:
		var content []byte
		content, err = io.ReadAll(reader)
		if err == nil {
			err = destination.Set(string(content))
		}
	}
	return err
}

// Run executes a configured invocation once. It keeps borrowed readers alive
// until the runner and its background work have stopped, then releases them even
// when execution fails or is canceled. Run must not follow Close.
func (i *Invocation) Run(ctx context.Context) (err error) {
	defer func() {
		err = errors.Join(err, i.Close())
		if i.logger != nil {
			i.logger.DebugContext(ctx, "invocation completed", "failed", err != nil)
		}
	}()
	return i.runner.Run(ctx)
}

// Close abandons a configured invocation or releases its completed execution.
// A loader that decides not to Run must call Close. Repeated calls do nothing.
func (i *Invocation) Close() (err error) {
	for n := len(i.borrowed) - 1; n >= 0; n-- {
		err = errors.Join(err, i.borrowed[n].Close())
	}
	i.borrowed = nil
	return err
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
