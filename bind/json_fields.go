package bind

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
)

type jsonField struct {
	name  string
	index []int
}

// jsonFields selects one object's fields before their values are inspected.
// Anonymous structs promote their fields unless a JSON name or an accessible
// binding capability makes the embedded field a leaf. Named structs remain
// selected fields for the caller to traverse separately.
func jsonFields(t reflect.Type, capability reflect.Type) []jsonField {
	type candidate struct {
		jsonField
		tagged bool
	}
	var candidates []candidate
	visiting := make(map[reflect.Type]bool)
	var collect func(reflect.Type, []int)
	collect = func(t reflect.Type, prefix []int) {
		if visiting[t] {
			return
		}
		visiting[t] = true
		defer delete(visiting, t)
		for i := range t.NumField() {
			field := t.Field(i)
			underlying := field.Type
			if underlying.Kind() == reflect.Pointer {
				underlying = underlying.Elem()
			}
			if !field.IsExported() && (!field.Anonymous || underlying.Kind() != reflect.Struct) {
				continue
			}
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, _, _ := strings.Cut(tag, ",")
			if !validJSONName(name) {
				name = ""
			}
			index := append(slices.Clone(prefix), i)
			handle := field.IsExported() && (field.Type.Implements(capability) || reflect.PointerTo(field.Type).Implements(capability))
			if name == "" && field.Anonymous && underlying.Kind() == reflect.Struct && !handle {
				collect(underlying, index)
				continue
			}
			tagged := name != ""
			if name == "" {
				name = field.Name
			}
			candidates = append(candidates, candidate{jsonField{name, index}, tagged})
		}
	}
	collect(t, nil)

	slices.SortFunc(candidates, func(a, b candidate) int {
		if order := strings.Compare(a.name, b.name); order != 0 {
			return order
		}
		if order := cmp.Compare(len(a.index), len(b.index)); order != 0 {
			return order
		}
		if a.tagged != b.tagged {
			if a.tagged {
				return -1
			}
			return 1
		}
		return slices.Compare(a.index, b.index)
	})
	var fields []jsonField
	for start := 0; start < len(candidates); {
		first := candidates[start]
		end := start + 1
		for end < len(candidates) && candidates[end].name == first.name {
			end++
		}
		// JSON silently omits equally dominant candidates. Binding treats that
		// ambiguity as a declaration mistake instead of losing named inputs.
		if end-start > 1 {
			second := candidates[start+1]
			if len(first.index) == len(second.index) && first.tagged == second.tagged {
				panic("bind: ambiguous JSON field: " + first.name)
			}
		}
		fields = append(fields, first.jsonField)
		start = end
	}
	slices.SortFunc(fields, func(a, b jsonField) int { return slices.Compare(a.index, b.index) })
	return fields
}
