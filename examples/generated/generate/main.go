// Command generate emits the catalogue linkage for the generated example.
package main

import (
	"log"
	"os"

	"github.com/modern-engineering/prototype/examples/design"
	"github.com/modern-engineering/prototype/solution/gen"
)

func main() {
	source, err := gen.Source("main", "registerDefinitions", design.Definition)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("definitions_gen.go", source, 0o644); err != nil {
		log.Fatal(err)
	}
}
