package cue_test

import (
	"fmt"
	"log"
	"strings"

	sefcue "github.com/modern-engineering/prototype/sef/cue"
)

func ExampleParse() {
	const source = `
version: "0.1"
let greeting = "hello"
definition: bindings: [{
    node: "print"
    input: "text"
    value: {kind: "literal", text: greeting + " world"}
}]
`
	file, err := sefcue.Parse("greeting.cue.sef", strings.NewReader(source))
	if err != nil {
		log.Fatal(err)
	}
	binding := file.Artifact().Definition.Bindings[0]
	fmt.Printf("%s.%s = %s\n", binding.Node, binding.Input, binding.Value)
	// Output:
	// print.text = hello world
}
