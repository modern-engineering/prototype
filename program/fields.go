package program

import "iter"

// InputField binds a declaration to a setter without wrapping that setter.
type InputField struct {
	Field
	Target Setter
}

func (f InputField) Declaration() Field { return f.Field }
func (f InputField) Value() Setter      { return f.Target }

// OutputField binds a declaration to a getter without wrapping that getter.
type OutputField struct {
	Field
	Source Getter
}

func (f OutputField) Declaration() Field { return f.Field }
func (f OutputField) Value() Getter      { return f.Source }

// InputFields is an input surface in declaration order. Field names must be
// unique. A nil slice is an empty surface.
type InputFields []Input

func (fields InputFields) All() iter.Seq[Input] {
	return func(yield func(Input) bool) {
		for _, field := range fields {
			if !yield(field) {
				return
			}
		}
	}
}

func (fields InputFields) Lookup(name string) (Input, bool) {
	for _, field := range fields {
		if field.Declaration().Name == name {
			return field, true
		}
	}
	return nil, false
}

// OutputFields is an output surface in declaration order. Field names must be
// unique. A nil slice is an empty surface.
type OutputFields []Output

func (fields OutputFields) All() iter.Seq[Output] {
	return func(yield func(Output) bool) {
		for _, field := range fields {
			if !yield(field) {
				return
			}
		}
	}
}

func (fields OutputFields) Lookup(name string) (Output, bool) {
	for _, field := range fields {
		if field.Declaration().Name == name {
			return field, true
		}
	}
	return nil, false
}
