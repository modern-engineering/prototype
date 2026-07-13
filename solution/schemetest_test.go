// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution"
)

// Lawful scheme-type citizens check clean — the accept half of the
// harness's own contract.
func TestCheckSchemeTypeAcceptsCitizens(t *testing.T) {
	// The compile-test schemes double as the harness's positive
	// cases: a keyed scheme, and a keyless one (nil Params claims the
	// qualifier and declares nothing).
	solution.CheckSchemeType(t, podSchemeType())
	solution.CheckSchemeType(t, &solution.SchemeType{
		Doc:       "keyless",
		Qualifier: "k8s.workload",
	})

	// A single-segment qualifier is legal: dots namespace, they are
	// not demanded.
	solution.CheckSchemeType(t, &solution.SchemeType{
		Doc:       "single segment",
		Qualifier: "rollout",
		Params:    func(fs *flag.FlagSet) { fs.Bool("canary", false, "stage a canary first") },
	})
}

// checkSchemeFails asserts CheckSchemeType rejects st, returning the
// failure message, on the failer harness CheckProvisionType's tests
// established.
func checkSchemeFails(t *testing.T, st *solution.SchemeType) string {
	t.Helper()
	f := &failer{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(failerStop); !ok {
					panic(r)
				}
			}
		}()
		solution.CheckSchemeType(f, st)
	}()
	if f.failed == "" {
		t.Fatal("CheckSchemeType accepted a scheme type that violates the contract")
	}
	return f.failed
}

// Every scheme violation fails the check with a message naming it:
// malformed qualifiers, panicking or unstable Params declarations —
// the reject half that makes the harness worth trusting.
func TestCheckSchemeTypeRejectsViolations(t *testing.T) {
	calls := 0
	tests := []struct {
		name string
		st   *solution.SchemeType
		want string
	}{
		{"nil scheme type", nil, "scheme type is nil"},
		{"empty qualifier", &solution.SchemeType{}, `qualifier ""`},
		{"hyphenated qualifier", &solution.SchemeType{Qualifier: "k8s-pod"},
			`qualifier "k8s-pod"`},
		{"keyword segment", &solution.SchemeType{Qualifier: "deploy.pod"},
			`qualifier "deploy.pod"`},
		{"dotted edge", &solution.SchemeType{Qualifier: "k8s."},
			`qualifier "k8s."`},
		{"panicking Params", &solution.SchemeType{Qualifier: "k8s.pod",
			Params: func(*flag.FlagSet) { panic("declaration exploded") }},
			"Params panicked"},
		{"unstable schema", &solution.SchemeType{Qualifier: "k8s.pod",
			Params: func(fs *flag.FlagSet) {
				calls++
				if calls > 1 {
					fs.Int("replicas", 1, "desired pod replicas")
					return
				}
				fs.String("priorityClass", "", "scheduling priority class")
			}},
			"missing on the second Params call"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkSchemeFails(t, tt.st)
			if !strings.Contains(got, tt.want) {
				t.Fatalf("failure message %q does not contain %q", got, tt.want)
			}
		})
	}
}
