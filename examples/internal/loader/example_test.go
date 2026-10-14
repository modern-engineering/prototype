package loader_test

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// A loader configures a fresh invocation, runs it, and exposes its outputs.
func Example() {
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		flags := flag.NewFlagSet("uppercase", flag.ContinueOnError)
		text := flags.String("text", "", "text to uppercase")
		var result string
		run := program.RunnerFunc(func(context.Context) error {
			result = strings.ToUpper(*text)
			return nil
		})
		return run, program.FromFlags(flags), program.OutputList{{
			Field: program.Field{Name: "result"},
			Value: program.GetterFunc(func() string { return result }),
		}}
	})
	ctx := context.Background()
	i, err := (loader.Loader{}).Load(ctx, factory, map[string]program.Getter{"text": literal("Hello")})
	if err != nil {
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
