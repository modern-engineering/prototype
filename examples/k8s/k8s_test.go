// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package k8s_test

import (
	"flag"
	"testing"

	"github.com/modern-engineering/prototype/examples/k8s"
	"github.com/modern-engineering/prototype/solution"
)

// TestCitizenship holds every exported scheme type to the catalogue
// citizenship contract through the library's own harness — a
// spellable qualifier and a repeatable, panic-free dry surface — plus
// the documentation this package owes the image.
func TestCitizenship(t *testing.T) {
	for name, st := range map[string]*solution.SchemeType{
		"Pod":      k8s.Pod,
		"Workload": k8s.Workload,
	} {
		t.Run(name, func(t *testing.T) {
			if st == nil {
				t.Fatalf("%s is nil; discovery needs a live *solution.SchemeType", name)
			}
			solution.CheckSchemeType(t, st)
			if st.Doc == "" {
				t.Error("no Doc; the image pins element documentation")
			}
		})
	}
}

// TestConventions pins the shapes the worked examples lean on. The
// qualifiers are this package's public contract with solution text —
// stanzas spell them verbatim, so a rename breaks every unit carrying
// one — and Pod's key schema is what examples/sample compiles
// against: replicas as an int defaulting to 1 (the default the image
// pins under the catalogue), priorityClass as a string.
func TestConventions(t *testing.T) {
	if k8s.Pod.Qualifier != "k8s.pod" {
		t.Errorf("Pod.Qualifier = %q, want k8s.pod", k8s.Pod.Qualifier)
	}
	if k8s.Workload.Qualifier != "k8s.workload" {
		t.Errorf("Workload.Qualifier = %q, want k8s.workload", k8s.Workload.Qualifier)
	}

	fs := flag.NewFlagSet("k8s.pod", flag.ContinueOnError)
	k8s.Pod.Params(fs)
	replicas := fs.Lookup("replicas")
	if replicas == nil || replicas.DefValue != "1" {
		t.Errorf("Pod replicas = %+v, want an int key defaulting to 1", replicas)
	}
	if fs.Lookup("priorityClass") == nil {
		t.Error("Pod declares no priorityClass key; the worked example writes it")
	}
}
