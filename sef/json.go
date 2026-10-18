package sef

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/solution"
)

// DecodeJSON reads one .json.sef document through EOF. It rejects unknown JSON
// fields, unsupported versions, invalid binding forms, and trailing documents or
// non-whitespace data. It leaves r open and returns a zero artifact on error.
// Binding values are decoded as solution.Literal or solution.Reference values,
// independent of whether their producer used pointers. Empty and nil slices are
// equivalent in the encoded definition; empty slices decode as nil.
func DecodeJSON(r io.Reader) (solution.Artifact, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	var wire wireDocument
	if err := decoder.Decode(&wire); err != nil {
		return solution.Artifact{}, fmt.Errorf("sef: decode JSON: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return solution.Artifact{}, fmt.Errorf("%w: %w", ErrTrailingData, err)
		}
		return solution.Artifact{}, ErrTrailingData
	}
	return unmarshalDocument(wire)
}

// EncodeJSON writes one .json.sef document followed by a newline. It validates
// the version and binding forms before writing any bytes. It accepts Literal and
// Reference values or non-nil pointers to them; no runtime Go values are encoded.
// It leaves w open. A write error may leave a partially written document.
func EncodeJSON(w io.Writer, document solution.Artifact) error {
	wire, err := marshalDocument(document)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(wire); err != nil {
		return fmt.Errorf("sef: encode JSON: %w", err)
	}
	return nil
}
