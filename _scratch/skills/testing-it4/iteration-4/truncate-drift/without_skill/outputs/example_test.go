package truncate_test

import (
	"fmt"

	"example.invalid/truncate"
)

// The doc example: two runes, three bytes, still valid UTF-8.
func ExampleHead() {
	fmt.Println(truncate.Head("héllo", 2))
	// Output: hé
}

func ExampleTail() {
	fmt.Println(truncate.Tail("héllo", 4))
	// Output: éllo
}
