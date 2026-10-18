package main

import "fmt"

// Your tools can preserve the artifact's identity and bindings while converting
// JSON to gob and back, without importing the referenced catalogue.
func Example() {
	if err := inspect(); err != nil {
		fmt.Println(err)
	}
	// Output:
	// text-cleanup at proposal-2 (SEF 0.1)
	// trim.text = "  hello  "
	// print.prefix = ""
	// print.text <- trim.text
}
