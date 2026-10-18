package input

import "github.com/modern-engineering/prototype/bind"

// String declares an input backed by a new string initialized to value.
// The returned pointer refers to the variable changed by its setter.
func (l *List) String(name, value, help string, options ...Option) *string {
	l.StringVar(&value, name, help, options...)
	return &value
}

// StringVar declares an input bound to target without changing it.
// It panics if target is nil.
func (l *List) StringVar(target *string, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil string input target")
	}
	l.Var(bind.String(target), name, help, options...)
}

// Bool declares an input backed by a new boolean initialized to value.
// The returned pointer refers to the variable changed by its setter.
func (l *List) Bool(name string, value bool, help string, options ...Option) *bool {
	l.BoolVar(&value, name, help, options...)
	return &value
}

// BoolVar declares an input bound to target without changing it.
// It panics if target is nil.
func (l *List) BoolVar(target *bool, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil bool input target")
	}
	l.Var(bind.Bool(target), name, help, options...)
}

// Bytes declares an input backed by a new byte-slice variable initialized to
// value. The initial backing array is shared with value. The setter and its
// io.ReaderFrom capability replace the slice through the returned pointer.
func (l *List) Bytes(name string, value []byte, help string, options ...Option) *[]byte {
	l.BytesVar(&value, name, help, options...)
	return &value
}

// BytesVar declares an input bound to target without changing it. The handle
// also implements io.ReaderFrom. It panics if target is nil.
func (l *List) BytesVar(target *[]byte, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil bytes input target")
	}
	l.Var(bind.Bytes(target), name, help, options...)
}
