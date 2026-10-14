package program_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/program"
)

// Eager consumers release resources during configuration; borrowers leave that
// responsibility with the loader until execution has finished.
func TestReaderLifetimes(t *testing.T) {
	eagerReader := &countedReader{Reader: strings.NewReader("eager payload")}
	source := readerGetter{open: func(context.Context) (io.ReadCloser, error) {
		return eagerReader, nil
	}}
	eager := &eagerInput{borrowedInput: borrowedInput{t: t}}
	cleanup, err := program.Wire(t.Context(), eager, source)
	if err != nil || cleanup != nil {
		t.Fatalf("eager transfer = %v, %v; want no cleanup or error", cleanup, err)
	}
	if eager.String() != "eager payload" || eager.reader != nil {
		t.Fatalf("eager value = %q, borrowed reader = %v; want eager payload without borrowing", eager.String(), eager.reader)
	}
	if eagerReader.closes != 1 {
		t.Errorf("eager closes = %d; want 1 before Wire returns", eagerReader.closes)
	}

	borrowedReader := &countedReader{Reader: strings.NewReader("borrowed payload")}
	source.open = func(context.Context) (io.ReadCloser, error) {
		return borrowedReader, nil
	}
	borrower := &borrowedInput{t: t}
	cleanup, err = program.Wire(t.Context(), borrower, source)
	if err != nil || cleanup == nil || borrower.reader == nil {
		t.Fatalf("borrowed transfer = %v, %v; want cleanup without error", cleanup, err)
	}
	if borrowedReader.closes != 0 {
		t.Fatalf("borrowed closes = %d; want 0 before the caller releases it", borrowedReader.closes)
	}
	value, err := io.ReadAll(borrower.reader)
	if err != nil || string(value) != "borrowed payload" {
		t.Fatalf("borrowed value = %q, %v; want borrowed payload", value, err)
	}
	if err := cleanup.Close(); err != nil {
		t.Fatal(err)
	}
	if borrowedReader.closes != 1 {
		t.Errorf("borrowed closes = %d; want 1 after the caller releases it", borrowedReader.closes)
	}
}

// A failed assignment leaves no resource with the caller, while preserving both
// the assignment failure and a failure to release the acquired reader.
func TestTransferFailureClosesReader(t *testing.T) {
	readErr := errors.New("source read failed")
	closeErr := errors.New("reader close failed")
	reader := &countedReader{Reader: iotest.ErrReader(readErr), closeErr: closeErr}
	source := readerGetter{open: func(context.Context) (io.ReadCloser, error) {
		return reader, nil
	}}
	eager := &eagerInput{borrowedInput: borrowedInput{t: t}}
	cleanup, err := program.Wire(t.Context(), eager, source)
	if cleanup != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) {
		t.Fatalf("read failure = %v, %v; want both causes and no cleanup", cleanup, err)
	}
	if reader.closes != 1 {
		t.Errorf("failed read closes = %d; want 1", reader.closes)
	}

	assignErr := errors.New("borrow rejected")
	reader = &countedReader{Reader: strings.NewReader("payload"), closeErr: closeErr}
	borrower := &borrowedInput{t: t, err: assignErr}
	cleanup, err = program.Wire(t.Context(), borrower, source)
	if cleanup != nil || !errors.Is(err, assignErr) || !errors.Is(err, closeErr) {
		t.Fatalf("borrow failure = %v, %v; want both causes and no cleanup", cleanup, err)
	}
	if reader.closes != 1 {
		t.Errorf("rejected borrow closes = %d; want 1", reader.closes)
	}
}

// String transfer is selected only when capabilities do not match; it must not
// conceal failure to acquire a reader for an otherwise supported transfer.
func TestReaderAcquisitionIsConditional(t *testing.T) {
	openErr := errors.New("source unavailable")
	opens, gets := 0, 0
	source := readerGetter{
		GetterFunc: func() string { gets++; return "string payload" },
		open: func(context.Context) (io.ReadCloser, error) {
			opens++
			return nil, openErr
		},
	}
	eager := &eagerInput{borrowedInput: borrowedInput{t: t}}
	cleanup, err := program.Wire(t.Context(), eager, source)
	if cleanup != nil || !errors.Is(err, openErr) || opens != 1 || gets != 0 {
		t.Fatalf("failed acquisition = %v, %v, %d opens, %d gets; want open error without string retry", cleanup, err, opens, gets)
	}

	var assigned string
	setter := program.SetterFunc(func(value string) error { assigned = value; return nil })
	cleanup, err = program.Wire(t.Context(), setter, source)
	if cleanup != nil || err != nil || assigned != "string payload" || opens != 1 || gets != 1 {
		t.Fatalf("string transfer = %v, %v, %q, %d opens, %d gets; want string payload without another open", cleanup, err, assigned, opens, gets)
	}
}

type readerGetter struct {
	program.GetterFunc
	open func(context.Context) (io.ReadCloser, error)
}

func (s readerGetter) Open(ctx context.Context) (io.ReadCloser, error) { return s.open(ctx) }

type countedReader struct {
	io.Reader
	closes   int
	closeErr error
}

func (r *countedReader) Close() error { r.closes++; return r.closeErr }

func (r *countedReader) Read(p []byte) (int, error) {
	if r.closes != 0 {
		return 0, io.ErrClosedPipe
	}
	return r.Reader.Read(p)
}

type borrowedInput struct {
	t      *testing.T
	reader io.Reader
	err    error
}

func (b *borrowedInput) Set(string) error {
	b.t.Fatal("unexpected string transfer")
	return nil
}

func (b *borrowedInput) BorrowReader(reader io.Reader) error {
	if _, ok := reader.(io.Closer); ok {
		b.t.Error("borrower received closing capability")
	}
	if b.err != nil {
		return b.err
	}
	b.reader = reader
	return nil
}

type eagerInput struct {
	borrowedInput
	bytes.Buffer
}

func (e *eagerInput) ReadFrom(reader io.Reader) (int64, error) {
	if _, ok := reader.(io.Closer); ok {
		e.t.Error("eager consumer received closing capability")
	}
	return e.Buffer.ReadFrom(reader)
}
