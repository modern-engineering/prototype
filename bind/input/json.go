package input

import (
	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/program"
)

// JSON registers target's exported fields using flattened JSON names.
// It delegates binding to [bind.JSONInputs]; metadata supplies Help and Sensitive
// by the resulting field name. It appends declarations in depth-first field order.
// Binding initializes nil nested struct pointers before list registration.
// JSON validates the entire batch before adding any declarations and panics
// if binding fails or a name conflicts with an existing input.
func (l *List) JSON(target any, metadata ...program.Field) {
	fields := bind.JSONInputs(target, metadata...)
	names := make(map[string]bool, len(l.fields)+len(fields))
	for _, field := range l.fields {
		names[field.Name] = true
	}
	for _, field := range fields {
		if names[field.Name] {
			panic("input: input redefined: " + field.Name)
		}
		names[field.Name] = true
	}
	l.fields = append(l.fields, fields...)
}
