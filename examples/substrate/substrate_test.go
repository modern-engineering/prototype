// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package substrate_test

import (
	"fmt"
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

// TestProvisionTypes holds the provision citizens to the citizenship
// contract through the library's own harness — explicit kinds, a
// vetted output scheme, a dry Make — plus the documentation this
// package owes the image.
func TestProvisionTypes(t *testing.T) {
	for name, pt := range map[string]*solution.ProvisionType{
		"NATS":     substrate.NATS,
		"Postgres": substrate.Postgres,
	} {
		t.Run(name, func(t *testing.T) {
			if pt == nil {
				t.Fatalf("%s is nil; discovery needs a live *solution.ProvisionType", name)
			}
			solution.CheckProvisionType(t, pt)
			if pt.Doc == "" {
				t.Error("no Doc; the image pins element documentation")
			}
			for _, out := range pt.Outputs {
				if out.Doc == "" {
					t.Errorf("output %s has no Doc", out.Name)
				}
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

// TestCompileOnlyDriversRefuse pins the teaching error of the types
// without a real driver: Attach fails with the exact wording
// enactment will surface, and writes nothing — a refused type must
// not leave half a scheme behind.
func TestCompileOnlyDriversRefuse(t *testing.T) {
	for name, pt := range map[string]*solution.ProvisionType{
		"NATS":     substrate.NATS,
		"Postgres": substrate.Postgres,
	} {
		t.Run(name, func(t *testing.T) {
			w := solution.NewOutputWriter(pt.Outputs)
			err := pt.Make().Attach(t.Context(), w)
			want := fmt.Sprintf("real %s provisioning is not implemented; the type compiles solutions but cannot enact them", name)
			if err == nil || err.Error() != want {
				t.Errorf("Attach error = %v, want %q", err, want)
			}
			if missing := w.Missing(); len(missing) != len(pt.Outputs) {
				t.Errorf("refusing driver wrote outputs; missing = %v, want all %d declared", missing, len(pt.Outputs))
			}
		})
	}
}
