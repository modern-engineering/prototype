// Package bind exposes Go variables through string Get and Set methods.
// Wrapping a variable preserves its current value; subsequent assignments
// through either the variable or its binding remain visible through both.
package bind

import "strconv"

// Value binds a variable of type T to its string representation. The same Value
// can serve as a getter or setter. Copies refer to the same variable. Access is
// not synchronized.
//
// Construct a Value with String, Int, or Bool; its zero value is not a binding.
type Value[T any] struct {
	target *T
	parse  func(string) (T, error)
	format func(T) string
}

// Get formats the variable's current value.
func (v Value[T]) Get() string {
	return v.format(*v.target)
}

// Set parses text and replaces the variable's value. On error the variable is
// unchanged, and the parser's error is returned without wrapping.
func (v Value[T]) Set(text string) error {
	value, err := v.parse(text)
	if err != nil {
		return err
	}
	*v.target = value
	return nil
}

// String binds target without changing it. The pointer must be nonnil.
func String(target *string) Value[string] {
	return Value[string]{
		target: target,
		parse:  func(text string) (string, error) { return text, nil },
		format: func(value string) string { return value },
	}
}

// Int binds target as a decimal integer, using strconv.Atoi and strconv.Itoa.
// It does not change target. The pointer must be nonnil.
func Int(target *int) Value[int] {
	return Value[int]{target: target, parse: strconv.Atoi, format: strconv.Itoa}
}

// Bool binds target using strconv.ParseBool and strconv.FormatBool. It does not
// change target. Get returns "true" or "false". The pointer must be nonnil.
func Bool(target *bool) Value[bool] {
	return Value[bool]{target: target, parse: strconv.ParseBool, format: strconv.FormatBool}
}
