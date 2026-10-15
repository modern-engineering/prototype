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

// A locally developed toolkit can supply a compiled catalogue host without a
// separately authored command. The materialized artifact is sent through stdin;
// compilation and logical checking do not construct or execute the database job.
func ExampleHost() {
	artifact := solution.Artifact{Version: "0.1", Definition: solution.Definition{
		Nodes: []solution.Node{{Name: "database", Definition: solution.Symbol{Package: build.ToolkitModule + "/examples/catalogue", Name: "Database"}}},
	}}
	result, err := build.Host(context.Background(), artifact.Definition, build.Options{ToolkitDir: "../.."})
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
