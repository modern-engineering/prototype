package program_test

import (
	"errors"
	"testing"

	"github.com/modern-engineering/prototype/program"
)

// Function-backed descriptors are not comparable, and registration must neither
// invoke them nor compare them when rejecting a duplicate address.
func TestRegistryPreservesDescriptors(t *testing.T) {
	var registry, other program.Registry
	calls := 0
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		calls++
		return nil, nil, nil
	})
	registry.MustRegister("local", "shared descriptor", descriptor)
	replacement := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		panic("duplicate replaced the original descriptor")
	})
	if err := registry.Register("local", "shared descriptor", replacement); !errors.Is(err, program.ErrDuplicate) {
		t.Fatalf("duplicate registration error = %v, want ErrDuplicate", err)
	}
	value, found := registry.Lookup("local", "shared descriptor")
	if !found || calls != 0 {
		t.Fatalf("lookup found = %t, constructor calls = %d; want true, 0", found, calls)
	}
	value.Make()
	if calls != 1 {
		t.Fatalf("constructor calls = %d, want 1", calls)
	}
	if _, found := other.Lookup("local", "shared descriptor"); found {
		t.Fatal("registration leaked into another registry")
	}
	if _, found := registry.Lookup("other", "shared descriptor"); found {
		t.Fatal("lookup ignored the package path")
	}
}

// Nil interfaces are not descriptors. A concrete nil receiver can nevertheless
// implement a usable Make method, so registration does not inspect its value.
func TestRegistryRejectsOnlyNilInterface(t *testing.T) {
	var registry program.Registry
	if err := registry.Register("local", "nil", nil); !errors.Is(err, program.ErrNilDescriptor) {
		t.Fatalf("nil registration error = %v, want ErrNilDescriptor", err)
	}
	if value, found := registry.Lookup("local", "nil"); value != nil || found {
		t.Fatalf("rejected registration lookup = %v, %t", value, found)
	}
	var descriptor *nilDescriptor
	if err := registry.Register("local", "nil", descriptor); err != nil {
		t.Fatal(err)
	}
	value, found := registry.Lookup("local", "nil")
	if !found || value != descriptor {
		t.Fatalf("typed nil lookup = %#v, %t", value, found)
	}
	value.Make()
}

// Static declarations must fail without discarding earlier registrations or
// leaving an entry for a rejected declaration.
func TestStaticRegistrationPanicsWithoutChanges(t *testing.T) {
	var registry program.Registry
	var descriptor *nilDescriptor
	registry.MustRegister("local", "existing", descriptor)
	for _, test := range []struct {
		name       string
		address    string
		descriptor program.Descriptor
		want       error
	}{
		{name: "duplicate", address: "existing", descriptor: &nilDescriptor{}, want: program.ErrDuplicate},
		{name: "nil", address: "absent", descriptor: nil, want: program.ErrNilDescriptor},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				value := recover()
				err, ok := value.(error)
				if !ok || !errors.Is(err, test.want) {
					t.Fatalf("panic = %v, want error wrapping %v", value, test.want)
				}
				if value, found := registry.Lookup("local", "existing"); !found || value != descriptor {
					t.Fatalf("existing descriptor = %v, %t; want original descriptor", value, found)
				}
				if _, found := registry.Lookup("local", "absent"); found {
					t.Error("failed registration added a descriptor")
				}
			}()
			registry.MustRegister("local", test.address, test.descriptor)
		})
	}
}

type nilDescriptor struct{}

func (*nilDescriptor) Make() (program.Runner, program.Inputs, program.Outputs) {
	return nil, nil, nil
}
