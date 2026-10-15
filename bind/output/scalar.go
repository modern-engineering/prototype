package output

import "github.com/modern-engineering/prototype/bind"

// String declares an output backed by a new string initialized to value.
// Assignments through the returned pointer are visible through its getter.
func (l *List) String(name, value, help string) *string {
	l.StringVar(&value, name, help)
	return &value
}

// StringVar declares an output bound to target without changing it.
// It panics if target is nil.
func (l *List) StringVar(target *string, name, help string) {
	if target == nil {
		panic("output: nil string output target")
	}
	l.Var(bind.String(target), name, help)
}

// Bool declares an output backed by a new boolean initialized to value.
// Assignments through the returned pointer are visible through its getter.
func (l *List) Bool(name string, value bool, help string) *bool {
	l.BoolVar(&value, name, help)
	return &value
}

// BoolVar declares an output bound to target without changing it.
// It panics if target is nil.
func (l *List) BoolVar(target *bool, name, help string) {
	if target == nil {
		panic("output: nil bool output target")
	}
	l.Var(bind.Bool(target), name, help)
}

// Bytes declares an output backed by a new byte-slice variable initialized to
// value. The initial backing array is shared with value. Assignments through the
// returned pointer are visible through the getter and its io.WriterTo capability.
func (l *List) Bytes(name string, value []byte, help string) *[]byte {
	l.BytesVar(&value, name, help)
	return &value
}

// BytesVar declares an output bound to target without changing it. The handle
// also implements io.WriterTo. It panics if target is nil.
func (l *List) BytesVar(target *[]byte, name, help string) {
	if target == nil {
		panic("output: nil bytes output target")
	}
	l.Var(bind.Bytes(target), name, help)
}
