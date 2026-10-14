// Command linked inspects compiled catalogue capabilities without constructing
// or running any program invocations.
package main

//go:generate go run ./generate

import (
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/examples/linked/design"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/symbol"
)

func main() {
	if err := inspect(); err != nil {
		log.Fatal(err)
	}
}

func inspect() error {
	var values symbol.Table
	if err := registerDefinitions(&values); err != nil {
		return err
	}
	for _, node := range design.Model.Nodes {
		value, ok := values.Lookup(node.Definition.Package, node.Definition.Name)
		if !ok {
			return fmt.Errorf("unlinked definition %s.%s", node.Definition.Package, node.Definition.Name)
		}
		_, canMake := value.(program.Descriptor)
		fmt.Printf("%s: %T, can instantiate: %t\n", node.Name, value, canMake)
	}
	for _, binding := range design.Model.Bindings {
		switch value := binding.Value.(type) {
		case solution.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, value)
		case solution.Reference:
			fmt.Printf("%s.%s <- %s.%s\n", binding.Node, binding.Input, value.Node, value.Output)
		}
	}
	return nil
}
