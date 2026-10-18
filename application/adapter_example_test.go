package application_test

import (
	"context"
	"flag"
	"fmt"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

type greeting struct {
	Name string
}

func (g *greeting) Inputs() program.Inputs {
	var inputs input.List
	inputs.StringVar(&g.Name, "name", "Person to greet.")
	return &inputs
}

func (g *greeting) Run(context.Context) error {
	fmt.Printf("Hello, %s!\n", g.Name)
	return nil
}

// Your catalogue can use a zero-valued runner when its inputs supply all
// configuration. Each Make call allocates independent state for the command.
func ExampleMake() {
	descriptor := application.Descriptor{
		Name:     "greet",
		MakeFunc: application.Make[greeting],
	}
	runner, inputs, _ := descriptor.Make()
	flags := flag.NewFlagSet(descriptor.Name, flag.ContinueOnError)
	if err := command.Parse(flags, inputs, []string{"-name", "Morgan"}); err != nil {
		fmt.Println(err)
		return
	}
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Hello, Morgan!
}

// If your organisation supplies configuration defaults, keep that construction
// in the descriptor. From exposes the same runner and its bound inputs.
func ExampleFrom() {
	descriptor := application.Descriptor{
		Name: "greet",
		MakeFunc: func() (program.Runner, program.Inputs) {
			return application.From(&greeting{Name: "team"})
		},
	}
	runner, inputs, _ := descriptor.Make()
	flags := flag.NewFlagSet(descriptor.Name, flag.ContinueOnError)
	if err := command.Parse(flags, inputs, nil); err != nil {
		fmt.Println(err)
		return
	}
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// Hello, team!
}
