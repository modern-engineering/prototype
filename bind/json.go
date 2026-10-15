package bind

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/modern-engineering/prototype/program"
)

// JSONInputs binds target's exported fields using encoding/json's field names.
// Target must be a nonnil pointer to a struct. A field's name defaults to its Go
// name; a json tag can override it, and json:"-" excludes it.
// Empty or invalid tag names use the Go name, while json:"-," names a field "-".
// Named structs and struct pointers are traversed recursively,
// joining names with dots: a database field with a host field becomes
// "database.host". These are flat declaration names, not JSONPath expressions.
// Literal dots in tags are preserved; any resulting name collision panics.
//
// Untagged embedded structs promote their fields. As with encoding/json, the
// shallowest field wins, and a tagged name wins over an untagged name at the
// same depth. An unresolved conflict panics rather than silently omitting the
// fields. Unexported fields are ignored, except for exported fields of embedded
// structs. Declaration order is depth-first struct field order
// after resolving promoted names.
//
// This is field binding, not JSON encoding. All tag options, including omitempty,
// omitzero, and string, are ignored. Declarations do not depend on field values,
// and JSON marshaler methods are not called. Setters accept the same text as
// bind's scalar helpers, without JSON quoting or base64 encoding for byte slices.
//
// Metadata supplies Help and Sensitive by the resulting flattened name. It cannot
// rename or select fields. Fields without metadata receive only their name.
// The returned slice copies the metadata.
//
// A field's address implementing program.Setter takes precedence over traversal,
// followed by its stored value implementing program.Setter. These handles retain
// their additional capabilities. Pointer and interface fields retain their current
// handle, including typed nils; replacing the handle later does not rebind it.
// Otherwise strings, booleans, numbers, and byte slices are bound in place,
// including defined types with those underlying types. Primitive pointers and
// slices with defined byte element types require custom setters.
//
// Binding preserves existing values and never calls Get or Set. Nil struct
// pointers needed for traversal are initialized so their fields remain available,
// including when tagged omitempty. Allocation requires a settable pointer;
// initialize unexported embedded pointers before binding. Allocations are applied
// only after the complete binding succeeds. Named recursive struct shapes panic
// because they would expand into indefinitely many declaration names.
//
// Binding also panics for an invalid target, unsupported field, duplicate field
// name, or metadata naming an absent field or naming the same field twice.
// Bindings retain their storage even if a parent pointer is later replaced.
// Returned handles and target share storage and provide no synchronization.
func JSONInputs(target any, metadata ...program.Field) []program.Input {
	var inputs []program.Input
	visitJSON(target, metadata, reflect.TypeFor[program.Setter](), func(field program.Field, value reflect.Value) {
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

// JSONOutputs binds target's exported fields to getters using the
// same names, exclusions, metadata, ordering, and validation as [JSONInputs].
// A field's address implementing program.Getter takes precedence, followed by
// its stored value implementing program.Getter, then traversal or primitive binding.
// Handles retain their additional capabilities and current storage.
// Binding initializes nil struct pointers as JSONInputs does, without calling
// Get or Set. Get returns bind's ordinary text representations, not JSON-encoded
// values.
func JSONOutputs(target any, metadata ...program.Field) []program.Output {
	var outputs []program.Output
	visitJSON(target, metadata, reflect.TypeFor[program.Getter](), func(field program.Field, value reflect.Value) {
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

func visitJSON(target any, metadata []program.Field, capability reflect.Type, yield func(program.Field, reflect.Value)) {
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
	names := make(map[string]bool)
	pointers := make(map[reflect.Value]reflect.Value)
	walkJSON(value, "", capability, make(map[reflect.Type]bool), pointers, func(name string, field reflect.Value) {
		if names[name] {
			panic("bind: repeated JSON field: " + name)
		}
		names[name] = true
		declaration := program.Field{Name: name}
		if supplied, exists := declarations[name]; exists {
			declaration = supplied
		}
		yield(declaration, field)
	})
	for _, field := range metadata {
		if !names[field.Name] {
			panic("bind: JSON metadata names an absent field: " + field.Name)
		}
	}
	for target, pointer := range pointers {
		target.Set(pointer)
	}
}

func walkJSON(value reflect.Value, prefix string, capability reflect.Type, active map[reflect.Type]bool, pointers map[reflect.Value]reflect.Value, yield func(string, reflect.Value)) {
	if active[value.Type()] {
		panic("bind: recursive JSON struct at " + prefix)
	}
	active[value.Type()] = true
	defer delete(active, value.Type())
	for _, field := range jsonFields(value.Type(), capability) {
		name := field.name
		if prefix != "" {
			name = prefix + "." + name
		}
		bound := value
		for _, index := range field.index {
			bound = indirectJSON(bound, name, pointers)
			bound = bound.Field(index)
		}
		if bound.CanInterface() && (bound.Type().Implements(capability) || bound.Addr().Type().Implements(capability)) {
			yield(name, bound)
			continue
		}
		kind := bound.Type()
		pointerTypes := make(map[reflect.Type]bool)
		for kind.Kind() == reflect.Pointer {
			if pointerTypes[kind] {
				panic("bind: recursive JSON pointer at " + name)
			}
			pointerTypes[kind] = true
			kind = kind.Elem()
		}
		if kind.Kind() == reflect.Struct {
			bound = indirectJSON(bound, name, pointers)
			walkJSON(bound, name, capability, active, pointers, yield)
			continue
		}
		yield(name, bound)
	}
}

// indirectJSON stages missing pointers without mutating target. A single staged
// pointee serves every descendant reached through the same slot, including
// promoted siblings. visitJSON applies the pointers after all fields validate.
func indirectJSON(value reflect.Value, name string, pointers map[reflect.Value]reflect.Value) reflect.Value {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			pointer, exists := pointers[value]
			if !exists {
				if !value.CanSet() {
					panic("bind: cannot initialize JSON pointer for " + name)
				}
				pointer = reflect.New(value.Type().Elem())
				pointers[value] = pointer
			}
			value = pointer.Elem()
		} else {
			value = value.Elem()
		}
	}
	return value
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
