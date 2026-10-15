package input

import "github.com/modern-engineering/prototype/bind/internal/structbind"

// Struct registers the explicitly tagged, immediate exported fields of target,
// which must be a nonnil pointer to a struct. For example:
//
//	Token string `input:"token,sensitive" help:"credential"`
//
// An input tag must contain a nonempty name, optionally followed by ",sensitive".
// An absent tag or input:"-" omits the field. Help comes from the help tag.
// Fields appear in struct declaration order, after any existing inputs.
// Struct never assigns defaults, traverses nested structs, or allocates pointers.
//
// Existing setters take precedence over primitive bindings and retain their
// additional capabilities. Concrete fields are bound through their addresses;
// pointer and interface fields retain their current handle, including typed nils.
// Replacing such a stored handle later does not rebind the input.
// Otherwise strings, booleans, numbers, and byte slices are bound in place,
// including defined types with those underlying types. Primitive pointers and
// slices with defined byte element types require explicit custom setters.
// Numeric parsing follows bind's decimal, width-preserving behavior.
// Registration never calls a getter or setter.
//
// Struct panics for invalid targets or declarations, unknown tag options,
// unsupported tagged fields, or duplicate names within target or this List.
// It validates the entire batch before adding any declarations.
func (l *List) Struct(target any) {
	fields := structbind.Inputs(target)
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
