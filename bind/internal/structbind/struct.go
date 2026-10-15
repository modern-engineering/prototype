// Package structbind projects tagged struct fields into bound declarations.
package structbind

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/program"
)

// Inputs binds the immediate fields carrying input tags. It panics for invalid
// declarations without calling any getter or setter or changing target.
func Inputs(target any) []program.Input {
	var inputs []program.Input
	visit(target, "input", func(field program.Field, value reflect.Value) {
		var setter program.Setter
		if handle, ok := value.Addr().Interface().(program.Setter); ok {
			setter = handle
		} else if handle, ok := value.Interface().(program.Setter); ok {
			setter = handle
		} else {
			setter = primitive(value)
		}
		if setter == nil {
			panic(fmt.Sprintf("input: field %q has unsupported type %s", field.Name, value.Type()))
		}
		inputs = append(inputs, program.Input{Field: field, Value: setter})
	})
	return inputs
}

// Outputs binds the immediate fields carrying output tags. It panics for invalid
// declarations without calling any getter or setter or changing target.
func Outputs(target any) []program.Output {
	var outputs []program.Output
	visit(target, "output", func(field program.Field, value reflect.Value) {
		var getter program.Getter
		if handle, ok := value.Addr().Interface().(program.Getter); ok {
			getter = handle
		} else if handle, ok := value.Interface().(program.Getter); ok {
			getter = handle
		} else {
			getter = primitive(value)
		}
		if getter == nil {
			panic(fmt.Sprintf("output: field %q has unsupported type %s", field.Name, value.Type()))
		}
		outputs = append(outputs, program.Output{Field: field, Value: getter})
	})
	return outputs
}

func visit(target any, direction string, yield func(program.Field, reflect.Value)) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		panic(direction + ": Struct requires a nonnil pointer to a struct")
	}
	value = value.Elem()
	for index := range value.NumField() {
		field := value.Type().Field(index)
		tag, present := field.Tag.Lookup(direction)
		if !present || tag == "-" {
			continue
		}
		if !field.IsExported() {
			panic(fmt.Sprintf("%s: tagged field %s is unexported", direction, field.Name))
		}
		parts := strings.Split(tag, ",")
		if parts[0] == "" || parts[0] == "-" {
			panic(fmt.Sprintf("%s: field %s requires an explicit name", direction, field.Name))
		}
		declaration := program.Field{Name: parts[0], Help: field.Tag.Get("help")}
		for _, option := range parts[1:] {
			if option != "sensitive" {
				panic(fmt.Sprintf("%s: field %s has unknown tag option %q", direction, field.Name, option))
			}
			declaration.Sensitive = true
		}
		yield(declaration, value.Field(index))
	}
}

type value interface {
	program.Setter
	program.Getter
}

func primitive(target reflect.Value) value {
	var pointerType reflect.Type
	switch target.Kind() {
	case reflect.String:
		pointerType = reflect.TypeFor[*bind.StringValue]()
	case reflect.Bool:
		pointerType = reflect.TypeFor[*bind.BoolValue]()
	case reflect.Slice:
		pointerType = reflect.TypeFor[*bind.BytesValue]()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return number{target}
	default:
		return nil
	}
	pointer := target.Addr()
	if !pointer.Type().ConvertibleTo(pointerType) {
		return nil
	}
	return pointer.Convert(pointerType).Interface().(value)
}
