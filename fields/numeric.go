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

// InputInt declares an integer input initialized to value and returns its storage.
// It accepts signed, unsigned, and named integer types.
func InputInt[T integer](inputs *Inputs, name string, value T, help string) *T {
	InputIntVar(inputs, &value, name, help)
	return &value
}

// InputIntVar binds an integer input to target without changing its value.
// It accepts signed, unsigned, and named integer types. A nil target panics.
// The handle uses [bind.Int], including its parsing and formatting behavior.
func InputIntVar[T integer](inputs *Inputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Int(target), name, help)
}

// OutputInt declares an integer output initialized to value and returns its storage.
// It accepts signed, unsigned, and named integer types.
func OutputInt[T integer](outputs *Outputs, name string, value T, help string) *T {
	OutputIntVar(outputs, &value, name, help)
	return &value
}

// OutputIntVar binds an integer output to target without changing its value.
// It accepts signed, unsigned, and named integer types. A nil target panics.
// The handle uses [bind.Int], including its parsing and formatting behavior.
func OutputIntVar[T integer](outputs *Outputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Int(target), name, help)
}

// InputFloat declares a floating-point input initialized to value and returns its storage.
// It accepts float32, float64, and named floating-point types.
func InputFloat[T floating](inputs *Inputs, name string, value T, help string) *T {
	InputFloatVar(inputs, &value, name, help)
	return &value
}

// InputFloatVar binds a floating-point input to target without changing its value.
// It accepts float32, float64, and named floating-point types. A nil target panics.
// The handle uses [bind.Float], including its parsing and formatting behavior.
func InputFloatVar[T floating](inputs *Inputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Float(target), name, help)
}

// OutputFloat declares a floating-point output initialized to value and returns its storage.
// It accepts float32, float64, and named floating-point types.
func OutputFloat[T floating](outputs *Outputs, name string, value T, help string) *T {
	OutputFloatVar(outputs, &value, name, help)
	return &value
}

// OutputFloatVar binds a floating-point output to target without changing its value.
// It accepts float32, float64, and named floating-point types. A nil target panics.
// The handle uses [bind.Float], including its parsing and formatting behavior.
func OutputFloatVar[T floating](outputs *Outputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Float(target), name, help)
}

// InputComplex declares a complex input initialized to value and returns its storage.
// It accepts complex64, complex128, and named complex types.
func InputComplex[T complexNumber](inputs *Inputs, name string, value T, help string) *T {
	InputComplexVar(inputs, &value, name, help)
	return &value
}

// InputComplexVar binds a complex input to target without changing its value.
// It accepts complex64, complex128, and named complex types. A nil target panics.
// The handle uses [bind.Complex], including its parsing and formatting behavior.
func InputComplexVar[T complexNumber](inputs *Inputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil input target")
	}
	inputs.Var(bind.Complex(target), name, help)
}

// OutputComplex declares a complex output initialized to value and returns its storage.
// It accepts complex64, complex128, and named complex types.
func OutputComplex[T complexNumber](outputs *Outputs, name string, value T, help string) *T {
	OutputComplexVar(outputs, &value, name, help)
	return &value
}

// OutputComplexVar binds a complex output to target without changing its value.
// It accepts complex64, complex128, and named complex types. A nil target panics.
// The handle uses [bind.Complex], including its parsing and formatting behavior.
func OutputComplexVar[T complexNumber](outputs *Outputs, target *T, name, help string) {
	if target == nil {
		panic("fields: nil output target")
	}
	outputs.Var(bind.Complex(target), name, help)
}
