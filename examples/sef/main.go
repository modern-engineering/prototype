// Command sef converts a solution artifact between encodings without loading
// catalogue code. You can use the same codecs with files or network streams.
package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// Embed a reviewable artifact so the command and tests need no working directory.
//
//go:embed testdata/cleanup.json.sef
var source []byte

func main() {
	if err := inspect(); err != nil {
		log.Fatal(err)
	}
}

func inspect() error {
	// Decoding gives your tooling structural data. Resolving its descriptor names
	// and checking program fields remain separate steps.
	document, err := sef.DecodeJSON(bytes.NewReader(source))
	if err != nil {
		return err
	}
	var binary bytes.Buffer
	if err := sef.EncodeGob(&binary, document); err != nil {
		return err
	}
	document, err = sef.DecodeGob(&binary)
	if err != nil {
		return err
	}
	var rewritten bytes.Buffer
	if err := sef.EncodeJSON(&rewritten, document); err != nil {
		return err
	}
	document, err = sef.DecodeJSON(&rewritten)
	if err != nil {
		return err
	}
	fmt.Printf("%s at %s (SEF %s)\n", document.ID, document.Revision, document.Version)
	for _, binding := range document.Definition.Bindings {
		switch value := binding.Value.(type) {
		case solution.Literal:
			fmt.Printf("%s.%s = %q\n", binding.Node, binding.Input, value)
		case solution.Reference:
			fmt.Printf("%s.%s <- %s.%s\n", binding.Node, binding.Input, value.Node, value.Output)
		}
	}
	return nil
}
