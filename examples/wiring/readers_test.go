package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/modern-engineering/prototype/program"
)

// A loader closes an eagerly consumed reader during configuration and keeps a
// borrowed reader open until the invocation finishes.
func Example_readers() {
	if err := runReaders(context.Background(), os.Stdout); err != nil {
		fmt.Println(err)
	}
	// Output:
	// open 1
	// consume eager
	// close 1
	// open 2
	// borrow reader
	// run: eager = "Hello, Ada!"
	// run: borrowed = "Hello, Ada!"
	// close 2
}

// Configuration must leave borrowed files usable through execution, while both
// completed and abandoned invocations release them.
func TestBorrowedReadersCloseAtInvocationBoundary(t *testing.T) {
	source := &fileContent{path: os.DevNull, trace: io.Discard}
	i, err := (loader{}).load(t.Context(), inspect(io.Discard), map[string]program.Getter{"borrowed": source})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := i.inputs.Lookup("borrowed")
	reader := input.Value.(*borrowedText).reader
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("configured reader: %v", err)
	}
	if err := i.Run(t.Context()); err != nil {
		t.Fatalf("run with borrowed reader: %v", err)
	}
	if _, err := io.ReadAll(reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after Run = %v; want closed file", err)
	}

	i, err = (loader{}).load(t.Context(), inspect(io.Discard), map[string]program.Getter{"borrowed": source})
	if err != nil {
		t.Fatal(err)
	}
	input, _ = i.inputs.Lookup("borrowed")
	reader = input.Value.(*borrowedText).reader
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
	var borrowed *borrowedText
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		borrowed = &borrowedText{trace: io.Discard}
		inputs := program.InputList{
			{Field: program.Field{Name: "reader"}, Value: borrowed},
			{Field: program.Field{Name: "rejected"}, Value: program.SetterFunc(func(string) error { return rejected })},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("failed configuration reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	i, err := (loader{}).load(t.Context(), factory, map[string]program.Getter{
		"reader":   &fileContent{path: os.DevNull, trace: io.Discard},
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
	i, err := (loader{}).load(ctx, inspect(io.Discard), map[string]program.Getter{
		"borrowed": &fileContent{path: os.DevNull, trace: io.Discard},
	})
	if err != nil {
		t.Fatal(err)
	}
	input, _ := i.inputs.Lookup("borrowed")
	reader := input.Value.(*borrowedText).reader
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
