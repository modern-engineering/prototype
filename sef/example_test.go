package sef_test

import (
	"bytes"
	"fmt"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// A SEF file carries structural intent without importing its catalogue packages.
// An empty literal remains an assignment when the document is written and read.
func ExampleDocument() {
	document := sef.Document{
		Version: sef.Version, ID: "text-cleanup", Revision: "proposal-2",
		Definition: solution.Definition{
			Nodes: []solution.Node{{Name: "trim", Definition: solution.Symbol{
				Package: "example.com/catalogue", Name: "Trim",
			}}},
			Bindings: []solution.Binding{{Node: "trim", Input: "prefix", Value: solution.Literal("")}},
		},
	}
	var encoded bytes.Buffer
	if err := sef.EncodeJSON(&encoded, document); err != nil {
		fmt.Println(err)
		return
	}
	restored, err := sef.DecodeJSON(&encoded)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%s at %s (%s)\n", restored.ID, restored.Revision, restored.Version)
	binding := restored.Definition.Bindings[0]
	fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, binding.Value)

	// Output:
	// text-cleanup at proposal-2 (0.1)
	// trim.prefix = ""
}
