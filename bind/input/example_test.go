package input_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
)

// Constructor defaults and computed outputs live in ordinary Go variables.
// The loader sees handles bound to those same variables.
func Example() {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		var greeting string
		var inputs input.List
		var outputs output.List
		name := inputs.String("name", "world", "recipient")
		outputs.StringVar(&greeting, "greeting", "uppercase greeting")
		run := program.RunnerFunc(func(context.Context) error {
			greeting = "Hello, " + strings.ToUpper(*name)
			return nil
		})
		return run, &inputs, &outputs
	})

	run, inputs, outputs := descriptor.Make()
	name, _ := inputs.Lookup("name")
	if err := name.Value.Set("Ada"); err != nil {
		fmt.Println(err)
		return
	}
	if err := run.Run(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	greeting, _ := outputs.Lookup("greeting")
	fmt.Println(greeting.Value.Get())
	// Output: Hello, ADA
}

// Reuse your configuration's JSON names and keep declaration metadata beside
// the descriptor. Separate result state gives your outputs their own surface.
func ExampleList_JSON() {
	cfg := struct {
		Name  string `json:"name"`
		Token string `json:"token,omitempty"`
	}{Name: "world"}
	result := struct {
		Greeting string `json:"greeting"`
	}{}
	var inputs input.List
	var outputs output.List
	inputs.JSON(&cfg,
		program.Field{Name: "name", Help: "recipient"},
		program.Field{Name: "token", Help: "credential", Sensitive: true},
	)
	outputs.JSON(&result, program.Field{Name: "greeting", Help: "uppercase greeting"})

	name, _ := inputs.Lookup("name")
	if err := name.Value.Set("Ada"); err != nil {
		fmt.Println(err)
		return
	}
	result.Greeting = "Hello, " + strings.ToUpper(cfg.Name)
	greeting, _ := outputs.Lookup("greeting")
	fmt.Println(greeting.Value.Get())
	token, _ := inputs.Lookup("token")
	fmt.Println("sensitive:", token.Sensitive)
	// Output:
	// Hello, ADA
	// sensitive: true
}
