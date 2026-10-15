package command

import (
	"flag"
	"iter"

	"github.com/modern-engineering/prototype/program"
)

// FromFlags exposes fs as a live input surface, including flags declared after
// this call, in flag.FlagSet's lexical order. Each Value is the original
// flag.Value, preserving any additional capabilities it implements. Setter calls
// act on those values directly; they do not update FlagSet's command-line parsing
// or Visit history.
//
// Optional declarations match flags by name and provide Sensitive and nonempty
// Help overrides. Other flags retain their Usage and unmarked
// metadata. The adapter does not change flag defaults or parse arguments.
// Declaration names must be unique and name flags already present in fs.
func FromFlags(fs *flag.FlagSet, declarations ...program.Field) program.Inputs {
	metadata := make(map[string]program.Field, len(declarations))
	for _, declaration := range declarations {
		metadata[declaration.Name] = declaration
	}
	return flagInputs{flags: fs, metadata: metadata}
}

type flagInputs struct {
	flags    *flag.FlagSet
	metadata map[string]program.Field
}

func (inputs flagInputs) All() iter.Seq[program.Input] {
	return func(yield func(program.Input) bool) {
		more := true
		inputs.flags.VisitAll(func(f *flag.Flag) {
			if more {
				more = yield(inputs.input(f))
			}
		})
	}
}

func (inputs flagInputs) Lookup(name string) (program.Input, bool) {
	f := inputs.flags.Lookup(name)
	if f == nil {
		return program.Input{}, false
	}
	return inputs.input(f), true
}

func (inputs flagInputs) input(f *flag.Flag) program.Input {
	declaration := inputs.metadata[f.Name]
	declaration.Name = f.Name
	if declaration.Help == "" {
		declaration.Help = f.Usage
	}
	return program.Input{Field: declaration, Value: f.Value}
}
