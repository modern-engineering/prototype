// Package symbol associates package paths and names with actual Go values.
// Callers supply the values explicitly; looking them up does not load packages,
// construct instances, or invoke functions. Concrete value types determine
// available capabilities independently of their registered addresses.
package symbol

import (
	"errors"
	"fmt"
)

// ErrDuplicate means an address already has a definition.
var ErrDuplicate = errors.New("symbol already defined")

// Table holds named Go values. Its zero value is ready to use.
// Tables have independent namespaces and no package-global registration.
//
// A Table must not be copied after its first definition. Concurrent lookups are
// safe once definitions have finished; callers must synchronize definitions
// with all other operations on the same table.
type Table struct {
	values map[address]any
}

type address struct {
	path string
	name string
}

// Define associates value with the exact packagePath and name strings.
// The caller supplies this identity; Define does not verify package provenance
// or require an exported Go identifier. Different addresses may hold values of
// the same type, and nil values are allowed.
//
// Define stores the supplied interface value with ordinary Go assignment
// semantics. It does not retain an alias to the variable passed as value;
// pointers, maps, slices, and functions retain their usual sharing behavior.
// If the address is already present, Define returns an error wrapping
// ErrDuplicate and leaves the table unchanged without comparing the values.
func (t *Table) Define(packagePath, name string, value any) error {
	key := address{path: packagePath, name: name}
	if _, exists := t.values[key]; exists {
		return fmt.Errorf("%w: %s.%s", ErrDuplicate, packagePath, name)
	}
	if t.values == nil {
		t.values = make(map[address]any)
	}
	t.values[key] = value
	return nil
}

// Lookup returns the value defined at packagePath and name. The boolean
// distinguishes an absent address from a present nil or typed-nil value.
func (t *Table) Lookup(packagePath, name string) (any, bool) {
	value, ok := t.values[address{path: packagePath, name: name}]
	return value, ok
}
