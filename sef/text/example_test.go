package text_test

import (
	"fmt"
	"strings"

	seftext "github.com/modern-engineering/prototype/sef/text"
)

// Parsing keeps unresolved names available for later checking and linking.
func ExampleParse() {
	file, err := seftext.Parse("cleanup.sef", strings.NewReader(`
sef "0.1"
solution "cleanup"
node "print" "example.com/catalogue" "Print"
bind "print" "text" reference "trim" "text"
`))
	if err != nil {
		fmt.Println(err)
		return
	}
	artifact := file.Artifact()
	fmt.Println(artifact.ID, artifact.Version)
	binding := artifact.Definition.Bindings[0]
	fmt.Printf("%s.%s = %#v\n", binding.Node, binding.Input, binding.Value)
	// Output:
	// cleanup 0.1
	// print.text = solution.Reference{Node:"trim", Output:"text"}
}
