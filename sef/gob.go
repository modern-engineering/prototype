package sef

import (
	"encoding/gob"
	"fmt"
	"io"
)

// DecodeGob reads one .gob.sef document through EOF. It rejects unsupported
// versions, invalid binding forms, and trailing documents or data. It leaves r
// open and returns a zero Document on error. Binding values are decoded as
// solution.Literal or solution.Reference values; empty slices decode as nil.
// It needs no gob registrations of catalogue types or structural value forms.
func DecodeGob(r io.Reader) (Document, error) {
	decoder := gob.NewDecoder(r)
	var wire wireDocument
	if err := decoder.Decode(&wire); err != nil {
		return Document{}, fmt.Errorf("sef: decode gob: %w", err)
	}
	var extra wireDocument
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return Document{}, fmt.Errorf("%w: %w", ErrTrailingData, err)
		}
		return Document{}, ErrTrailingData
	}
	return unmarshalDocument(wire)
}

// EncodeGob writes one .gob.sef document in a fresh gob stream. It validates the
// version and binding forms before writing any bytes, accepting the same values
// as EncodeJSON. It leaves w open. A write error may leave a partially written
// document. It needs no gob registrations and encodes no runtime Go values.
func EncodeGob(w io.Writer, document Document) error {
	wire, err := marshalDocument(document)
	if err != nil {
		return err
	}
	if err := gob.NewEncoder(w).Encode(wire); err != nil {
		return fmt.Errorf("sef: encode gob: %w", err)
	}
	return nil
}
