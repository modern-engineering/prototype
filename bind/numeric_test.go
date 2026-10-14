package bind_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/modern-engineering/prototype/bind"
)

// Numeric bindings configure sampling parameters while retaining their Go types.
func Example_numeric() {
	type sampleCount uint16
	type gainValue float32
	type offsetValue complex64
	samples, gain, offset := sampleCount(64), gainValue(1), offsetValue(0)
	count := bind.Int(&samples)
	amplitude := bind.Float(&gain)
	shift := bind.Complex(&offset)
	fmt.Println(count.Get(), amplitude.Get(), shift.Get())
	if err := count.Set("256"); err != nil {
		fmt.Println(err)
		return
	}
	if err := amplitude.Set("1.25"); err != nil {
		fmt.Println(err)
		return
	}
	if err := shift.Set("2-3i"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(samples, gain, offset)
	samples, gain, offset = 512, 2, -1i
	fmt.Println(count.Get(), amplitude.Get(), shift.Get())
	// Output:
	// 64 1 (0+0i)
	// 256 1.25 (2-3i)
	// 512 2 (0-1i)
}

// Width checks must happen before conversion, especially for unsigned values
// above MaxInt64 and named types whose storage width must remain visible.
func TestIntegerBounds(t *testing.T) {
	type small int16
	type large uint64
	intMin, intMax := "-9223372036854775808", "9223372036854775807"
	intBelow, intAbove := "-9223372036854775809", "9223372036854775808"
	uintMax, uintAbove := "18446744073709551615", "18446744073709551616"
	if strconv.IntSize == 32 {
		intMin, intMax = "-2147483648", "2147483647"
		intBelow, intAbove = "-2147483649", "2147483648"
		uintMax, uintAbove = "4294967295", "4294967296"
	}
	for _, test := range []struct {
		name                   string
		value                  numericValue
		min, max, below, above string
		belowErr               error
	}{
		{name: "int", value: bind.Int(new(int)), min: intMin, max: intMax, below: intBelow, above: intAbove, belowErr: strconv.ErrRange},
		{name: "int8", value: bind.Int(new(int8)), min: "-128", max: "127", below: "-129", above: "128", belowErr: strconv.ErrRange},
		{name: "int16", value: bind.Int(new(int16)), min: "-32768", max: "32767", below: "-32769", above: "32768", belowErr: strconv.ErrRange},
		{name: "int32", value: bind.Int(new(int32)), min: "-2147483648", max: "2147483647", below: "-2147483649", above: "2147483648", belowErr: strconv.ErrRange},
		{name: "int64", value: bind.Int(new(int64)), min: "-9223372036854775808", max: "9223372036854775807", below: "-9223372036854775809", above: "9223372036854775808", belowErr: strconv.ErrRange},
		{name: "uint", value: bind.Int(new(uint)), min: "0", max: uintMax, below: "-1", above: uintAbove, belowErr: strconv.ErrSyntax},
		{name: "uint8", value: bind.Int(new(uint8)), min: "0", max: "255", below: "-1", above: "256", belowErr: strconv.ErrSyntax},
		{name: "uint16", value: bind.Int(new(uint16)), min: "0", max: "65535", below: "-1", above: "65536", belowErr: strconv.ErrSyntax},
		{name: "uint32", value: bind.Int(new(uint32)), min: "0", max: "4294967295", below: "-1", above: "4294967296", belowErr: strconv.ErrSyntax},
		{name: "uint64", value: bind.Int(new(uint64)), min: "0", max: "18446744073709551615", below: "-1", above: "18446744073709551616", belowErr: strconv.ErrSyntax},
		{name: "uintptr", value: bind.Int(new(uintptr)), min: "0", max: uintMax, below: "-1", above: uintAbove, belowErr: strconv.ErrSyntax},
		{name: "named-int16", value: bind.Int(new(small)), min: "-32768", max: "32767", below: "-32769", above: "32768", belowErr: strconv.ErrRange},
		{name: "named-uint64", value: bind.Int(new(large)), min: "0", max: "18446744073709551615", below: "-1", above: "18446744073709551616", belowErr: strconv.ErrSyntax},
	} {
		for _, text := range []string{test.min, test.max} {
			if err := test.value.Set(text); err != nil {
				t.Fatalf("%s.Set(%q): %v", test.name, text, err)
			}
			if got := test.value.Get(); got != text {
				t.Errorf("%s.Get() = %q; want %q", test.name, got, text)
			}
		}
		for _, invalid := range []struct {
			text  string
			cause error
		}{
			{text: test.below, cause: test.belowErr},
			{text: test.above, cause: strconv.ErrRange},
			{text: "0x10", cause: strconv.ErrSyntax},
			{text: "1_0", cause: strconv.ErrSyntax},
		} {
			if err := test.value.Set(invalid.text); !errors.Is(err, invalid.cause) {
				t.Errorf("%s.Set(%q) = %v; want %v", test.name, invalid.text, err, invalid.cause)
			}
			if got := test.value.Get(); got != test.max {
				t.Errorf("%s after rejected %q = %q; want %q", test.name, invalid.text, got, test.max)
			}
		}
	}
}

// Floating and complex bindings must reject overflow at their destination's
// precision rather than silently assigning infinity after a narrowing conversion.
func TestFloatingBounds(t *testing.T) {
	type approximate float32
	type precise float64
	type smallPhasor complex64
	type phasor complex128
	for _, test := range []struct {
		name            string
		value           numericValue
		valid, overflow string
	}{
		{name: "float32", value: bind.Float(new(float32)), valid: "3.4028235e+38", overflow: "3.4028236e38"},
		{name: "float64", value: bind.Float(new(float64)), valid: "1.7976931348623157e+308", overflow: "1.7976931348623159e308"},
		{name: "complex64", value: bind.Complex(new(complex64)), valid: "(3.4028235e+38+1i)", overflow: "1+3.4028236e38i"},
		{name: "complex128", value: bind.Complex(new(complex128)), valid: "(1.7976931348623157e+308+1i)", overflow: "1+1.7976931348623159e308i"},
		{name: "named-float32", value: bind.Float(new(approximate)), valid: "3.4028235e+38", overflow: "3.4028236e38"},
		{name: "named-float64", value: bind.Float(new(precise)), valid: "1.7976931348623157e+308", overflow: "1.7976931348623159e308"},
		{name: "named-complex64", value: bind.Complex(new(smallPhasor)), valid: "(3.4028235e+38+1i)", overflow: "1+3.4028236e38i"},
		{name: "named-complex128", value: bind.Complex(new(phasor)), valid: "(1.7976931348623157e+308+1i)", overflow: "1+1.7976931348623159e308i"},
	} {
		if err := test.value.Set(test.valid); err != nil {
			t.Fatalf("%s.Set(%q): %v", test.name, test.valid, err)
		}
		if got := test.value.Get(); got != test.valid {
			t.Errorf("%s.Get() = %q; want %q", test.name, got, test.valid)
		}
		for _, invalid := range []struct {
			text  string
			cause error
		}{
			{text: test.overflow, cause: strconv.ErrRange},
			{text: "not a number", cause: strconv.ErrSyntax},
		} {
			if err := test.value.Set(invalid.text); !errors.Is(err, invalid.cause) {
				t.Errorf("%s.Set(%q) = %v; want %v", test.name, invalid.text, err, invalid.cause)
			}
			if got := test.value.Get(); got != test.valid {
				t.Errorf("%s after rejected %q = %q; want %q", test.name, invalid.text, got, test.valid)
			}
		}
	}
}

type numericValue interface {
	Get() string
	Set(string) error
}
