package bind

import (
	"bytes"
	"io"
)

// BytesValue is a byte slice with Get and Set methods. It also implements
// io.ReaderFrom and io.WriterTo. Its zero value is an empty slice.
// Use Bytes to bind an existing byte-slice variable.
type BytesValue []byte

// Bytes binds target without changing it. The pointer must be nonnil;
// the slice stored in it may be nil.
func Bytes(target *[]byte) *BytesValue {
	return (*BytesValue)(target)
}

// Get returns a string copy of the current bytes.
func (v *BytesValue) Get() string { return string(*v) }

// Set replaces the slice with a byte copy of text.
func (v *BytesValue) Set(text string) error {
	*v = []byte(text)
	return nil
}

// ReadFrom consumes r into a new slice and returns the number of bytes read.
// On a read error the slice is unchanged. It neither retains nor closes r.
func (v *BytesValue) ReadFrom(r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return int64(len(data)), err
	}
	*v = data
	return int64(len(data)), nil
}

// WriteTo writes the current bytes to w without changing them.
// It returns the number of bytes written and any write error, including
// io.ErrShortWrite when w accepts fewer bytes without reporting an error.
// It neither retains nor closes w.
func (v *BytesValue) WriteTo(w io.Writer) (int64, error) {
	return bytes.NewReader(*v).WriteTo(w)
}
