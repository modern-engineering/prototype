package bind

import (
	"reflect"
	"strconv"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer binds a signed or unsigned integer, including a named integer type.
// Construct it with Int; its zero value has no target and cannot be used.
type Integer[T integer] struct {
	target *T
}

// Int binds target without changing it. The pointer must be nonnil.
func Int[T integer](target *T) *Integer[T] { return &Integer[T]{target: target} }

// Get formats the current integer in decimal.
func (v *Integer[T]) Get() string {
	if ^T(0) < 0 {
		return strconv.FormatInt(int64(*v.target), 10)
	}
	return strconv.FormatUint(uint64(*v.target), 10)
}

// Set parses a decimal integer using strconv.ParseInt or strconv.ParseUint at
// the target's width. On any error the target is unchanged, and the parser's
// error is returned without wrapping.
func (v *Integer[T]) Set(text string) error {
	bits := reflect.TypeFor[T]().Bits()
	if ^T(0) < 0 {
		value, err := strconv.ParseInt(text, 10, bits)
		if err != nil {
			return err
		}
		*v.target = T(value)
		return nil
	}
	value, err := strconv.ParseUint(text, 10, bits)
	if err != nil {
		return err
	}
	*v.target = T(value)
	return nil
}

type floating interface {
	~float32 | ~float64
}

// FloatValue binds a float32 or float64, including named floating-point types.
// Construct it with Float; its zero value has no target and cannot be used.
type FloatValue[T floating] struct {
	target *T
}

// Float binds target without changing it. The pointer must be nonnil.
func Float[T floating](target *T) *FloatValue[T] { return &FloatValue[T]{target: target} }

// Get formats the current float using strconv.FormatFloat's shortest 'g'
// representation that round-trips at the target's width.
func (v *FloatValue[T]) Get() string {
	return strconv.FormatFloat(float64(*v.target), 'g', -1, reflect.TypeFor[T]().Bits())
}

// Set parses text using strconv.ParseFloat at the target's width. On any error
// the target is unchanged, and the parser's error is returned without wrapping.
func (v *FloatValue[T]) Set(text string) error {
	value, err := strconv.ParseFloat(text, reflect.TypeFor[T]().Bits())
	if err != nil {
		return err
	}
	*v.target = T(value)
	return nil
}

type complexNumber interface {
	~complex64 | ~complex128
}

// ComplexValue binds a complex64 or complex128, including named complex types.
// Construct it with Complex; its zero value has no target and cannot be used.
type ComplexValue[T complexNumber] struct {
	target *T
}

// Complex binds target without changing it. The pointer must be nonnil.
func Complex[T complexNumber](target *T) *ComplexValue[T] {
	return &ComplexValue[T]{target: target}
}

// Get formats the current complex number using strconv.FormatComplex's shortest
// 'g' representation that round-trips at the target's width.
func (v *ComplexValue[T]) Get() string {
	return strconv.FormatComplex(complex128(*v.target), 'g', -1, reflect.TypeFor[T]().Bits())
}

// Set parses text using strconv.ParseComplex at the target's width. On any error
// the target is unchanged, and the parser's error is returned without wrapping.
func (v *ComplexValue[T]) Set(text string) error {
	value, err := strconv.ParseComplex(text, reflect.TypeFor[T]().Bits())
	if err != nil {
		return err
	}
	*v.target = T(value)
	return nil
}
