// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package substrate_test

import (
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
		"Secret":   substrate.Secret,
		"Endpoint": substrate.Endpoint,
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
