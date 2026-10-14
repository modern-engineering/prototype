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

// Wire assigns source to destination, preferring a reader transfer when source
// implements ReaderGetter and destination implements io.ReaderFrom or
// ReaderBorrower. If both destination capabilities exist, ReadFrom wins. Without
// a matching pair, Wire uses destination.Set(source.Get()). A reader acquisition
// or transfer failure is returned without retrying through strings.
//
// Eager readers are closed before Wire returns. A successful borrow returns a
// closer that the loader must retain: close it after the destination's Run has
// returned, or if configuration is abandoned before Run. Cancellation alone is
// not proof that a runner has finished using its reader. All other paths return
// a nil closer. Transfer and cleanup errors are joined, preserving both causes.
// Destinations receive only io.Reader; closing remains the loader's responsibility.
func Wire(ctx context.Context, destination Setter, source Getter) (borrowed io.Closer, err error) {
	readerSource, canOpen := source.(ReaderGetter)
	eager, canRead := destination.(io.ReaderFrom)
	borrower, canBorrow := destination.(ReaderBorrower)
	if !canOpen || (!canRead && !canBorrow) {
		return nil, destination.Set(source.Get())
	}
	reader, err := readerSource.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if borrowed == nil {
			err = errors.Join(err, reader.Close())
		}
	}()
	view := struct{ io.Reader }{reader}
	if canRead {
		_, err := eager.ReadFrom(view)
		return nil, err
	}
	if err := borrower.BorrowReader(view); err != nil {
		return nil, err
	}
	return reader, nil
}
