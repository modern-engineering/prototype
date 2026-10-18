package sef_test

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// A gob file contains one document, not a prefix of a stream that happens to
// begin with one. Both another gob item and malformed suffixes must fail.
func TestGobRejectsTrailingDataAndTruncation(t *testing.T) {
	var encoded bytes.Buffer
	if err := sef.EncodeGob(&encoded, solution.Artifact{Version: sef.Version}); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range [][]byte{encoded.Bytes(), []byte("garbage")} {
		input := io.MultiReader(bytes.NewReader(encoded.Bytes()), bytes.NewReader(suffix))
		if got, err := sef.DecodeGob(input); !errors.Is(err, sef.ErrTrailingData) {
			t.Errorf("DecodeGob with suffix = %#v, %v; want ErrTrailingData", got, err)
		}
	}
	if _, err := sef.DecodeGob(bytes.NewReader(encoded.Bytes()[:encoded.Len()-1])); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("DecodeGob truncated = %v, want io.ErrUnexpectedEOF", err)
	}
}

// Gob transmits exported record fields, so files can be produced without this
// package's private Go types. They still have to satisfy the SEF schema.
func TestGobRejectsUnsupportedVersionAndMissingValue(t *testing.T) {
	var unsupported bytes.Buffer
	if err := gob.NewEncoder(&unsupported).Encode(struct{ Version string }{"future"}); err != nil {
		t.Fatal(err)
	}
	if _, err := sef.DecodeGob(&unsupported); !errors.Is(err, sef.ErrVersion) {
		t.Errorf("DecodeGob unsupported version = %v, want ErrVersion", err)
	}

	record := struct {
		Version    string
		Definition struct {
			Bindings []struct{ Node, Input string }
		}
	}{Version: sef.Version}
	record.Definition.Bindings = []struct{ Node, Input string }{{Node: "trim", Input: "text"}}
	var invalid bytes.Buffer
	if err := gob.NewEncoder(&invalid).Encode(record); err != nil {
		t.Fatal(err)
	}
	if _, err := sef.DecodeGob(&invalid); !errors.Is(err, sef.ErrValue) {
		t.Errorf("DecodeGob absent binding value = %v, want ErrValue", err)
	}
	if _, err := sef.DecodeGob(strings.NewReader("")); !errors.Is(err, io.EOF) {
		t.Errorf("DecodeGob empty input = %v, want io.EOF", err)
	}
}
