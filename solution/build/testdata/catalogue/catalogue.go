// Package catalogue supplies descriptors to an independently compiled host.
package catalogue

import (
	"context"
	"fmt"
	"os"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/program"
)

type descriptor struct{}

var Service descriptor

func (descriptor) Make() (program.Runner, program.Inputs, program.Outputs) {
	return program.RunnerFunc(func(context.Context) error { panic("inspection ran a program") }),
		program.InputList{{Field: program.Field{Name: "name", Help: "service name", Sensitive: true}, Value: program.SetterFunc(func(string) error { panic("inspection assigned input") })}},
		program.OutputList{{Field: program.Field{Name: "message"}, Value: program.GetterFunc(func() string { panic("inspection read output") })}}
}

// Lifecycle exposes process startup and cooperative cleanup to its caller.
var Lifecycle = application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
	return program.RunnerFunc(func(ctx context.Context) error {
		fmt.Fprintln(os.Stdout, "started")
		<-ctx.Done()
		fmt.Fprintln(os.Stdout, "stopped")
		return ctx.Err()
	}), nil
}}
