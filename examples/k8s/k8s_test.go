// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package k8s_test

import (
	"testing"

	"github.com/modern-engineering/prototype/examples/k8s"
	"github.com/modern-engineering/prototype/solution"
)

// Every exported scheme type passes the library's citizenship harness
// — a spellable qualifier and a repeatable, panic-free dry surface —
// and carries the documentation this package owes the image.
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
