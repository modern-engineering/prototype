package sef_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

var encodings = []struct {
	name   string
	encode func(io.Writer, solution.Artifact) error
	decode func(io.Reader) (solution.Artifact, error)
}{
	{name: "JSON", encode: sef.EncodeJSON, decode: sef.DecodeJSON},
	{name: "Gob", encode: sef.EncodeGob, decode: sef.DecodeGob},
}

// Neither encoding may erase explicit empty assignments or change them into
// absent bindings. Crossing encodings also removes Go pointer representation.
func TestEncodingsPreserveAssignments(t *testing.T) {
	literal := solution.Literal("")
	reference := solution.Reference{}
	document := solution.Artifact{
		Version: sef.Version, ID: "cleanup", Revision: "proposal-2",
		Definition: solution.Definition{
			Modules: []solution.Module{{Path: "example.com/catalogue", Version: "v1.2.0"}},
			Nodes: []solution.Node{
				{Name: "print", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Print"}},
				{Name: "trim", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Trim"}},
			},
			Bindings: []solution.Binding{
				{Node: "print", Input: "prefix", Value: &literal},
				{Node: "print", Input: "text", Value: solution.Reference{Node: "trim", Output: "text"}},
				{Node: "trim", Input: "text", Value: solution.Literal("  hello  ")},
				{Node: "trim", Input: "unresolved", Value: &reference},
			},
		},
	}
	want := document
	want.Definition.Bindings = []solution.Binding{
		{Node: "print", Input: "prefix", Value: solution.Literal("")},
		{Node: "print", Input: "text", Value: solution.Reference{Node: "trim", Output: "text"}},
		{Node: "trim", Input: "text", Value: solution.Literal("  hello  ")},
		{Node: "trim", Input: "unresolved", Value: solution.Reference{}},
	}
	for _, first := range encodings {
		for _, second := range encodings {
			t.Run(first.name+"To"+second.name, func(t *testing.T) {
				var input ownedBuffer
				if err := first.encode(&input, document); err != nil {
					t.Fatal(err)
				}
				decoded, err := first.decode(&input)
				if err != nil {
					t.Fatal(err)
				}
				var output ownedBuffer
				if err := second.encode(&output, decoded); err != nil {
					t.Fatal(err)
				}
				got, err := second.decode(&output)
				if err != nil {
					t.Fatal(err)
				}
				if input.closed || output.closed {
					t.Fatal("codec closed a caller-owned stream")
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("document = %#v, want %#v", got, want)
				}
			})
		}
	}
}

type ownedBuffer struct {
	bytes.Buffer
	closed bool
}

func (b *ownedBuffer) Close() error {
	b.closed = true
	return nil
}

// Invalid in-memory values must not leave a file that looks partly valid.
func TestEncodingsValidateBeforeWriting(t *testing.T) {
	for _, codec := range encodings {
		t.Run(codec.name, func(t *testing.T) {
			var nilLiteral *solution.Literal
			var nilReference *solution.Reference
			for _, value := range []solution.Value{nil, nilLiteral, nilReference} {
				var output bytes.Buffer
				document := solution.Artifact{Version: sef.Version, Definition: solution.Definition{
					Bindings: []solution.Binding{{Node: "trim", Input: "text", Value: value}},
				}}
				if err := codec.encode(&output, document); !errors.Is(err, sef.ErrValue) {
					t.Errorf("Encode with %T = %v, want ErrValue", value, err)
				}
				if output.Len() != 0 {
					t.Errorf("Encode with %T wrote %d bytes", value, output.Len())
				}
			}
			var output bytes.Buffer
			if err := codec.encode(&output, solution.Artifact{}); !errors.Is(err, sef.ErrVersion) {
				t.Errorf("Encode without version = %v, want ErrVersion", err)
			}
			if output.Len() != 0 {
				t.Errorf("Encode without version wrote %d bytes", output.Len())
			}
		})
	}
}

// A caller needs the original I/O failure to distinguish unavailable storage from
// malformed content, including failures discovered while checking the file end.
func TestEncodingsPreserveStreamErrors(t *testing.T) {
	for _, codec := range encodings {
		t.Run(codec.name, func(t *testing.T) {
			unavailable := errors.New("storage unavailable")
			var valid bytes.Buffer
			if err := codec.encode(&valid, solution.Artifact{Version: sef.Version}); err != nil {
				t.Fatal(err)
			}
			for _, input := range []io.Reader{
				iotest.ErrReader(unavailable),
				io.MultiReader(&valid, iotest.ErrReader(unavailable)),
			} {
				if _, err := codec.decode(input); !errors.Is(err, unavailable) {
					t.Errorf("Decode = %v, want original read error", err)
				}
			}
			if err := codec.encode(errorWriter{unavailable}, solution.Artifact{Version: sef.Version}); !errors.Is(err, unavailable) {
				t.Errorf("Encode = %v, want original write error", err)
			}
		})
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }
