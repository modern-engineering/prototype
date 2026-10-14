package main

import (
	"context"
	"fmt"

	"github.com/modern-engineering/prototype/examples/internal/loader"
)

// A loader configures a finite job and wires its result into an application
// through the same descriptor contract.
func Example() {
	l := loader.Loader{}
	address := l.Make(Address)
	defer func() {
		if err := address.Close(); err != nil {
			fmt.Println("close address:", err)
		}
	}()
	host, err := address.Input("host")
	if err != nil {
		fmt.Println("host input:", err)
		return
	}
	if err := host.Set("api.example.test:8443"); err != nil {
		fmt.Println("configure host:", err)
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
	backend := l.Make(Backend)
	defer func() {
		if err := backend.Close(); err != nil {
			fmt.Println("close backend:", err)
		}
	}()
	upstream, err := backend.Input("address")
	if err != nil {
		fmt.Println("upstream input:", err)
		return
	}
	if err := upstream.Set(url.Get()); err != nil {
		fmt.Println("configure upstream:", err)
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
