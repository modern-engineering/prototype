package local

import (
	"context"
	"github.com/modern-engineering/prototype/program"
)

var Service = program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
	return program.RunnerFunc(func(context.Context) error { panic("runner must not execute during inspection") }),
		program.InputList{{Field: program.Field{Name: "from-local"}, Value: program.SetterFunc(func(string) error { panic("input must not be set during inspection") })}}, nil
})
