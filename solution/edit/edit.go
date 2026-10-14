// Package edit transforms structural solution definitions without loading
// catalogue code. Edits preserve authored order and unrelated information.
package edit

import (
	"errors"
	"fmt"
	"slices"

	"github.com/modern-engineering/prototype/solution"
)

var (
	ErrMissingNode   = errors.New("occurrence does not exist")
	ErrDuplicateNode = errors.New("occurrence name is ambiguous or occupied")
)

// Rename returns a copy with one occurrence renamed and all references to that
// occurrence rewritten. Descriptor symbols, input/output names, literal text,
// module requirements, and authored order are unchanged. Names are exact strings.
//
// The source occurrence must exist exactly once; a different occurrence must not
// already have newName. Other structural problems are preserved for later checks.
// On failure the result is zero and source is unchanged. On success structural
// slices and supported pointer-valued bindings are copied, including when the
// names are equal. The function never loads descriptors or calls user code.
func Rename(source solution.Definition, oldName, newName string) (solution.Definition, error) {
	count := 0
	occupied := false
	for _, node := range source.Nodes {
		if node.Name == oldName {
			count++
		} else if node.Name == newName {
			occupied = true
		}
	}
	if count == 0 {
		return solution.Definition{}, fmt.Errorf("%w: %q", ErrMissingNode, oldName)
	}
	if count != 1 || occupied {
		return solution.Definition{}, fmt.Errorf("%w: %q -> %q", ErrDuplicateNode, oldName, newName)
	}
	result := solution.Definition{
		Modules:  slices.Clone(source.Modules),
		Nodes:    slices.Clone(source.Nodes),
		Bindings: slices.Clone(source.Bindings),
	}
	for i := range result.Nodes {
		if result.Nodes[i].Name == oldName {
			result.Nodes[i].Name = newName
		}
	}
	for i := range result.Bindings {
		binding := &result.Bindings[i]
		if binding.Node == oldName {
			binding.Node = newName
		}
		switch value := binding.Value.(type) {
		case solution.Reference:
			if value.Node == oldName {
				value.Node = newName
			}
			binding.Value = value
		case *solution.Reference:
			if value != nil {
				copy := *value
				if copy.Node == oldName {
					copy.Node = newName
				}
				binding.Value = &copy
			}
		case *solution.Literal:
			if value != nil {
				copy := *value
				binding.Value = &copy
			}
		}
	}
	return result, nil
}
