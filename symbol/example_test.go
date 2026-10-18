package symbol_test

import (
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/symbol"
)

// Lookup returns the supplied function value. The caller chooses when to call
// it, and multiple exports may have the same concrete function type.
func Example() {
	var values symbol.Table
	if err := values.Define("strings", "TrimSpace", strings.TrimSpace); err != nil {
		fmt.Println(err)
		return
	}
	if err := values.Define("strings", "ToUpper", strings.ToUpper); err != nil {
		fmt.Println(err)
		return
	}

	text := "  hello  "
	for _, name := range []string{"TrimSpace", "ToUpper"} {
		value, found := values.Lookup("strings", name)
		if !found {
			fmt.Println("missing:", name)
			return
		}
		text = value.(func(string) string)(text)
	}
	fmt.Println(text)

	// Output: HELLO
}
