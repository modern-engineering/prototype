package link_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/symbol"
)

// Definitions can be linked through handwritten registration. The resulting
// binding refers to the same occurrence objects held in the linked solution.
func ExampleResolve() {
	var values symbol.Table
	for _, entry := range []struct {
		name  string
		value any
	}{
		{"Database", catalogue.Database},
		{"Backend", catalogue.Backend},
		{"Europe", catalogue.Europe},
	} {
		if err := values.Define(design.CataloguePath, entry.name, entry.value); err != nil {
			fmt.Println(err)
			return
		}
	}
	resolved, err := link.Resolve(design.Definition, &values)
	if err != nil {
		fmt.Println(err)
		return
	}
	binding := resolved.Bindings[1]
	reference := binding.Value.(link.Reference)
	fmt.Println("receiver is backend:", binding.Node == resolved.Nodes[1])
	fmt.Println("source is database:", reference.Node == resolved.Nodes[0])
	fmt.Printf("source definition: %T\n", reference.Node.Definition)

	// Output:
	// receiver is backend: true
	// source is database: true
	// source definition: job.Descriptor
}
