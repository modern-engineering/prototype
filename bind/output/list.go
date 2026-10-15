// Package output assembles named output declarations while a descriptor
// constructs an invocation. A *List implements program.Outputs.
//
// String, Bool, and Bytes allocate initialized variables. Their Var siblings
// bind existing variables without changing them; target pointers must be nonnil.
// Int, Float, and Complex preserve the chosen numeric type through generics.
// Var accepts custom handles, including program.GetterFunc; Define accepts
// explicit declaration metadata. Registration copies metadata and retains the
// original handle, including its additional capabilities. It never calls
// Get, so output state remains unchanged.
//
// Names are unique within a list. Duplicate names and nil handle interfaces panic
// as declaration mistakes. Typed nil handles are retained as supplied; their
// methods determine usability. Lists and bound values provide no synchronization.
package output

import (
	"iter"

	"github.com/modern-engineering/prototype/program"
)

// List collects output declarations in registration order. Its zero value is
// an empty output surface. A List must not be copied after its first use.
type List struct {
	fields []program.Output
}

// Var defines an output with name and help, retaining value unchanged.
// Use Define to supply Sensitive or other explicit declaration metadata.
func (l *List) Var(value program.Getter, name, help string) {
	l.Define(program.Field{Name: name, Help: help}, value)
}

// Define registers an output with a copy of field and the original value. It
// panics if field.Name is already defined or value is a nil interface, leaving
// existing declarations unchanged. It does not inspect or read value.
func (l *List) Define(field program.Field, value program.Getter) {
	if value == nil {
		panic("output: nil output value")
	}
	if _, exists := l.Lookup(field.Name); exists {
		panic("output: output redefined: " + field.Name)
	}
	l.fields = append(l.fields, program.Output{Field: field, Value: value})
}

// All yields outputs in declaration order. Each iteration sees the fields
// registered when that iteration begins. Changing the returned metadata does
// not change the declarations; each Value remains the original bound handle.
func (l *List) All() iter.Seq[program.Output] {
	return func(yield func(program.Output) bool) {
		for _, field := range l.fields {
			if !yield(field) {
				return
			}
		}
	}
}

// Lookup returns the named output and whether it exists. The returned metadata
// is a copy; Value is the original bound handle.
func (l *List) Lookup(name string) (program.Output, bool) {
	for _, field := range l.fields {
		if field.Name == name {
			return field, true
		}
	}
	return program.Output{}, false
}
