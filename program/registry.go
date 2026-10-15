package program

import (
	"errors"
	"fmt"
)

// Registration errors distinguish a reused address from a missing descriptor.
var (
	ErrDuplicate     = errors.New("descriptor already registered")
	ErrNilDescriptor = errors.New("nil descriptor")
)

// Registry holds named program descriptors. Its zero value is ready to use.
// Registries have independent namespaces and no package-global registration.
//
// A Registry must not be copied after its first registration. Concurrent lookups
// are safe once registration has finished; callers must synchronize registration
// with all other operations on the same registry.
type Registry struct {
	descriptors map[address]Descriptor
}

type address struct {
	path string
	name string
}

// Register associates descriptor with the exact packagePath and name strings.
// The caller supplies this identity; Register does not verify package provenance
// or require an exported Go identifier. Different addresses may hold descriptors
// of the same concrete type. Registration never calls Make.
//
// An existing address returns an error wrapping ErrDuplicate without replacing
// its descriptor. A nil interface returns ErrNilDescriptor. Typed nils retain
// ordinary Go method semantics: their Make must satisfy the Descriptor contract.
//
// Register stores the supplied interface with ordinary Go assignment semantics;
// it does not retain an alias to the variable passed as descriptor.
func (r *Registry) Register(packagePath, name string, descriptor Descriptor) error {
	key := address{path: packagePath, name: name}
	if _, exists := r.descriptors[key]; exists {
		return fmt.Errorf("%w: %s.%s", ErrDuplicate, packagePath, name)
	}
	if descriptor == nil {
		return ErrNilDescriptor
	}
	if r.descriptors == nil {
		r.descriptors = make(map[address]Descriptor)
	}
	r.descriptors[key] = descriptor
	return nil
}

// MustRegister calls [Registry.Register] and panics with its error on failure.
// It is intended for static descriptor declarations.
func (r *Registry) MustRegister(packagePath, name string, descriptor Descriptor) {
	if err := r.Register(packagePath, name, descriptor); err != nil {
		panic(err)
	}
}

// Lookup returns the descriptor registered at packagePath and name and whether
// the address is present. It does not instantiate the descriptor.
func (r *Registry) Lookup(packagePath, name string) (Descriptor, bool) {
	descriptor, ok := r.descriptors[address{path: packagePath, name: name}]
	return descriptor, ok
}
