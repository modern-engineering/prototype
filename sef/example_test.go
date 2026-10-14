package sef_test

import (
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/symbol"
)

// A document carries a named definition independently of its encoding. The
// consumer supplies runtime values and links the definition when needed.
func ExampleDocument() {
	document := sef.Document{
		Version:  "example-schema",
		ID:       "text-cleanup",
		Revision: "proposal-2",
		Definition: solution.Definition{Nodes: []solution.Node{
			{Name: "trim", Definition: solution.Symbol{Package: "strings", Name: "TrimSpace"}},
		}},
	}

	var values symbol.Table
	if err := values.Define("strings", "TrimSpace", strings.TrimSpace); err != nil {
		fmt.Println(err)
		return
	}
	resolved, err := link.Resolve(document.Definition, &values)
	if err != nil {
		fmt.Println(err)
		return
	}
	trim := resolved.Nodes[0].Definition.(func(string) string)
	fmt.Printf("%s at %s (%s)\n", document.ID, document.Revision, document.Version)
	fmt.Println(trim("  hello  "))

	// Output:
	// text-cleanup at proposal-2 (example-schema)
	// hello
}
