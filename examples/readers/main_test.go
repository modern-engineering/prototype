package main

import (
	"context"
	"fmt"
)

// Eager and borrowed inputs provide the same content to the invocation.
func Example() {
	if err := run(context.Background()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// eager: Hello, Ada!
	// borrowed: Hello, Ada!
}
