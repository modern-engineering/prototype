package bind

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/modern-engineering/prototype/program"
)

// JSONInputs binds the immediate exported fields of target using encoding/json's
// field names. Target must be a nonnil pointer to a struct. A field's name
// defaults to its Go name; a json tag can override it, and json:"-" excludes it.
// Empty or invalid tag names use the Go name, while json:"-," names a field "-".
// Unexported fields are ignored. There is no embedded-field promotion or nested
// traversal; bind related configuration structs separately.
//
// This is field binding, not JSON encoding. All tag options, including omitempty,
// omitzero, and string, are ignored. Declarations do not depend on field values,
// and JSON marshaler methods are not called. Setters accept the same text as
// bind's scalar helpers, without JSON quoting or base64 encoding for byte slices.
//
// Metadata supplies Help and Sensitive by the resulting field name. It cannot
// rename or select fields. Fields without metadata receive only their name.
// The returned slice follows struct declaration order and copies the metadata.
//
// A field's address implementing program.Setter takes precedence, followed by
// its stored value implementing program.Setter. These handles retain their
// additional capabilities. Pointer and interface fields retain their current
// handle, including typed nils; replacing the handle later does not rebind it.
// Otherwise strings, booleans, numbers, and byte slices are bound in place,
// including defined types with those underlying types. Primitive pointers and
// slices with defined byte element types require custom setters.
//
// Binding never changes target or calls Get or Set. It panics for an invalid
// target, unsupported field, duplicate field name, or metadata naming an absent
// field or naming the same field twice. Returned handles and target share
// storage and provide no synchronization.
func JSONInputs(target any, metadata ...program.Field) []program.Input {
	var inputs []program.Input
	visitJSON(target, metadata, func(field program.Field, value reflect.Value) {
		var setter program.Setter
		if handle, ok := value.Addr().Interface().(program.Setter); ok {
			setter = handle
		} else if handle, ok := value.Interface().(program.Setter); ok {
			setter = handle
		} else {
			setter = primitive(value)
		}
		if setter == nil {
			panic(fmt.Sprintf("bind: input %q has unsupported type %s", field.Name, value.Type()))
		}
		inputs = append(inputs, program.Input{Field: field, Value: setter})
	})
	return inputs
}

// JSONOutputs binds target's immediate exported fields to getters using the
// same names, exclusions, metadata, ordering, and validation as [JSONInputs].
// A field's address implementing program.Getter takes precedence, followed by
// its stored value implementing program.Getter, then a primitive binding.
// Handles retain their additional capabilities and current storage.
// Binding never changes target or calls Get or Set. Get returns bind's ordinary
// text representations, not JSON-encoded values.
func JSONOutputs(target any, metadata ...program.Field) []program.Output {
	var outputs []program.Output
	visitJSON(target, metadata, func(field program.Field, value reflect.Value) {
		var getter program.Getter
		if handle, ok := value.Addr().Interface().(program.Getter); ok {
			getter = handle
		} else if handle, ok := value.Interface().(program.Getter); ok {
			getter = handle
		} else {
			getter = primitive(value)
		}
		if getter == nil {
			panic(fmt.Sprintf("bind: output %q has unsupported type %s", field.Name, value.Type()))
		}
		outputs = append(outputs, program.Output{Field: field, Value: getter})
	})
	return outputs
}

func visitJSON(target any, metadata []program.Field, yield func(program.Field, reflect.Value)) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		panic("bind: JSON binding requires a nonnil pointer to a struct")
	}
	declarations := make(map[string]program.Field, len(metadata))
	for _, field := range metadata {
		if _, exists := declarations[field.Name]; exists {
			panic("bind: repeated JSON metadata: " + field.Name)
		}
		declarations[field.Name] = field
	}
	value = value.Elem()
	names := make(map[string]bool, value.NumField())
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if !validJSONName(name) {
			name = field.Name
		}
		if names[name] {
			panic("bind: repeated JSON field: " + name)
		}
		names[name] = true
		declaration := program.Field{Name: name}
		if supplied, exists := declarations[name]; exists {
			declaration = supplied
		}
		yield(declaration, value.Field(index))
	}
	for _, field := range metadata {
		if !names[field.Name] {
			panic("bind: JSON metadata names an absent field: " + field.Name)
		}
	}
}

// validJSONName follows encoding/json's tag-name grammar. The tag's options
// describe JSON encoding, so only its name participates in field binding.
func validJSONName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) {
			return false
		}
	}
	return true
}

type value interface {
	program.Setter
	program.Getter
}

func primitive(target reflect.Value) value {
	var pointerType reflect.Type
	switch target.Kind() {
	case reflect.String:
		pointerType = reflect.TypeFor[*StringValue]()
	case reflect.Bool:
		pointerType = reflect.TypeFor[*BoolValue]()
	case reflect.Slice:
		pointerType = reflect.TypeFor[*BytesValue]()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return reflectedNumber{target}
	default:
		return nil
	}
	pointer := target.Addr()
	if !pointer.Type().ConvertibleTo(pointerType) {
		return nil
	}
	return pointer.Convert(pointerType).Interface().(value)
}
