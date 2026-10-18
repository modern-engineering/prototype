// Package bind exposes Go variables through string Get and Set methods.
// Wrapping a variable preserves its current value; subsequent assignments
// through either the variable or its binding remain visible through both.
// Bindings provide no synchronization.
//
// JSONInputs and JSONOutputs bind configuration structs using JSON field names
// while keeping the same scalar value representations.
package bind

import "strconv"

// StringValue is a string with Get and Set methods. Its zero value is an empty
// string. Use String to bind an existing string variable.
type StringValue string

// String binds target without changing it. The pointer must be nonnil.
func String(target *string) *StringValue {
	return (*StringValue)(target)
}

// Get returns the current string.
func (v *StringValue) Get() string { return string(*v) }

// Set replaces the string with text.
func (v *StringValue) Set(text string) error {
	*v = StringValue(text)
	return nil
}

// BoolValue is a boolean with Get and Set methods. Its zero value is false.
// Use Bool to bind an existing bool variable.
type BoolValue bool

// Bool binds target without changing it. The pointer must be nonnil.
func Bool(target *bool) *BoolValue {
	return (*BoolValue)(target)
}

// Get returns "true" or "false" using strconv.FormatBool.
func (v *BoolValue) Get() string { return strconv.FormatBool(bool(*v)) }

// IsBoolFlag permits command-line adapters to accept the flag package's
// boolean shorthand: a bare flag sets the value to true.
func (v *BoolValue) IsBoolFlag() bool { return true }

// Set parses text using strconv.ParseBool and replaces the boolean. On error the
// boolean is unchanged, and the parser's error is returned without wrapping.
func (v *BoolValue) Set(text string) error {
	value, err := strconv.ParseBool(text)
	if err != nil {
		return err
	}
	*v = BoolValue(value)
	return nil
}
