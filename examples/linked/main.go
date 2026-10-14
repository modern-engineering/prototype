// Command linked resolves structural intent using manually registered Go
// values. Linking resolves occurrences and symbols without invoking programs.
package main

import (
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/symbol"
)

func main() {
	if err := inspect(); err != nil {
		log.Fatal(err)
	}
}

func inspect() error {
	var values symbol.Table
	for _, definition := range []struct {
		name  string
		value any
	}{
		{"Database", catalogue.Database},
		{"Backend", catalogue.Backend},
		{"Europe", catalogue.Europe},
	} {
		if err := values.Define(design.CataloguePath, definition.name, definition.value); err != nil {
			return err
		}
	}

	resolved, err := link.Resolve(design.Definition, &values)
	if err != nil {
		return err
	}
	for _, node := range resolved.Nodes {
		_, canMake := node.Definition.(program.Descriptor)
		fmt.Printf("%s: %T, can instantiate: %t\n", node.Name, node.Definition, canMake)
	}
	for _, binding := range resolved.Bindings {
		switch value := binding.Value.(type) {
		case link.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node.Name, binding.Input, value)
		case link.Reference:
			fmt.Printf("%s.%s <- %s.%s\n", binding.Node.Name, binding.Input, value.Node.Name, value.Output)
		}
	}
	return nil
}
