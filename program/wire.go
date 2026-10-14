package program

import (
	"context"
	"errors"
	"io"
)

// ReaderGetter provides a reader representation of a getter's value. Each Open
// returns an independent reader positioned at the start. On success, ownership
// of the reader passes to the caller; on failure, Open returns a nil reader and
// releases anything it acquired. Make must not call Open.
//
// The context governs acquisition. It does not make subsequent reads cancelable.
type ReaderGetter interface {
	Getter
	Open(context.Context) (io.ReadCloser, error)
}

// ReaderBorrower accepts a reader for use during the destination's invocation.
// A successful call may retain the reader until Run returns. A failed call must
// stop using it before returning. The borrower must neither close the reader
// nor use it after Run returns, including from background goroutines.
//
// An eager destination can instead implement io.ReaderFrom. It must finish using
// the reader before ReadFrom returns and must not close it.
type ReaderBorrower interface {
	Setter
	BorrowReader(io.Reader) error
}

// Wiring holds resources borrowed while configuring an invocation. Its zero
// value is ready to use. The loader owns its lifetime: call Close after Run has
// returned, or when abandoning configuration before Run. Cancellation alone is
// not proof that a runner has finished using its readers.
//
// Wiring does not synchronize access. Do not copy it after use, or call Close
// while an assignment or invocation is still using its readers.
type Wiring struct {
	borrowed []io.Closer
}

// Wire assigns source to destination, preferring a reader transfer when source
// implements ReaderGetter and destination implements io.ReaderFrom or
// ReaderBorrower. If both destination capabilities exist, ReadFrom wins. Without
// a matching pair, Wire uses destination.Set(source.Get()). A reader acquisition
// or transfer failure is returned without retrying through strings.
//
// Eager readers and failed borrows are closed before Wire returns. Successful
// borrows are retained until Close. A failed assignment leaves earlier successful
// borrows in the scope; the loader decides whether to abandon configuration.
// Transfer and cleanup errors are joined, preserving both causes. Destinations
// receive only io.Reader; closing remains the loader's responsibility.
func (w *Wiring) Wire(ctx context.Context, destination Setter, source Getter) (err error) {
	readerSource, canOpen := source.(ReaderGetter)
	eager, canRead := destination.(io.ReaderFrom)
	borrower, canBorrow := destination.(ReaderBorrower)
	if !canOpen || (!canRead && !canBorrow) {
		return destination.Set(source.Get())
	}
	reader, err := readerSource.Open(ctx)
	if err != nil {
		return err
	}
	retained := false
	defer func() {
		if !retained {
			err = errors.Join(err, reader.Close())
		}
	}()
	view := struct{ io.Reader }{reader}
	if canRead {
		_, err := eager.ReadFrom(view)
		return err
	}
	if err := borrower.BorrowReader(view); err != nil {
		return err
	}
	w.borrowed = append(w.borrowed, reader)
	retained = true
	return nil
}

// Close releases retained readers in reverse acquisition order, attempting every
// close and joining failures. It drains the scope, so repeated calls do nothing
// until Wire retains another reader. Readers must no longer be in use.
func (w *Wiring) Close() error {
	var err error
	for index := len(w.borrowed) - 1; index >= 0; index-- {
		err = errors.Join(err, w.borrowed[index].Close())
	}
	w.borrowed = nil
	return err
}
