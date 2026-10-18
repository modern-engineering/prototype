// Package input assembles named input declarations while a descriptor
// constructs an invocation. A *List implements program.Inputs.
//
// String, Bool, and Bytes allocate initialized variables. Their Var siblings
// bind existing variables without changing them; target pointers must be nonnil.
// Int, Float, and Complex preserve the chosen numeric type through generics.
// Var accepts custom handles, including program.SetterFunc. Helpers and Var accept
// trailing metadata options such as Sensitive; Define accepts explicit metadata.
// JSON registers a configuration struct using JSON field names via bind.JSONInputs.
// Registration copies metadata and retains the original handle, including
// its additional capabilities. It never calls Set, so constructor defaults
// remain unchanged.
//
// Names are unique within a list. Duplicate names and nil handle interfaces panic
// as declaration mistakes. Typed nil handles are retained as supplied; their
// methods determine usability. Lists and bound values provide no synchronization.
package input

import (
	"iter"

	"github.com/modern-engineering/prototype/program"
)

// List collects input declarations in registration order. Its zero value is
// an empty input surface. A List must not be copied after its first use.
type List struct {
	fields []program.Input
}

// Var defines an input with name and help, retaining value unchanged.
// Options modify the declaration in argument order before Define validates it.
// A nil option panics before registration.
func (l *List) Var(value program.Setter, name, help string, options ...Option) {
	field := program.Field{Name: name, Help: help}
	for _, option := range options {
		if option == nil {
			panic("input: nil declaration option")
		}
		option(&field)
	}
	l.Define(field, value)
}

// Define registers an input with a copy of field and the original value. It
// panics if field.Name is already defined or value is a nil interface, leaving
// existing declarations unchanged. It does not inspect or assign value.
func (l *List) Define(field program.Field, value program.Setter) {
	if value == nil {
		panic("input: nil input value")
	}
	if _, exists := l.Lookup(field.Name); exists {
		panic("input: input redefined: " + field.Name)
	}
	l.fields = append(l.fields, program.Input{Field: field, Value: value})
}

// All yields inputs in declaration order. Each iteration sees the fields
// registered when that iteration begins. Changing the returned metadata does
// not change the declarations; each Value remains the original bound handle.
func (l *List) All() iter.Seq[program.Input] {
	return func(yield func(program.Input) bool) {
		for _, field := range l.fields {
			if !yield(field) {
				return
			}
		}
	}
}

// Lookup returns the named input and whether it exists. The returned metadata
// is a copy; Value is the original bound handle.
func (l *List) Lookup(name string) (program.Input, bool) {
	for _, field := range l.fields {
		if field.Name == name {
			return field, true
		}
	}
	return program.Input{}, false
}
