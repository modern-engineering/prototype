// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package k8s_test

import (
	"flag"
	"fmt"

	"github.com/modern-engineering/prototype/examples/k8s"
	"github.com/modern-engineering/prototype/solution"
)

// A community package publishes its conventions as exported scheme
// types: each claims the qualifier that with-stanzas spell verbatim —
// renaming one breaks every unit carrying it — and declares its key
// schema through a plain flag set. The enumeration below is the exact
// surface the linker validates stanzas against once any unit imports
// the package, defaults included: they are what the image pins under
// the catalogue, so examples/sample compiles against precisely these
// keys.
func Example() {
	for _, scheme := range []*solution.SchemeType{k8s.Pod, k8s.Workload} {
		fmt.Println(scheme.Qualifier)
		fs := flag.NewFlagSet(scheme.Qualifier, flag.ContinueOnError)
		scheme.Params(fs)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Printf("  %s = %q (%s)\n", f.Name, f.DefValue, f.Usage)
		})
	}
	// Output:
	// k8s.pod
	//   priorityClass = "" (scheduling priority class)
	//   replicas = "1" (desired pod replicas)
	// k8s.workload
	//   partOf = "" (umbrella workload this instance joins)
}
