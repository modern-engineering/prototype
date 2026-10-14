// Command linked checks, links, and inspects a solution using manually registered
// descriptors. No program invocations are constructed.
package main

import (
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/examples/catalogue"
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
	var registry program.Registry
	for _, entry := range []struct {
		name       string
		descriptor program.Descriptor
	}{
		{"Database", catalogue.Database},
		{"Backend", catalogue.Backend},
	} {
		if err := registry.Register(design.CataloguePath, entry.name, entry.descriptor); err != nil {
			return err
		}
	}
	// Resolve performs structural and registry checks before building the graph.
	resolved, err := link.Resolve(design.Definition, &registry)
	if err != nil {
		return err
	}
	if err := verify(resolved); err != nil {
		return err
	}

	backend, _ := resolved.Lookup("backend")
	for producer := range backend.Dependencies() {
		fmt.Printf("backend depends on %s (%T)\n", producer.Name(), producer.Descriptor())
	}
	for binding := range backend.BindingsFor("database") {
		reference := binding.Value().(link.Reference)
		fmt.Printf("backend.database <- %s.%s\n", reference.Node.Name(), reference.Output)
	}
	database, _ := resolved.Lookup("database")
	for binding := range database.Consumers("address") {
		fmt.Printf("database.address is used by %s.%s\n", binding.Node().Name(), binding.Input())
	}
	return nil
}

// verify understands the application catalogue's no-output contract. This check
// needs concrete descriptors but no Make calls; exact field existence remains
// for the loader to check when it instantiates each program.
func verify(resolved *link.Solution) error {
	for binding := range resolved.Bindings() {
		reference, ok := binding.Value().(link.Reference)
		if !ok {
			continue
		}
		switch reference.Node.Descriptor().(type) {
		case application.Descriptor, *application.Descriptor:
			return fmt.Errorf("%s.%s refers to output %q of application %q", binding.Node().Name(), binding.Input(), reference.Output, reference.Node.Name())
		}
	}
	return nil
}
