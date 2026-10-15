package input

import "github.com/modern-engineering/prototype/bind"

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type floating interface {
	~float32 | ~float64
}

type complexNumber interface {
	~complex64 | ~complex128
}

// Int declares an integer input initialized to value and returns its storage.
// It accepts signed, unsigned, and named integer types.
func (l *List) Int[T integer](name string, value T, help string, options ...Option) *T {
	l.IntVar(&value, name, help, options...)
	return &value
}

// IntVar binds an integer input to target without changing its value.
// It accepts signed, unsigned, and named integer types. A nil target panics.
// The handle uses [bind.Int], including its parsing and formatting behavior.
func (l *List) IntVar[T integer](target *T, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil input target")
	}
	l.Var(bind.Int(target), name, help, options...)
}

// Float declares a floating-point input initialized to value and returns its storage.
// It accepts float32, float64, and named floating-point types.
func (l *List) Float[T floating](name string, value T, help string, options ...Option) *T {
	l.FloatVar(&value, name, help, options...)
	return &value
}

// FloatVar binds a floating-point input to target without changing its value.
// It accepts float32, float64, and named floating-point types. A nil target panics.
// The handle uses [bind.Float], including its parsing and formatting behavior.
func (l *List) FloatVar[T floating](target *T, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil input target")
	}
	l.Var(bind.Float(target), name, help, options...)
}

// Complex declares a complex input initialized to value and returns its storage.
// It accepts complex64, complex128, and named complex types.
func (l *List) Complex[T complexNumber](name string, value T, help string, options ...Option) *T {
	l.ComplexVar(&value, name, help, options...)
	return &value
}

// ComplexVar binds a complex input to target without changing its value.
// It accepts complex64, complex128, and named complex types. A nil target panics.
// The handle uses [bind.Complex], including its parsing and formatting behavior.
func (l *List) ComplexVar[T complexNumber](target *T, name, help string, options ...Option) {
	if target == nil {
		panic("input: nil input target")
	}
	l.Var(bind.Complex(target), name, help, options...)
}
