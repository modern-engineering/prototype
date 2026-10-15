package command_test

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// Your catalogue declares its input surface once. Each command can expose that
// surface through flags while retaining its own execution and resource policy.
func ExampleParse() {
	announce := application.Descriptor{
		Name: "announce",
		MakeFunc: func() (program.Runner, program.Inputs) {
			var inputs input.List
			name := inputs.String("name", "team", "audience name")
			loud := inputs.Bool("loud", false, "uppercase the greeting")
			run := program.RunnerFunc(func(context.Context) error {
				message := "Hello, " + *name
				if *loud {
					message = strings.ToUpper(message)
				}
				fmt.Println(message)
				return nil
			})
			return run, &inputs
		},
	}

	// In your main function, pass os.Args[1:] and choose how to report errors.
	runner, inputs, _ := announce.Make()
	flags := flag.NewFlagSet(announce.Name, flag.ContinueOnError)
	if err := command.Parse(flags, inputs, []string{"-name=operations", "-loud"}); err != nil {
		fmt.Println(err)
		return
	}
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// HELLO, OPERATIONS
}
