// Command generate emits the catalogue linkage for the linked example.
package main

import (
	"log"
	"os"

	"github.com/modern-engineering/prototype/examples/linked/design"
	"github.com/modern-engineering/prototype/ir/link"
)

func main() {
	source, err := link.Source("main", "definitions", design.Model)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("definitions_gen.go", source, 0o644); err != nil {
		log.Fatal(err)
	}
}
