package build_test

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// An authoring project keeps dependency intent in go.mod. Tidy prepares the
// complete context for an artifact, and Host builds without further changes.
// Development mode explicitly enables the project's local toolkit replacement.
func ExampleHost() {
	manifest, err := os.ReadFile("testdata/service/go.mod")
	if err != nil {
		fmt.Println(err)
		return
	}
	module, err := solution.ParseModule(bytes.NewReader(manifest), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	artifact := solution.Artifact{Version: "0.1", Module: module, Definition: solution.Definition{
		Nodes: []solution.Node{{Name: "database", Definition: solution.Symbol{Package: build.ToolkitModule + "/examples/catalogue", Name: "Database"}}},
	}}
	options := build.Options{Dev: true, Dir: "testdata/service"}
	artifact.Module, err = build.Tidy(context.Background(), artifact, options)
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := build.Host(context.Background(), artifact, options)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := result.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	var input bytes.Buffer
	if err := sef.EncodeGob(&input, artifact); err != nil {
		fmt.Println(err)
		return
	}
	if err := result.Run(context.Background(), []string{"check"}, &input, os.Stdout, os.Stdout); err != nil {
		fmt.Println(err)
	}
	// Output:
	// ok
}
