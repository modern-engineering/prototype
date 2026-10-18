package main

import (
	"context"
	"fmt"
)

// A job's output configures an application through the bound handles returned
// by Make. The unassigned scheme input retains its constructor value.
func Example() {
	address, inputs, outputs := Address.Make()
	host, _ := inputs.Lookup("host")
	if err := host.Value.Set("api.example.test:8443"); err != nil {
		fmt.Println("configure host:", err)
		return
	}
	if err := address.Run(context.Background()); err != nil {
		fmt.Println("run address:", err)
		return
	}
	url, _ := outputs.Lookup("url")
	backend, inputs, _ := Backend.Make()
	upstream, _ := inputs.Lookup("address")
	if err := upstream.Value.Set(url.Value.Get()); err != nil {
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
