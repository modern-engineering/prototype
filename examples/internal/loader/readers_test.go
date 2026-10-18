package loader_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// File inputs can consume immediately or borrow through execution, without
// giving either destination ownership of the opened file.
func TestFileTransferLifetimes(t *testing.T) {
	const path = "payload.txt"
	var eager *eagerValue
	var borrowed *borrowedValue
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		eager = &eagerValue{borrowedValue: borrowedValue{t: t}}
		borrowed = &borrowedValue{t: t}
		var plain string
		inputs := program.InputList{
			{Field: program.Field{Name: "eager"}, Value: eager},
			{Field: program.Field{Name: "borrowed"}, Value: borrowed},
			{Field: program.Field{Name: "plain"}, Value: program.SetterFunc(func(text string) error { plain = text; return nil })},
		}
		run := program.RunnerFunc(func(context.Context) error {
			body, err := io.ReadAll(borrowed.reader)
			if err != nil {
				return err
			}
			if eager.text != "file contents\n" || plain != "file contents\n" || string(body) != "file contents\n" {
				t.Errorf("input values = %q, %q, %q; want file contents", eager.text, plain, body)
			}
			return nil
		})
		return run, inputs, program.OutputList(nil)
	})
	i := (loader.Loader{FS: os.DirFS(".")}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	if err := i.ReadFile(t.Context(), "eager", path); err != nil {
		t.Fatal(err)
	}
	if eager.reader != nil {
		t.Fatal("eager input was assigned through BorrowReader instead of ReadFrom")
	}
	if _, err := io.ReadAll(eager.seen); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("eager reader after assignment = %v; want closed file", err)
	}
	if err := i.ReadFile(t.Context(), "borrowed", path); err != nil {
		t.Fatal(err)
	}
	if err := i.ReadFile(t.Context(), "plain", path); err != nil {
		t.Fatal(err)
	}
	if err := i.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(borrowed.reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("borrowed reader after Run = %v; want closed file", err)
	}
	if err := i.Close(); err != nil {
		t.Fatalf("repeated close: %v", err)
	}
}

// A failed assignment releases its file, while earlier borrows stay usable
// until the caller abandons the invocation.
func TestAbandonedConfigurationReleasesBorrowedFiles(t *testing.T) {
	const path = "payload.txt"
	rejected := errors.New("assignment rejected")
	var accepted, refused *borrowedValue
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		accepted = &borrowedValue{t: t}
		refused = &borrowedValue{t: t, err: rejected}
		inputs := program.InputList{
			{Field: program.Field{Name: "accepted"}, Value: accepted},
			{Field: program.Field{Name: "refused"}, Value: refused},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("abandoned configuration reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	i := (loader.Loader{}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	if err := i.ReadFile(t.Context(), "accepted", path); err != nil {
		t.Fatal(err)
	}
	if err := i.ReadFile(t.Context(), "refused", path); !errors.Is(err, rejected) {
		t.Fatalf("file assignment error = %v; want assignment rejection", err)
	}
	if _, err := io.ReadAll(refused.seen); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("refused reader after failed assignment = %v; want closed file", err)
	}
	value, err := io.ReadAll(accepted.reader)
	if err != nil || string(value) != "file contents\n" {
		t.Fatalf("earlier borrow = %q, %v; want file contents still available", value, err)
	}
	if err := i.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(accepted.reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("borrowed reader after abandonment = %v; want closed file", err)
	}
}

// Cancellation prevents acquisition, but does not revoke an existing borrow
// before the runner has finished using it.
func TestCancellationRespectsFileLifetime(t *testing.T) {
	const path = "payload.txt"
	var borrowed *borrowedValue
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		borrowed = &borrowedValue{t: t}
		run := program.RunnerFunc(func(ctx context.Context) error {
			body, err := io.ReadAll(borrowed.reader)
			if err != nil || string(body) != "file contents\n" {
				t.Errorf("reader during canceled Run = %q, %v; want file contents", body, err)
			}
			return ctx.Err()
		})
		return run, program.InputList{{Field: program.Field{Name: "body"}, Value: borrowed}}, program.OutputList(nil)
	})
	i := (loader.Loader{}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := i.ReadFile(ctx, "body", filepath.Join(t.TempDir(), "missing")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled assignment = %v; want context cancellation before opening", err)
	}
	if err := i.ReadFile(t.Context(), "body", path); err != nil {
		t.Fatal(err)
	}
	if err := i.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v; want context cancellation", err)
	}
	if _, err := io.ReadAll(borrowed.reader); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("reader after canceled Run = %v; want closed file", err)
	}
}

type borrowedValue struct {
	t      *testing.T
	reader io.Reader
	seen   io.Reader
	err    error
}

func (v *borrowedValue) Set(text string) error {
	v.reader = strings.NewReader(text)
	return nil
}

func (v *borrowedValue) BorrowReader(reader io.Reader) error {
	if _, ok := reader.(io.Closer); ok {
		v.t.Error("borrower received closing capability")
	}
	v.seen = reader
	if v.err != nil {
		return v.err
	}
	v.reader = reader
	return nil
}

type eagerValue struct {
	borrowedValue
	text string
}

func (v *eagerValue) ReadFrom(reader io.Reader) (int64, error) {
	if _, ok := reader.(io.Closer); ok {
		v.t.Error("eager consumer received closing capability")
	}
	v.seen = reader
	data, err := io.ReadAll(reader)
	v.text = string(data)
	return int64(len(data)), err
}
