package sef_test

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// JSON must reject text that its standard decoder would silently replace.
// Literal backslash escapes and valid supplementary characters remain data.
func TestJSONRejectsLossyUnicode(t *testing.T) {
	for _, path := range []string{"testdata/surrogate-high.json.sef", "testdata/surrogate-low.json.sef", "testdata/surrogate-pair.json.sef"} {
		input, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sef.DecodeJSON(bytes.NewReader(input)); !errors.Is(err, sef.ErrUnicode) {
			t.Errorf("DecodeJSON(%s) = %v, want ErrUnicode", path, err)
		}
	}
	if _, err := sef.DecodeJSON(strings.NewReader("{\"version\":\"0.1\",\"id\":\"\xff\"}")); !errors.Is(err, sef.ErrUnicode) {
		t.Errorf("DecodeJSON invalid UTF-8 = %v, want ErrUnicode", err)
	}
	input, err := os.ReadFile("testdata/unicode.json.sef")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sef.DecodeJSON(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.ID != "🚀" || artifact.Revision != `\ud800` || artifact.Definition.Bindings[0].Value != solution.Literal("�") {
		t.Fatalf("decoded Unicode = %#v", artifact)
	}
	var output bytes.Buffer
	if err := sef.EncodeJSON(&output, artifact); err != nil {
		t.Fatal(err)
	}
	got, err := sef.DecodeJSON(&output)
	if err != nil || !reflect.DeepEqual(got, artifact) {
		t.Errorf("valid Unicode round trip = %#v, %v", got, err)
	}
}

// Any string in the artifact can originate from a byte-preserving encoding.
// Rejecting before writing lets conversion leave an existing file untouched.
func TestJSONChecksEveryStringBeforeWriting(t *testing.T) {
	for _, test := range []struct {
		name string
		set  func(*solution.Artifact)
	}{
		{"ID", func(a *solution.Artifact) { a.ID = "\xff" }},
		{"Revision", func(a *solution.Artifact) { a.Revision = "\xff" }},
		{"ModulePath", func(a *solution.Artifact) { a.Definition.Modules[0].Path = "\xff" }},
		{"ModuleVersion", func(a *solution.Artifact) { a.Definition.Modules[0].Version = "\xff" }},
		{"NodeName", func(a *solution.Artifact) { a.Definition.Nodes[0].Name = "\xff" }},
		{"Package", func(a *solution.Artifact) { a.Definition.Nodes[0].Definition.Package = "\xff" }},
		{"Descriptor", func(a *solution.Artifact) { a.Definition.Nodes[0].Definition.Name = "\xff" }},
		{"BindingNode", func(a *solution.Artifact) { a.Definition.Bindings[0].Node = "\xff" }},
		{"Input", func(a *solution.Artifact) { a.Definition.Bindings[0].Input = "\xff" }},
		{"Literal", func(a *solution.Artifact) { a.Definition.Bindings[0].Value = solution.Literal("\xff") }},
		{"ReferenceNode", func(a *solution.Artifact) { a.Definition.Bindings[1].Value = solution.Reference{Node: "\xff"} }},
		{"Output", func(a *solution.Artifact) { a.Definition.Bindings[1].Value = solution.Reference{Output: "\xff"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifact := solution.Artifact{Version: sef.Version, Definition: solution.Definition{
				Modules: []solution.Module{{}}, Nodes: []solution.Node{{}},
				Bindings: []solution.Binding{{Value: solution.Literal("")}, {Value: solution.Reference{}}},
			}}
			test.set(&artifact)
			var output bytes.Buffer
			if err := sef.EncodeJSON(&output, artifact); !errors.Is(err, sef.ErrUnicode) || output.Len() != 0 {
				t.Fatalf("EncodeJSON wrote %d bytes, err = %v; want ErrUnicode before output", output.Len(), err)
			}
			if err := sef.EncodeGob(&output, artifact); err != nil {
				t.Fatal(err)
			}
			got, err := sef.DecodeGob(&output)
			if err != nil || !reflect.DeepEqual(got, artifact) {
				t.Fatalf("gob byte-string round trip = %#v, %v", got, err)
			}
		})
	}
}
