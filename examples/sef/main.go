package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/sef"
)

//go:embed cleanup.json.sef
var source []byte

func main() {
	if err := inspect(); err != nil {
		log.Fatal(err)
	}
}

func inspect() error {
	document, err := sef.DecodeJSON(bytes.NewReader(source))
	if err != nil {
		return err
	}
	var rewritten bytes.Buffer
	if err := sef.EncodeJSON(&rewritten, document); err != nil {
		return err
	}
	fmt.Printf("%s at %s (SEF %s)\n", document.ID, document.Revision, document.Version)
	for _, binding := range document.Definition.Bindings {
		fmt.Printf("%s.%s = %#v\n", binding.Node, binding.Input, binding.Value)
	}
	fmt.Printf("rewritten JSON: %d bytes\n", rewritten.Len())
	return nil
}
