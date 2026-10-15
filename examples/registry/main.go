// Command registry shows how your loading code can find a descriptor by name
// and run it directly, without a solution definition or generated registration.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/program"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Register the catalogue entries your executable supports. Registration
	// stores descriptors; each Make call creates fresh program state.
	var registry program.Registry
	if err := registry.Register("example.org/catalogue", "Database", catalogue.Database); err != nil {
		return err
	}
	descriptor, _ := registry.Lookup("example.org/catalogue", "Database")
	runner, inputs, outputs := descriptor.Make()
	region, _ := inputs.Lookup("region")
	if err := region.Value.Set("eu-west"); err != nil {
		return err
	}
	if err := runner.Run(context.Background()); err != nil {
		return err
	}
	address, _ := outputs.Lookup("address")
	fmt.Println(address.Value.Get())
	return nil
}
