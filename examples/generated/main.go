// Command generated shows how generated imports and registrations make your
// catalogue available for inspection without constructing or running programs.
package main

//go:generate go run ./generate

import (
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
)

func main() {
	if err := inspect(); err != nil {
		log.Fatal(err)
	}
}

func inspect() error {
	var values program.Registry
	// Generated registration uses ordinary Go imports, so the compiler checks
	// that your catalogue exports each descriptor named by the solution.
	if err := registerDefinitions(&values); err != nil {
		return err
	}
	resolved, err := link.Resolve(design.Definition, &values)
	if err != nil {
		return err
	}
	for node := range resolved.Nodes() {
		fmt.Printf("%s: %T\n", node.Name(), node.Descriptor())
	}
	for binding := range resolved.Bindings() {
		switch value := binding.Value().(type) {
		case link.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node().Name(), binding.Input(), value)
		case link.Reference:
			fmt.Printf("%s.%s <- %s.%s\n", binding.Node().Name(), binding.Input(), value.Node.Name(), value.Output)
		}
	}
	return nil
}
