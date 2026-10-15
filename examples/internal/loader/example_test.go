package loader_test

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// A loader configures a fresh invocation, runs it, and exposes its outputs.
func Example() {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		flags := flag.NewFlagSet("uppercase", flag.ContinueOnError)
		text := flags.String("text", "", "text to uppercase")
		var result string
		run := program.RunnerFunc(func(context.Context) error {
			result = strings.ToUpper(*text)
			return nil
		})
		var outputs output.List
		outputs.Var(program.GetterFunc(func() string { return result }), "result", "")
		return run, program.FromFlags(flags), &outputs
	})
	ctx := context.Background()
	i := (loader.Loader{}).Make(descriptor)
	defer func() {
		if err := i.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	text, err := i.Input("text")
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := text.Set("Hello"); err != nil {
		fmt.Println(err)
		return
	}
	if err := i.Run(ctx); err != nil {
		fmt.Println(err)
		return
	}
	result, err := i.Output("result")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result.Get())
	// Output: HELLO
}
