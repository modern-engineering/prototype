// Package loader supplies invocation loading and resource custody for the example commands.
package loader

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// Loader constructs invocations with optional diagnostics. Its zero value is
// ready to use and produces no diagnostics.
type Loader struct {
	// Logger enables debug diagnostics for file assignment and execution.
	// A nil Logger disables diagnostics, including use of slog.Default.
	// Diagnostics contain field names and failure status, never values or causes.
	Logger *slog.Logger
}

// Invocation holds configured state and owns the readers borrowed by its runner.
// The caller must run or close each invocation.
type Invocation struct {
	runner  program.Runner
	inputs  program.Inputs
	outputs program.Outputs
	files   []*os.File
	logger  *slog.Logger
}

// Make constructs fresh invocation state without I/O or assignment. Configure
// its inputs before Run; untouched inputs retain their constructor values.
// The caller must Run or Close each invocation, including on setup failure.
func (l Loader) Make(descriptor program.Descriptor) *Invocation {
	runner, inputs, outputs := descriptor.Make()
	return &Invocation{runner: runner, inputs: inputs, outputs: outputs, logger: l.Logger}
}

// Input returns the original setter for a declared input. The caller can assign
// literals or copy an output with input.Set(output.Get()). Additional transfer
// capabilities remain available on that setter. The caller handles its errors;
// these direct assignments produce no loader diagnostics.
func (i *Invocation) Input(name string) (program.Setter, error) {
	input, ok := i.inputs.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownInput, name)
	}
	return input.Value, nil
}

// ReadFile configures an input from the contents of path. It checks the input
// name and context before opening the file, and owns every file it opens.
//
// An io.ReaderFrom consumes the file during this call. A setter implementing
// BorrowReader(io.Reader) error may retain the reader through Run. If both
// capabilities exist, ReadFrom wins. Otherwise ReadFile reads the whole file
// and calls Set with its contents. Destinations receive only io.Reader and must
// not close it. Eager destinations must finish using it before returning; a
// borrower must finish before Run returns, including its background work.
//
// Eager reads and failed assignments close the file before returning. A successful
// borrow stays open until Run returns or Close abandons the invocation. Failure
// leaves earlier borrows intact; the caller must close an abandoned invocation.
// Assignment and close errors are joined. A failed assignment is not retried
// through another capability. The context does not make file reads cancelable.
func (i *Invocation) ReadFile(ctx context.Context, name, path string) (err error) {
	target, err := i.Input(name)
	if err != nil {
		return err
	}
	defer func() {
		if i.logger != nil {
			i.logger.DebugContext(ctx, "file assignment completed", "input", name, "failed", err != nil)
		}
		if err != nil {
			err = &assignmentError{name: name, cause: err}
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	retained := false
	defer func() {
		if !retained {
			err = errors.Join(err, file.Close())
		}
	}()
	reader := struct{ io.Reader }{file}
	switch target := target.(type) {
	case io.ReaderFrom:
		_, err = target.ReadFrom(reader)
		return err
	case readerBorrower:
		if err := target.BorrowReader(reader); err != nil {
			return err
		}
		i.files = append(i.files, file)
		retained = true
		return nil
	default:
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		return target.Set(string(data))
	}
}

// readerBorrower is the reader convention understood by this loader. A failed
// call must stop using the reader before returning.
type readerBorrower interface {
	BorrowReader(io.Reader) error
}

// Run executes a configured invocation once. It keeps borrowed readers alive
// until the runner and its background work have stopped, then releases them even
// when execution fails or is canceled. Cancellation alone does not release files;
// Run must return first. Run must not follow Close.
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
func (i *Invocation) Close() error {
	var err error
	for index := len(i.files) - 1; index >= 0; index-- {
		err = errors.Join(err, i.files[index].Close())
	}
	i.files = nil
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
