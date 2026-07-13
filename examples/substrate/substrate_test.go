// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package substrate_test

import (
	"flag"
	"reflect"
	"testing"

	"github.com/modern-engineering/prototype/examples/substrate"
	"github.com/modern-engineering/prototype/solution"
)

// TestSymbolTypes pins the discovery-visible shape: exported pointer
// vars, documented, with the intended sensitivity. Symbol types carry
// no behaviour to harness (application.CheckDescriptor is for
// components), so the shape is the whole citizenship contract.
func TestSymbolTypes(t *testing.T) {
	for name, st := range map[string]*solution.SymbolType{
		"Secret":         substrate.Secret,
		"Endpoint":       substrate.Endpoint,
		"NATSCluster":    substrate.NATSCluster,
		"PostgresServer": substrate.PostgresServer,
	} {
		if st == nil {
			t.Errorf("%s is nil; discovery needs a live *solution.SymbolType", name)
			continue
		}
		if st.Doc == "" {
			t.Errorf("%s has no Doc; the image pins element documentation", name)
		}
	}
	if !substrate.Secret.Sensitive {
		t.Error("Secret must be sensitive: it exists to demonstrate taint")
	}
	if substrate.Endpoint.Sensitive {
		t.Error("Endpoint must not be sensitive: it exists to demonstrate the plain case")
	}
}

// TestProvisionTypes holds the provision citizens to their contract:
// explicit kinds (zero is a compile-catalogue fault), documented
// outputs, and a Make factory honouring the dry-instantiation
// invariant — a fresh provisioner with the same slot schema on every
// call.
func TestProvisionTypes(t *testing.T) {
	for name, pt := range map[string]*solution.ProvisionType{
		"NATS":     substrate.NATS,
		"Postgres": substrate.Postgres,
	} {
		t.Run(name, func(t *testing.T) {
			if pt == nil {
				t.Fatalf("%s is nil; discovery needs a live *solution.ProvisionType", name)
			}
			if pt.Doc == "" {
				t.Error("no Doc; the image pins element documentation")
			}
			if pt.Kinds == 0 {
				t.Error("no Kinds; registration is explicit and zero kinds is a catalogue error")
			}
			for _, out := range pt.Outputs {
				if out.Doc == "" {
					t.Errorf("output %s has no Doc", out.Name)
				}
			}
			if pt.Make == nil {
				return
			}
			if got, want := dryNames(pt), dryNames(pt); !reflect.DeepEqual(got, want) {
				t.Errorf("Make is not dry: two fresh provisioners declare %v and %v", got, want)
			}
		})
	}
	if substrate.NATS.Kinds != solution.Slice|solution.Attach {
		t.Error("NATS must register both kinds: it exists to demonstrate the kind word")
	}
	if substrate.Postgres.Kinds != solution.Attach {
		t.Error("Postgres must register attach only: it exists to demonstrate kind omission and legacy substrate")
	}
	if len(substrate.NATS.Outputs) == 0 || !substrate.NATS.Outputs[0].Sensitive {
		t.Error("NATS.config must be a sensitive output: it exists to demonstrate output taint")
	}
}

// dryNames makes one throwaway provisioner and reads back the slot
// names its flag surface carries.
func dryNames(pt *solution.ProvisionType) []string {
	var names []string
	if fs := pt.Make().Flags(); fs != nil {
		fs.VisitAll(func(f *flag.Flag) { names = append(names, f.Name) })
	}
	return names
}
