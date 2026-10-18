package sef

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/modern-engineering/prototype/solution"
)

// DecodeJSON reads one .json.sef document through EOF. It rejects unknown JSON
// fields, unsupported versions, invalid binding forms, and trailing documents or
// non-whitespace data. Invalid UTF-8 and unpaired Unicode surrogate escapes
// return ErrUnicode instead of being replaced. It leaves r open and returns a
// zero artifact on error.
// Binding values are decoded as solution.Literal or solution.Reference values,
// independent of whether their producer used pointers. Empty and nil slices are
// equivalent in the encoded definition; empty slices decode as nil.
func DecodeJSON(r io.Reader) (solution.Artifact, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return solution.Artifact{}, fmt.Errorf("sef: read JSON: %w", err)
	}
	if !validJSONUnicode(data) {
		return solution.Artifact{}, ErrUnicode
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
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
// All strings must contain valid UTF-8, otherwise it returns ErrUnicode. Gob and
// text encodings can carry arbitrary byte strings.
// It leaves w open. A write error may leave a partially written document.
func EncodeJSON(w io.Writer, document solution.Artifact) error {
	wire, err := marshalDocument(document)
	if err != nil {
		return err
	}
	if err := checkJSONStrings(wire); err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(wire); err != nil {
		return fmt.Errorf("sef: encode JSON: %w", err)
	}
	return nil
}

func checkJSONStrings(wire wireDocument) error {
	valid := func(values ...string) bool {
		for _, value := range values {
			if !utf8.ValidString(value) {
				return false
			}
		}
		return true
	}
	if !valid(wire.Version, wire.ID, wire.Revision) {
		return ErrUnicode
	}
	if m := wire.Module; m != nil {
		if !valid(m.Path, m.Go, m.Toolchain) {
			return ErrUnicode
		}
		for _, collection := range [][]wireModule{m.Require, m.Exclude} {
			for _, entry := range collection {
				if !valid(entry.Path, entry.Version) {
					return ErrUnicode
				}
			}
		}
		for _, entry := range m.Replace {
			if !valid(entry.Old.Path, entry.Old.Version, entry.New.Path, entry.New.Version) {
				return ErrUnicode
			}
		}
		for _, entry := range m.Sum {
			if !valid(entry.Path, entry.Version, entry.Hash) {
				return ErrUnicode
			}
		}
	}
	for i, module := range wire.Definition.Modules {
		if !valid(module.Path, module.Version) {
			return fmt.Errorf("%w: module %d", ErrUnicode, i)
		}
	}
	for i, node := range wire.Definition.Nodes {
		if !valid(node.Name, node.Definition.Package, node.Definition.Name) {
			return fmt.Errorf("%w: node %d", ErrUnicode, i)
		}
	}
	for i, binding := range wire.Definition.Bindings {
		if !valid(binding.Node, binding.Input, binding.Value.Text) {
			return fmt.Errorf("%w: binding %d", ErrUnicode, i)
		}
		if reference := binding.Value.Reference; reference != nil && !valid(reference.Node, reference.Output) {
			return fmt.Errorf("%w: binding %d reference", ErrUnicode, i)
		}
	}
	return nil
}

// encoding/json replaces invalid Unicode with U+FFFD. Check just that lossy case;
// the decoder remains responsible for the JSON grammar and structural schema.
func validJSONUnicode(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		if i+1 >= len(data) {
			return true
		}
		if data[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(data) {
			return true
		}
		code, err := strconv.ParseUint(string(data[i+2:i+6]), 16, 16)
		if err != nil {
			return true
		}
		switch {
		case code >= 0xdc00 && code <= 0xdfff:
			return false
		case code >= 0xd800 && code <= 0xdbff:
			if i+12 > len(data) || data[i+6] != '\\' || data[i+7] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+8:i+12]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 11
		default:
			i += 5
		}
	}
	return true
}
