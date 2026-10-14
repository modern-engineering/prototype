// Command loading instantiates a solution, runs its finite database job, and
// explicitly transfers its output into the application's input.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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
	graph, err := link.Resolve(design.Definition, &registry)
	if err != nil {
		return err
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		return err
	}

	// This caller chooses the finite job and the point at which its outputs are
	// ready. Instantiation neither schedules programs nor transfers values.
	database, _ := instance.Lookup("database")
	for source := range database.Node().Bindings() {
		binding, _ := instance.Binding(source)
		literal, _ := binding.Literal()
		if err := binding.Input().Value.Set(literal); err != nil {
			return err
		}
	}
	if err := database.Runner().Run(context.Background()); err != nil {
		return err
	}

	backend, _ := instance.Lookup("backend")
	for source := range backend.Node().Bindings() {
		binding, _ := instance.Binding(source)
		output, _ := binding.Output()
		text := output.Value.Get()
		if err := binding.Input().Value.Set(text); err != nil {
			return err
		}
		fmt.Printf("%s.%s = %s\n", source.Node().Name(), source.Input(), text)
	}
	return nil
}
