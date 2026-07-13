// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"slices"

	"github.com/modern-engineering/prototype/solution/image"
)

// A Vocab is the linker-owned slice of the language vocabulary a
// statement body is checked against: the section words a body may
// open and the closed per-verb top-level field lists. It exists for
// tooling — highlighters, shell completion, source lint, a language
// server — generated from the very value the linker enforces, so a
// generated surface can never drift from the compiler.
type Vocab struct {
	// Sections are the section words a statement body may open,
	// sorted. The per-word policies (params and metadata refuse a
	// qualifier, with requires one) stay with the linker.
	Sections []string

	// RootFields maps each statement verb to the closed list of
	// top-level fields its statements may set, each list sorted; an
	// empty list means the verb takes no top-level fields. Growing a
	// verb's list is a deliberate vocabulary decision here, never a
	// catalogue side effect (D-12).
	RootFields map[string][]string
}

// vocabulary is the one authoritative value. The linker consumes it
// directly — route's section membership, the rootFields sets, and the
// diagnostics that teach both are all derived from it (compile.go) —
// and [Vocabulary] copies it out for tooling, so the checks and the
// export cannot disagree.
var vocabulary = Vocab{
	Sections: []string{"metadata", "params", "with"},
	RootFields: map[string][]string{
		image.VerbDeploy:    {"location"},
		image.VerbProvision: {},
	},
}

// Vocabulary returns the body vocabulary the linker enforces. The
// result is the caller's own copy: tooling may reorder and annotate
// it freely without reaching the linker's tables.
func Vocabulary() Vocab {
	fields := make(map[string][]string, len(vocabulary.RootFields))
	for verb, list := range vocabulary.RootFields {
		fields[verb] = slices.Clone(list)
	}
	return Vocab{Sections: slices.Clone(vocabulary.Sections), RootFields: fields}
}
