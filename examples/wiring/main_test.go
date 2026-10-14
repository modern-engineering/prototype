package main

import (
	"context"
	"fmt"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// A loader configures a finite job and wires its result into an application
// through the same factory contract.
func Example() {
	l := loader.Loader{}
	address, err := l.Load(context.Background(), Address, map[string]program.Getter{
		"host": literal("api.example.test:8443"),
	})
	if err != nil {
		fmt.Println("load address:", err)
		return
	}
	if err := address.Run(context.Background()); err != nil {
		fmt.Println("run address:", err)
		return
	}
	url, err := address.Output("url")
	if err != nil {
		fmt.Println("address output:", err)
		return
	}
	backend, err := l.Load(context.Background(), Backend, map[string]program.Getter{"address": url})
	if err != nil {
		fmt.Println("load backend:", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := backend.Run(ctx); err != nil {
		fmt.Println("run backend:", err)
	}
	// Output:
	// backend configured: https://api.example.test:8443
}
