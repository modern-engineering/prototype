package fields

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
func (inputs *Inputs) Int[T integer](name string, value T, help string) *T {
	inputs.IntVar(&value, name, help)
	return &value
}

// IntVar binds an integer input to target without changing its value.
// It accepts signed, unsigned, and named integer types. A nil target panics.
// The handle uses [bind.Int], including its parsing and formatting behavior.
func (inputs *Inputs) IntVar[T integer](target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Int(target), name, help)
}

// Int declares an integer output initialized to value and returns its storage.
// It accepts signed, unsigned, and named integer types.
func (outputs *Outputs) Int[T integer](name string, value T, help string) *T {
	outputs.IntVar(&value, name, help)
	return &value
}

// IntVar binds an integer output to target without changing its value.
// It accepts signed, unsigned, and named integer types. A nil target panics.
// The handle uses [bind.Int], including its parsing and formatting behavior.
func (outputs *Outputs) IntVar[T integer](target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Int(target), name, help)
}

// Float declares a floating-point input initialized to value and returns its storage.
// It accepts float32, float64, and named floating-point types.
func (inputs *Inputs) Float[T floating](name string, value T, help string) *T {
	inputs.FloatVar(&value, name, help)
	return &value
}

// FloatVar binds a floating-point input to target without changing its value.
// It accepts float32, float64, and named floating-point types. A nil target panics.
// The handle uses [bind.Float], including its parsing and formatting behavior.
func (inputs *Inputs) FloatVar[T floating](target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Float(target), name, help)
}

// Float declares a floating-point output initialized to value and returns its storage.
// It accepts float32, float64, and named floating-point types.
func (outputs *Outputs) Float[T floating](name string, value T, help string) *T {
	outputs.FloatVar(&value, name, help)
	return &value
}

// FloatVar binds a floating-point output to target without changing its value.
// It accepts float32, float64, and named floating-point types. A nil target panics.
// The handle uses [bind.Float], including its parsing and formatting behavior.
func (outputs *Outputs) FloatVar[T floating](target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Float(target), name, help)
}

// Complex declares a complex input initialized to value and returns its storage.
// It accepts complex64, complex128, and named complex types.
func (inputs *Inputs) Complex[T complexNumber](name string, value T, help string) *T {
	inputs.ComplexVar(&value, name, help)
	return &value
}

// ComplexVar binds a complex input to target without changing its value.
// It accepts complex64, complex128, and named complex types. A nil target panics.
// The handle uses [bind.Complex], including its parsing and formatting behavior.
func (inputs *Inputs) ComplexVar[T complexNumber](target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Complex(target), name, help)
}

// Complex declares a complex output initialized to value and returns its storage.
// It accepts complex64, complex128, and named complex types.
func (outputs *Outputs) Complex[T complexNumber](name string, value T, help string) *T {
	outputs.ComplexVar(&value, name, help)
	return &value
}

// ComplexVar binds a complex output to target without changing its value.
// It accepts complex64, complex128, and named complex types. A nil target panics.
// The handle uses [bind.Complex], including its parsing and formatting behavior.
func (outputs *Outputs) ComplexVar[T complexNumber](target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Complex(target), name, help)
}
