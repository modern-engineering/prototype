// Package fields assembles named input and output surfaces while a descriptor
// constructs an invocation. Inputs and Outputs have usable empty zero values.
// Their pointers implement program.Inputs and program.Outputs respectively.
//
// Var follows flag's value/name/help registration shape. Use bind adapters for
// Go variables, program.SetterFunc and program.GetterFunc for functions, or pass
// custom handles directly. Define accepts explicit declaration metadata.
// Registration copies metadata but retains the original handle, including its
// additional capabilities. It never calls Set or Get, so constructor defaults
// and output state are unchanged.
//
// Names are unique within each surface; an input and output may share a name.
// Duplicate names and nil handle interfaces panic as declaration mistakes.
// Typed nil handles are retained as supplied; their methods determine usability.
// These sets and their bound values provide no synchronization.
package fields

import (
	"iter"

	"github.com/modern-engineering/prototype/program"
)

// Inputs collects input declarations in registration order. Its zero value is
// an empty input surface. An Inputs must not be copied after its first use.
type Inputs struct {
	list program.InputList
}

// Var defines an input with name and help, retaining value unchanged.
// Use Define to supply Sensitive or other explicit declaration metadata.
func (inputs *Inputs) Var(value program.Setter, name, help string) {
	inputs.Define(program.Field{Name: name, Help: help}, value)
}

// Define registers an input with a copy of field and the original value. It
// panics if field.Name is already defined or value is a nil interface, leaving
// existing declarations unchanged. It does not inspect or assign value.
func (inputs *Inputs) Define(field program.Field, value program.Setter) {
	if value == nil {
		panic("fields: nil input value")
	}
	if _, exists := inputs.Lookup(field.Name); exists {
		panic("fields: input redefined: " + field.Name)
	}
	inputs.list = append(inputs.list, program.Input{Field: field, Value: value})
}

// All yields inputs in declaration order. Each iteration sees the fields
// registered when that iteration begins. Changing the returned metadata does
// not change the declarations; each Value remains the original bound handle.
func (inputs *Inputs) All() iter.Seq[program.Input] {
	return func(yield func(program.Input) bool) {
		inputs.list.All()(yield)
	}
}

// Lookup returns the named input and whether it exists. The returned metadata
// is a copy; Value is the original bound handle.
func (inputs *Inputs) Lookup(name string) (program.Input, bool) {
	return inputs.list.Lookup(name)
}

// Outputs collects output declarations in registration order. Its zero value is
// an empty output surface. An Outputs must not be copied after its first use.
type Outputs struct {
	list program.OutputList
}

// Var defines an output with name and help, retaining value unchanged.
// Use Define to supply Sensitive or other explicit declaration metadata.
func (outputs *Outputs) Var(value program.Getter, name, help string) {
	outputs.Define(program.Field{Name: name, Help: help}, value)
}

// Define registers an output with a copy of field and the original value. It
// panics if field.Name is already defined or value is a nil interface, leaving
// existing declarations unchanged. It does not inspect or read value.
func (outputs *Outputs) Define(field program.Field, value program.Getter) {
	if value == nil {
		panic("fields: nil output value")
	}
	if _, exists := outputs.Lookup(field.Name); exists {
		panic("fields: output redefined: " + field.Name)
	}
	outputs.list = append(outputs.list, program.Output{Field: field, Value: value})
}

// All yields outputs in declaration order. Each iteration sees the fields
// registered when that iteration begins. Changing the returned metadata does
// not change the declarations; each Value remains the original bound handle.
func (outputs *Outputs) All() iter.Seq[program.Output] {
	return func(yield func(program.Output) bool) {
		outputs.list.All()(yield)
	}
}

// Lookup returns the named output and whether it exists. The returned metadata
// is a copy; Value is the original bound handle.
func (outputs *Outputs) Lookup(name string) (program.Output, bool) {
	return outputs.list.Lookup(name)
}
