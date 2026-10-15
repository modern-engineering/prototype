// Package catalogue supplies descriptors to an independently compiled host.
package catalogue

import (
	"context"
	"github.com/modern-engineering/prototype/program"
)

type descriptor struct{}

var Service descriptor

func (descriptor) Make() (program.Runner, program.Inputs, program.Outputs) {
	return program.RunnerFunc(func(context.Context) error { panic("inspection ran a program") }),
		program.InputList{{Field: program.Field{Name: "name", Help: "service name", Sensitive: true}, Value: program.SetterFunc(func(string) error { panic("inspection assigned input") })}},
		program.OutputList{{Field: program.Field{Name: "message"}, Value: program.GetterFunc(func() string { panic("inspection read output") })}}
}
