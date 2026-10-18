// Command solution-demo hosts SEF tools with the example Database and Backend
// descriptors compiled into its registry. It supports the same flags as sef.
//
// Inspect declared fields without assigning or executing programs:
//
//	go run ./cmd/solution-demo inspect -inputs internal/seftool/testdata/service.sef
package main

import (
	"fmt"
	"os"

	"github.com/modern-engineering/prototype/examples/catalogue"
	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/program"
)

func main() {
	var registry program.Registry
	const packagePath = "github.com/modern-engineering/prototype/examples/catalogue"
	for _, registration := range []struct {
		name       string
		descriptor program.Descriptor
	}{
		{"Database", catalogue.Database},
		{"Backend", catalogue.Backend},
	} {
		if err := registry.Register(packagePath, registration.name, registration.descriptor); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	command := seftool.Command{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Registry: &registry}
	os.Exit(command.Run(os.Args[1:]))
}
