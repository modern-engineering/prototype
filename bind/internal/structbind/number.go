package structbind

import (
	"reflect"
	"strconv"
)

// number binds numeric types discovered at runtime, including defined types
// that cannot be named as compile-time type arguments to bind's generic helpers.
type number struct {
	target reflect.Value
}

func (n number) Get() string {
	switch n.target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(n.target.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(n.target.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(n.target.Float(), 'g', -1, n.target.Type().Bits())
	default:
		return strconv.FormatComplex(n.target.Complex(), 'g', -1, n.target.Type().Bits())
	}
}

func (n number) Set(text string) error {
	bits := n.target.Type().Bits()
	switch n.target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(text, 10, bits)
		if err != nil {
			return err
		}
		n.target.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value, err := strconv.ParseUint(text, 10, bits)
		if err != nil {
			return err
		}
		n.target.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(text, bits)
		if err != nil {
			return err
		}
		n.target.SetFloat(value)
	default:
		value, err := strconv.ParseComplex(text, bits)
		if err != nil {
			return err
		}
		n.target.SetComplex(value)
	}
	return nil
}
