package program_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/program"
)

// Eager consumers release resources during configuration; borrowers leave that
// responsibility with the loader until execution has finished.
func TestReaderLifetimes(t *testing.T) {
	var wiring program.Wiring
	eagerReader := &countedReader{Reader: strings.NewReader("eager payload")}
	source := readerGetter{open: func(context.Context) (io.ReadCloser, error) {
		return eagerReader, nil
	}}
	eager := &eagerInput{borrowedInput: borrowedInput{t: t}}
	if err := wiring.Wire(t.Context(), eager, source); err != nil {
		t.Fatal(err)
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
	if err := wiring.Wire(t.Context(), borrower, source); err != nil {
		t.Fatal(err)
	}
	if borrower.reader == nil {
		t.Fatal("borrower received no reader")
	}
	if borrowedReader.closes != 0 {
		t.Fatalf("borrowed closes = %d; want 0 before the caller releases it", borrowedReader.closes)
	}
	value, err := io.ReadAll(borrower.reader)
	if err != nil || string(value) != "borrowed payload" {
		t.Fatalf("borrowed value = %q, %v; want borrowed payload", value, err)
	}
	if err := wiring.Close(); err != nil {
		t.Fatal(err)
	}
	if borrowedReader.closes != 1 {
		t.Errorf("borrowed closes = %d; want 1 after the caller releases it", borrowedReader.closes)
	}
	if eagerReader.closes != 1 {
		t.Errorf("eager closes = %d; want 1 after scope cleanup", eagerReader.closes)
	}
}

// Readers acquired for failed assignments are released immediately, preserving
// both the assignment failure and a failure to release the acquired reader.
func TestTransferFailureClosesReader(t *testing.T) {
	var wiring program.Wiring
	readErr := errors.New("source read failed")
	closeErr := errors.New("reader close failed")
	reader := &countedReader{Reader: iotest.ErrReader(readErr), closeErr: closeErr}
	source := readerGetter{open: func(context.Context) (io.ReadCloser, error) {
		return reader, nil
	}}
	eager := &eagerInput{borrowedInput: borrowedInput{t: t}}
	err := wiring.Wire(t.Context(), eager, source)
	if !errors.Is(err, readErr) || !errors.Is(err, closeErr) {
		t.Fatalf("read failure = %v; want both causes", err)
	}
	eagerReader := reader
	if reader.closes != 1 {
		t.Errorf("failed read closes = %d; want 1", reader.closes)
	}

	assignErr := errors.New("borrow rejected")
	reader = &countedReader{Reader: strings.NewReader("payload"), closeErr: closeErr}
	borrower := &borrowedInput{t: t, err: assignErr}
	err = wiring.Wire(t.Context(), borrower, source)
	if !errors.Is(err, assignErr) || !errors.Is(err, closeErr) {
		t.Fatalf("borrow failure = %v; want both causes", err)
	}
	if reader.closes != 1 {
		t.Errorf("rejected borrow closes = %d; want 1", reader.closes)
	}
	if err := wiring.Close(); err != nil || reader.closes != 1 || eagerReader.closes != 1 {
		t.Errorf("scope cleanup = %v, %d borrow closes, %d eager closes; want no additional cleanup", err, reader.closes, eagerReader.closes)
	}
}

// String transfer is selected only when capabilities do not match; it must not
// conceal failure to acquire a reader for an otherwise supported transfer.
func TestReaderAcquisitionIsConditional(t *testing.T) {
	var wiring program.Wiring
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
	err := wiring.Wire(t.Context(), eager, source)
	if !errors.Is(err, openErr) || opens != 1 || gets != 0 {
		t.Fatalf("failed acquisition = %v, %d opens, %d gets; want open error without string retry", err, opens, gets)
	}

	var assigned string
	setter := program.SetterFunc(func(value string) error { assigned = value; return nil })
	err = wiring.Wire(t.Context(), setter, source)
	if err != nil || assigned != "string payload" || opens != 1 || gets != 1 {
		t.Fatalf("string transfer = %v, %q, %d opens, %d gets; want string payload without another open", err, assigned, opens, gets)
	}
	if err := wiring.Close(); err != nil {
		t.Fatal(err)
	}
}

// A loader can abandon partially configured state with one cleanup, even when
// several releases fail, and then reuse the emptied scope.
func TestBorrowedReadersReleasedTogether(t *testing.T) {
	var wiring program.Wiring
	var closed []string
	firstErr, secondErr := errors.New("first close failed"), errors.New("second close failed")
	source := readerGetter{open: func(context.Context) (io.ReadCloser, error) {
		return closingReader{Reader: strings.NewReader("first"), close: func() error {
			closed = append(closed, "first")
			return firstErr
		}}, nil
	}}
	if err := wiring.Wire(t.Context(), &borrowedInput{t: t}, source); err != nil {
		t.Fatal(err)
	}
	source.open = func(context.Context) (io.ReadCloser, error) {
		return closingReader{Reader: strings.NewReader("second"), close: func() error {
			closed = append(closed, "second")
			return secondErr
		}}, nil
	}
	if err := wiring.Wire(t.Context(), &borrowedInput{t: t}, source); err != nil {
		t.Fatal(err)
	}
	assignErr := errors.New("assignment rejected")
	err := wiring.Wire(t.Context(), program.SetterFunc(func(string) error { return assignErr }), program.GetterFunc(func() string { return "payload" }))
	if !errors.Is(err, assignErr) || len(closed) != 0 {
		t.Fatalf("failed assignment = %v, closed = %v; want assignment error with prior borrows retained", err, closed)
	}
	err = wiring.Close()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Errorf("scope cleanup = %v; want both close causes", err)
	}
	if !slices.Equal(closed, []string{"second", "first"}) {
		t.Errorf("close order = %v; want [second first]", closed)
	}
	if err := wiring.Close(); err != nil || len(closed) != 2 {
		t.Fatalf("repeated cleanup = %v, closed = %v; want no further closes", err, closed)
	}

	source.open = func(context.Context) (io.ReadCloser, error) {
		return closingReader{Reader: strings.NewReader("reused"), close: func() error {
			closed = append(closed, "reused")
			return nil
		}}, nil
	}
	if err := wiring.Wire(t.Context(), &borrowedInput{t: t}, source); err != nil {
		t.Fatal(err)
	}
	if err := wiring.Close(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(closed, []string{"second", "first", "reused"}) {
		t.Errorf("close order after reuse = %v; want [second first reused]", closed)
	}
}

type closingReader struct {
	io.Reader
	close func() error
}

func (r closingReader) Close() error { return r.close() }

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
