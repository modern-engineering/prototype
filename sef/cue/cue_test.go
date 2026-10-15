package cue_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	cueerrors "cuelang.org/go/cue/errors"

	"github.com/modern-engineering/prototype/sef"
	sefcue "github.com/modern-engineering/prototype/sef/cue"
	"github.com/modern-engineering/prototype/solution"
)

// Source expressions construct data; a node/output reference remains an explicit
// graph edge even when its target cannot be resolved in this process.
func TestParseEvaluatesArtifact(t *testing.T) {
	input, err := os.ReadFile("testdata/service.input")
	if err != nil {
		t.Fatal(err)
	}
	file, err := sefcue.Parse("service.cue.sef", bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	got := file.Artifact()
	want := solution.Artifact{
		Version: sef.Version, ID: "service", Revision: "draft",
		Definition: solution.Definition{
			Modules: []solution.Module{{Path: "example.com/catalogue", Version: "v1.2.0"}},
			Nodes: []solution.Node{
				{Name: "trim", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Trim"}},
				{Name: "print", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Print"}},
			},
			Bindings: []solution.Binding{
				{Node: "trim", Input: "text", Value: solution.Literal("hello world")},
				{Node: "print", Input: "text", Value: solution.Reference{Node: "trim", Output: "text"}},
				{Node: "print", Input: "prefix", Value: solution.Literal("")},
				{Node: "trim", Input: "code", Value: solution.Literal("package main\nfunc main() { println(\"hello\") }")},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Artifact = %#v, want %#v", got, want)
	}
	got.Definition.Modules[0].Path = "changed"
	got.Definition.Nodes[0].Name = "changed"
	got.Definition.Bindings[0].Value = solution.Literal("changed")
	if !reflect.DeepEqual(file.Artifact(), want) {
		t.Fatal("editing a returned artifact changed the file")
	}
}

func TestParseDefaults(t *testing.T) {
	file, err := sefcue.Parse("empty.cue.sef", strings.NewReader(`version: "0.1"`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := file.Artifact(), (solution.Artifact{Version: sef.Version}); !reflect.DeepEqual(got, want) {
		t.Errorf("Artifact = %#v, want %#v", got, want)
	}
}

func TestParseRejectsInvalidArtifact(t *testing.T) {
	for _, test := range []struct {
		name, source string
		version      bool
	}{
		{name: "MissingVersion", source: `id: "sample"`, version: true},
		{name: "UnsupportedVersion", source: `version: "future"`, version: true},
		{name: "NumericVersion", source: `version: 0.1`},
		{name: "BytesVersion", source: `version: '0.1'`},
		{name: "IncompleteVersion", source: `version: string`},
		{name: "UnknownField", source: `version: "0.1", typo: "lost"`},
		{name: "UnknownNestedField", source: `version: "0.1", definition: {nodes: [{name: "a", definition: {package: "p", name: "N"}, typo: 1}]}`},
		{name: "MissingLiteral", source: `version: "0.1", definition: {bindings: [{node: "a", input: "x", value: {kind: "literal"}}]}`},
		{name: "IncompleteLiteral", source: `version: "0.1", definition: {bindings: [{node: "a", input: "x", value: {kind: "literal", text: string}}]}`},
		{name: "MixedValue", source: `version: "0.1", definition: {bindings: [{node: "a", input: "x", value: {kind: "literal", text: "", reference: {node: "a", output: "x"}}}]}`},
		{name: "Syntax", source: "version: \"0.1\"\ndefinition: {"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := sefcue.Parse("invalid.cue.sef", strings.NewReader(test.source))
			if err == nil || file != nil {
				t.Fatalf("Parse = %v, %v; want nil file and error", file, err)
			}
			if test.version {
				if !errors.Is(err, sef.ErrVersion) {
					t.Errorf("Parse = %v, want ErrVersion", err)
				}
				return
			}
			var detail cueerrors.Error
			if !errors.As(err, &detail) {
				t.Fatalf("Parse = %v, want inspectable CUE diagnostic", err)
			}
			var located bool
			for _, diagnostic := range cueerrors.Errors(err) {
				for _, position := range cueerrors.Positions(diagnostic) {
					if position.Line() > 0 && position.Column() > 0 && (position.Filename() == "invalid.cue.sef" || test.name == "MissingLiteral" && position.Filename() == "sef/cue/schema.cue") {
						located = true
					}
				}
			}
			if !located {
				t.Errorf("diagnostic has no input source position: %v", err)
			}
		})
	}
}

// The artifact's existing string representation includes arbitrary bytes. CUE
// byte literals must preserve them through all structural fields and payloads.
func TestEncodeRoundTripBytes(t *testing.T) {
	for _, value := range []string{"", "hello\n世界\x00", "\xff\xfe\x00\\\"'", "\xed\xa0\x80"} {
		artifact := solution.Artifact{
			Version: sef.Version, ID: value, Revision: value,
			Definition: solution.Definition{
				Modules: []solution.Module{{Path: value, Version: value}},
				Nodes:   []solution.Node{{Name: value, Definition: solution.Symbol{Package: value, Name: value}}},
				Bindings: []solution.Binding{
					{Node: value, Input: value, Value: solution.Literal(value)},
					{Node: value, Input: value, Value: solution.Reference{Node: value, Output: value}},
				},
			},
		}
		var output bytes.Buffer
		if err := sefcue.Encode(&output, artifact); err != nil {
			t.Fatal(err)
		}
		file, err := sefcue.Parse("encoded.cue.sef", bytes.NewReader(output.Bytes()))
		if err != nil {
			t.Fatalf("Parse encoded %q: %v\n%s", value, err, output.Bytes())
		}
		if !reflect.DeepEqual(file.Artifact(), artifact) {
			t.Errorf("round trip %q = %#v, want %#v", value, file.Artifact(), artifact)
		}
		var again bytes.Buffer
		if err := sefcue.Encode(&again, file.Artifact()); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again.Bytes(), output.Bytes()) {
			t.Fatalf("encoding is not a fixed point for %q", value)
		}
	}
}

func TestEncodeValidatesBeforeWriting(t *testing.T) {
	var nilLiteral *solution.Literal
	var nilReference *solution.Reference
	for _, value := range []solution.Value{nil, nilLiteral, nilReference} {
		artifact := solution.Artifact{Version: sef.Version, Definition: solution.Definition{
			Bindings: []solution.Binding{{Value: value}},
		}}
		var output bytes.Buffer
		if err := sefcue.Encode(&output, artifact); !errors.Is(err, sef.ErrValue) || output.Len() != 0 {
			t.Errorf("Encode %T wrote %d bytes, err = %v; want ErrValue before writing", value, output.Len(), err)
		}
	}
	var output bytes.Buffer
	if err := sefcue.Encode(&output, solution.Artifact{}); !errors.Is(err, sef.ErrVersion) || output.Len() != 0 {
		t.Errorf("Encode unsupported version wrote %d bytes, err = %v", output.Len(), err)
	}
}

func TestStreamOwnershipAndErrors(t *testing.T) {
	var stream ownedBuffer
	if err := sefcue.Encode(&stream, solution.Artifact{Version: sef.Version}); err != nil {
		t.Fatal(err)
	}
	file, err := sefcue.Parse("owned.cue.sef", &stream)
	if err != nil {
		t.Fatal(err)
	}
	if err := sefcue.Format(&stream, file); err != nil {
		t.Fatal(err)
	}
	if stream.closed {
		t.Fatal("CUE operation closed a caller-owned stream")
	}
	failure := errors.New("storage unavailable")
	for _, input := range []io.Reader{
		iotest.ErrReader(failure),
		io.MultiReader(strings.NewReader(`version: "0.1"`), iotest.ErrReader(failure)),
	} {
		if _, err := sefcue.Parse("failed.cue.sef", input); !errors.Is(err, failure) {
			t.Errorf("Parse = %v, want original reader failure", err)
		}
	}
	if err := sefcue.Encode(errorWriter{failure}, file.Artifact()); !errors.Is(err, failure) {
		t.Errorf("Encode = %v, want original writer failure", err)
	}
	if err := sefcue.Format(errorWriter{failure}, file); !errors.Is(err, failure) {
		t.Errorf("Format = %v, want original writer failure", err)
	}
}

type ownedBuffer struct {
	bytes.Buffer
	closed bool
}

func (b *ownedBuffer) Close() error { b.closed = true; return nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }
