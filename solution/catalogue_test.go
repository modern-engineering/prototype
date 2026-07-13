// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"testing"

	"github.com/modern-engineering/prototype/solution"
)

// TestUnpack proves the read-back mirrors the constructors: the same
// name and the very value that went in come back out, in the one arm
// the element's kind selects.
func TestUnpack(t *testing.T) {
	desc := pingDescriptor()
	prov := busProvisionType()
	sym := &solution.SymbolType{Doc: "an endpoint", Sensitive: true}
	sch := &solution.SchemeType{Doc: "pod conventions", Qualifier: "k8s.pod"}

	tests := []struct {
		name string
		el   solution.Element
		want solution.Registration
	}{
		{"app", solution.App("Ping", desc), solution.Registration{Name: "Ping", App: desc}},
		{"provision", solution.Provision("Bus", prov), solution.Registration{Name: "Bus", Provision: prov}},
		{"symbol", solution.Symbol("Endpoint", sym), solution.Registration{Name: "Endpoint", Symbol: sym}},
		{"scheme", solution.Scheme("Pod", sch), solution.Registration{Name: "Pod", Scheme: sch}},
		{"nil", nil, solution.Registration{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := solution.Unpack(tt.el); got != tt.want {
				t.Errorf("Unpack() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
