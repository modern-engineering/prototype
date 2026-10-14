package sef_test

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// Reading and rewriting a document must retain authored relationships without
// loading catalogue packages or interpreting names as available runtime fields.
func TestJSONPreservesStructuralIntent(t *testing.T) {
	const source = `{
		"version":"0.1", "id":"cleanup", "revision":"proposal-2",
		"definition": {
			"modules":[{"path":"example.com/catalogue","version":"v1.2.0"}],
			"nodes":[
				{"name":"print","definition":{"package":"example.com/catalogue","name":"Print"}},
				{"name":"trim","definition":{"package":"example.com/catalogue","name":"Trim"}}
			],
			"bindings":[
				{"node":"print","input":"prefix","value":{"kind":"literal","text":""}},
				{"node":"print","input":"text","value":{"kind":"reference","reference":{"node":"trim","output":"text"}}}
			]
		}
	}`
	want := sef.Document{
		Version: sef.Version, ID: "cleanup", Revision: "proposal-2",
		Definition: solution.Definition{
			Modules: []solution.Module{{Path: "example.com/catalogue", Version: "v1.2.0"}},
			Nodes: []solution.Node{
				{Name: "print", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Print"}},
				{Name: "trim", Definition: solution.Symbol{Package: "example.com/catalogue", Name: "Trim"}},
			},
			Bindings: []solution.Binding{
				{Node: "print", Input: "prefix", Value: solution.Literal("")},
				{Node: "print", Input: "text", Value: solution.Reference{Node: "trim", Output: "text"}},
			},
		},
	}
	got, err := sef.DecodeJSON(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded document = %#v, want %#v", got, want)
	}
	var rewritten bytes.Buffer
	if err := sef.EncodeJSON(&rewritten, got); err != nil {
		t.Fatal(err)
	}
	got, err = sef.DecodeJSON(&rewritten)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewritten document = %#v, want %#v", got, want)
	}
}

// Schema mistakes must fail rather than silently changing an assignment's form.
func TestJSONRejectsInvalidDocuments(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         error
	}{
		{name: "AbsentVersion", source: `{}`, want: sef.ErrVersion},
		{name: "UnsupportedVersion", source: `{"version":"future"}`, want: sef.ErrVersion},
		{name: "AbsentValue", source: `{"version":"0.1","definition":{"bindings":[{}]}}`, want: sef.ErrValue},
		{name: "UnknownValue", source: `{"version":"0.1","definition":{"bindings":[{"value":{"kind":"expression"}}]}}`, want: sef.ErrValue},
		{name: "MixedValue", source: `{"version":"0.1","definition":{"bindings":[{"value":{"kind":"literal","reference":{}}}]}}`, want: sef.ErrValue},
		{name: "MissingReference", source: `{"version":"0.1","definition":{"bindings":[{"value":{"kind":"reference"}}]}}`, want: sef.ErrValue},
		{name: "TrailingDocument", source: `{"version":"0.1"}{"version":"0.1"}`, want: sef.ErrTrailingData},
		{name: "TrailingGarbage", source: `{"version":"0.1"}garbage`, want: sef.ErrTrailingData},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := sef.DecodeJSON(strings.NewReader(test.source))
			if !errors.Is(err, test.want) {
				t.Fatalf("DecodeJSON = %v, want %v", err, test.want)
			}
			if !reflect.DeepEqual(got, sef.Document{}) {
				t.Errorf("DecodeJSON returned a partial document: %#v", got)
			}
		})
	}
	for _, source := range []string{
		`{"version":"0.1","unknown":true}`,
		`{"version":"0.1","definition":{"unknown":true}}`,
		`{"version":"0.1"`,
	} {
		if _, err := sef.DecodeJSON(strings.NewReader(source)); err == nil {
			t.Errorf("DecodeJSON(%q) succeeded", source)
		}
	}
}
