package solution_test

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/solution"
)

// A project can export typed requirements without embedding Go source syntax.
// Updating the typed requirement also updates the generated module file.
func ExampleParseModule() {
	context, err := solution.ParseModule(strings.NewReader("module example.test/service\ngo 1.27.0\nrequire example.org/catalogue v1.2.0\n"), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	artifact := solution.Artifact{Version: "0.1", ID: "service", Module: context}
	artifact.Module.Require[0].Version = "v1.3.0"
	var module bytes.Buffer
	if err := artifact.Module.Write(&module, nil); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(module.String())
	// Output:
	// module example.test/service
	//
	// go 1.27.0
	//
	// require example.org/catalogue v1.3.0
}
