// Command generate emits catalogue imports and registrations for its parent
// command. Run go generate in examples/generated after changing the solution.
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
	// go generate runs this command in its package directory. Keep the generated
	// source beside main.go so ordinary go build includes your catalogue imports.
	if err := os.WriteFile("definitions_gen.go", source, 0o644); err != nil {
		log.Fatal(err)
	}
}
