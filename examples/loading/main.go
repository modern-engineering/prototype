// Command loading shows where your loading code chooses execution order and
// transfers a finite job's output into an application's input.
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
	registry.MustRegister(design.CataloguePath, "Database", catalogue.Database)
	registry.MustRegister(design.CataloguePath, "Backend", catalogue.Backend)
	graph, err := link.Resolve(design.Definition, &registry)
	if err != nil {
		return err
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		return err
	}

	// Your loading code chooses when to run a job and when its outputs are ready.
	// Instantiate checks and binds fields; scheduling and value transfer stay here.
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
	// Read this output after the job succeeds. An application input keeps the
	// assigned value, so your command can start the application when it is ready.
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
