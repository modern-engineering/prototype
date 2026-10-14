package link_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
)

// A consumer can inspect relationships and concrete descriptor types without
// constructing any program invocations.
func ExampleResolve() {
	var registry program.Registry
	for _, entry := range []struct {
		name       string
		descriptor program.Descriptor
	}{
		{"Database", catalogue.Database},
		{"Backend", catalogue.Backend},
	} {
		if err := registry.Register(design.CataloguePath, entry.name, entry.descriptor); err != nil {
			fmt.Println(err)
			return
		}
	}
	resolved, err := link.Resolve(design.Definition, &registry)
	if err != nil {
		fmt.Println(err)
		return
	}
	backend, _ := resolved.Lookup("backend")
	for dependency := range backend.Dependencies() {
		fmt.Printf("backend depends on %s (%T)\n", dependency.Name(), dependency.Descriptor())
	}
	database, _ := resolved.Lookup("database")
	for binding := range database.Consumers("address") {
		fmt.Printf("database.address configures %s.%s\n", binding.Node().Name(), binding.Input())
	}

	// Output:
	// backend depends on database (job.Descriptor)
	// database.address configures backend.database
}
