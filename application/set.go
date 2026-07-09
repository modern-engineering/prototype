package application

import (
	"fmt"
	"go/token"
	"iter"
)

// A Set is an explicit collection of descriptors, assembled once and then
// read-only. It is the single seam where names resolve to descriptors and
// where relational invariants are checked; code that holds a *Descriptor
// never needs a Set, and nothing else in the package maintains a global
// registry.
type Set struct {
	ordered []*Descriptor
	byName  map[string]*Descriptor
}

// NewSet assembles descriptors into a Set, validating the invariants that
// individual descriptor values cannot check about each other: every
// descriptor is non-nil, every Name is a valid Go identifier (descriptors
// surface in command-line words, URLs, and config keys), and names are
// unique across the set.
func NewSet(ds ...*Descriptor) (*Set, error) {
	s := &Set{
		ordered: make([]*Descriptor, 0, len(ds)),
		byName:  make(map[string]*Descriptor, len(ds)),
	}
	for i, d := range ds {
		if d == nil {
			return nil, fmt.Errorf("application: descriptor %d is nil", i)
		}
		if !token.IsIdentifier(d.Name) {
			return nil, fmt.Errorf("application: descriptor %d has invalid name %q", i, d.Name)
		}
		if prev, ok := s.byName[d.Name]; ok {
			if prev == d {
				return nil, fmt.Errorf("application: descriptor %q listed twice", d.Name)
			}
			return nil, fmt.Errorf("application: duplicate descriptor name %q", d.Name)
		}
		s.ordered = append(s.ordered, d)
		s.byName[d.Name] = d
	}
	return s, nil
}

// Lookup returns the descriptor registered under name, or nil when the set
// holds none.
func (s *Set) Lookup(name string) *Descriptor {
	return s.byName[name]
}

// All yields the descriptors in the order they were given to NewSet.
func (s *Set) All() iter.Seq[*Descriptor] {
	return func(yield func(*Descriptor) bool) {
		for _, d := range s.ordered {
			if !yield(d) {
				return
			}
		}
	}
}

// Len reports the number of descriptors in the set.
func (s *Set) Len() int {
	return len(s.ordered)
}
