package loader_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// Configuration must leave borrowed files usable through execution, while both
// completed and abandoned invocations release them.
func TestBorrowedReadersCloseAtInvocationBoundary(t *testing.T) {
	source := fileSource(os.DevNull)
	factory := new(readProgram)
	i, err := (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{"borrowed": source})
	if err != nil {
		t.Fatal(err)
	}
	reader := factory.value.reader
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("configured reader: %v", err)
	}
	if err := i.Run(t.Context()); err != nil {
		t.Fatalf("run with borrowed reader: %v", err)
	}
	if _, err := io.ReadAll(reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after Run = %v; want closed file", err)
	}

	i, err = (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{"borrowed": source})
	if err != nil {
		t.Fatal(err)
	}
	reader = factory.value.reader
	if err := i.Close(); err != nil {
		t.Fatalf("abandon invocation: %v", err)
	}
	if _, err := io.ReadAll(reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after abandonment = %v; want closed file", err)
	}
	if err := i.Close(); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
}

// A later assignment can fail after the loader has already acquired readers;
// those readers must be released even though no invocation reaches the caller.
func TestBindingFailureReleasesEarlierBorrow(t *testing.T) {
	rejected := errors.New("assignment rejected")
	var borrowed *borrowedValue
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		borrowed = &borrowedValue{}
		inputs := program.InputList{
			{Field: program.Field{Name: "reader"}, Value: borrowed},
			{Field: program.Field{Name: "rejected"}, Value: program.SetterFunc(func(string) error { return rejected })},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("failed configuration reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	i, err := (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{
		"reader":   fileSource(os.DevNull),
		"rejected": literal("value"),
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("load error = %v; want assignment rejection", err)
	}
	if i != nil {
		t.Fatal("failed configuration returned an invocation")
	}
	if _, err := io.ReadAll(borrowed.reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after failed configuration = %v; want closed file", err)
	}
}

// Cancellation alone does not end a borrow: the runner must return before the
// loader can release a reader that the runner might still be using.
func TestCanceledRunReleasesBorrow(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	factory := new(readProgram)
	i, err := (loader.Loader{}).Load(ctx, factory, map[string]program.Getter{
		"borrowed": fileSource(os.DevNull),
	})
	if err != nil {
		t.Fatal(err)
	}
	reader := factory.value.reader
	cancel()
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("reader after cancellation before Run returns: %v", err)
	}
	if err := i.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v; want context cancellation", err)
	}
	if _, err := io.ReadAll(reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after canceled Run = %v; want closed file", err)
	}
}

// readProgram captures the most recently constructed input so tests can observe
// the lifetime of its borrowed file without exposing loader internals.
type readProgram struct {
	value *borrowedValue
}

func (p *readProgram) Make() (program.Runner, program.Inputs, program.Outputs) {
	value := new(borrowedValue)
	p.value = value
	run := program.RunnerFunc(func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := io.Copy(io.Discard, value.reader)
		return err
	})
	return run, program.InputList{{Field: program.Field{Name: "borrowed"}, Value: value}}, program.OutputList(nil)
}

type borrowedValue struct {
	reader io.Reader
}

func (v *borrowedValue) Set(text string) error {
	v.reader = strings.NewReader(text)
	return nil
}

func (v *borrowedValue) BorrowReader(reader io.Reader) error {
	v.reader = reader
	return nil
}

type fileSource string

func (s fileSource) Get() string                                 { return "" }
func (s fileSource) Open(context.Context) (io.ReadCloser, error) { return os.Open(string(s)) }
