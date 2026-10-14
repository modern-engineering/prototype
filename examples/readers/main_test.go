package main

import (
	"context"
	"fmt"
	"os"
)

// A loader closes an eagerly consumed reader during configuration and keeps a
// borrowed reader open until the invocation finishes.
func Example() {
	if err := run(context.Background(), os.Stdout); err != nil {
		fmt.Println(err)
	}
	// Output:
	// open 1
	// consume eager
	// close 1
	// open 2
	// borrow reader
	// run: eager = "Hello, Ada!"
	// run: borrowed = "Hello, Ada!"
	// close 2
}
