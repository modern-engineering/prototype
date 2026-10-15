package fields

import "github.com/modern-engineering/prototype/bind"

// String declares an input backed by a new string initialized to value.
// The returned pointer refers to the variable changed by its setter.
func (inputs *Inputs) String(name, value, help string) *string {
	inputs.StringVar(&value, name, help)
	return &value
}

// StringVar declares an input bound to target without changing it.
// It panics if target is nil.
func (inputs *Inputs) StringVar(target *string, name, help string) {
	if target == nil {
		panic("fields: nil string input target")
	}
	inputs.Var(bind.String(target), name, help)
}

// Bool declares an input backed by a new boolean initialized to value.
// The returned pointer refers to the variable changed by its setter.
func (inputs *Inputs) Bool(name string, value bool, help string) *bool {
	inputs.BoolVar(&value, name, help)
	return &value
}

// BoolVar declares an input bound to target without changing it.
// It panics if target is nil.
func (inputs *Inputs) BoolVar(target *bool, name, help string) {
	if target == nil {
		panic("fields: nil bool input target")
	}
	inputs.Var(bind.Bool(target), name, help)
}

// Bytes declares an input backed by a new byte-slice variable initialized to
// value. The initial backing array is shared with value. The setter and its
// io.ReaderFrom capability replace the slice through the returned pointer.
func (inputs *Inputs) Bytes(name string, value []byte, help string) *[]byte {
	inputs.BytesVar(&value, name, help)
	return &value
}

// BytesVar declares an input bound to target without changing it. The handle
// also implements io.ReaderFrom. It panics if target is nil.
func (inputs *Inputs) BytesVar(target *[]byte, name, help string) {
	if target == nil {
		panic("fields: nil bytes input target")
	}
	inputs.Var(bind.Bytes(target), name, help)
}

// String declares an output backed by a new string initialized to value.
// Assignments through the returned pointer are visible through its getter.
func (outputs *Outputs) String(name, value, help string) *string {
	outputs.StringVar(&value, name, help)
	return &value
}

// StringVar declares an output bound to target without changing it.
// It panics if target is nil.
func (outputs *Outputs) StringVar(target *string, name, help string) {
	if target == nil {
		panic("fields: nil string output target")
	}
	outputs.Var(bind.String(target), name, help)
}

// Bool declares an output backed by a new boolean initialized to value.
// Assignments through the returned pointer are visible through its getter.
func (outputs *Outputs) Bool(name string, value bool, help string) *bool {
	outputs.BoolVar(&value, name, help)
	return &value
}

// BoolVar declares an output bound to target without changing it.
// It panics if target is nil.
func (outputs *Outputs) BoolVar(target *bool, name, help string) {
	if target == nil {
		panic("fields: nil bool output target")
	}
	outputs.Var(bind.Bool(target), name, help)
}

// Bytes declares an output backed by a new byte-slice variable initialized to
// value. The initial backing array is shared with value. Assignments through the
// returned pointer are visible through the getter and its io.WriterTo capability.
func (outputs *Outputs) Bytes(name string, value []byte, help string) *[]byte {
	outputs.BytesVar(&value, name, help)
	return &value
}

// BytesVar declares an output bound to target without changing it. The handle
// also implements io.WriterTo. It panics if target is nil.
func (outputs *Outputs) BytesVar(target *[]byte, name, help string) {
	if target == nil {
		panic("fields: nil bytes output target")
	}
	outputs.Var(bind.Bytes(target), name, help)
}
