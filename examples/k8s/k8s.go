// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package k8s is the sample community-conventions catalogue: stanza
// schemes that platform and community teams publish as shared
// building blocks. Where package ff shows the component kind of
// catalogue citizen and substrate the symbol and provision kinds, k8s
// shows the fourth — exported *solution.SchemeType vars that
// with-stanzas attach to by each scheme's self-declared qualifier,
// never by a pkg.Elem reference.
//
// The stanzas are advisory: a solution may write `with k8s.pod { }`
// whether or not any unit imports this package. Importing it is what
// flips those stanzas from opaque ride-alongs to compile-checked
// blocks, solution-wide — the linker validates every stanza whose
// qualifier a registered scheme claims, and qualifiers nothing claims
// keep riding the image untouched for whichever controller recognizes
// them at enactment.
package k8s

import (
	"flag"

	"github.com/modern-engineering/prototype/solution"
)

// Pod carries the pod-level scheduling conventions a deployment
// controller reads off a deploy record. Its two keys deliberately
// span the value discipline: replicas is an int with a default the
// image pins, priorityClass a string whose values are usually written
// as bare profile tokens — which pass the dry surface unvalidated, a
// token's meaning being the controller's business.
var Pod = &solution.SchemeType{
	Doc:       "pod-level scheduling conventions",
	Qualifier: "k8s.pod",
	Params: func(fs *flag.FlagSet) {
		fs.Int("replicas", 1, "desired pod replicas")
		fs.String("priorityClass", "", "scheduling priority class")
	},
}

// Workload carries the workload grouping conventions: how instances
// aggregate into the umbrella a fleet dashboard or rollout controller
// groups by. It exists beside Pod to show one package publishing
// several schemes, each claiming its own qualifier.
var Workload = &solution.SchemeType{
	Doc:       "workload grouping conventions",
	Qualifier: "k8s.workload",
	Params: func(fs *flag.FlagSet) {
		fs.String("partOf", "", "umbrella workload this instance joins")
	},
}
