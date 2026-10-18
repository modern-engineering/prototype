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
	ErrNameCapture   = errors.New("new occurrence name would capture an unresolved reference")
)

// Rename returns a copy with one occurrence renamed and all references to that
// occurrence rewritten. Descriptor symbols, input/output names, literal text,
// module requirements, and authored order are unchanged. Names are exact strings.
//
// The source occurrence must exist exactly once; a different occurrence must not
// already have newName. A name used by an unresolved binding endpoint cannot be
// captured by the rename. Other structural problems are preserved for later checks.
// On failure the result is zero and source is unchanged. On success structural
// slices and supported pointer-valued bindings are copied, including when the
// names are equal. The function never loads descriptors or calls user code.
func Rename(source solution.Definition, oldName, newName string) (solution.Definition, error) {
	count := 0
	occupied := false
	for _, node := range source.Nodes {
		switch node.Name {
		case oldName:
			count++
		case newName:
			occupied = true
		}
	}
	if count == 0 {
		return solution.Definition{}, fmt.Errorf("%w: %q", ErrMissingNode, oldName)
	}
	if count != 1 || occupied {
		return solution.Definition{}, fmt.Errorf("%w: %q -> %q", ErrDuplicateNode, oldName, newName)
	}
	if oldName != newName {
		for _, binding := range source.Bindings {
			capture := binding.Node == newName
			switch value := binding.Value.(type) {
			case solution.Reference:
				capture = capture || value.Node == newName
			case *solution.Reference:
				capture = capture || (value != nil && value.Node == newName)
			}
			if capture {
				return solution.Definition{}, fmt.Errorf("%w: %q", ErrNameCapture, newName)
			}
		}
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
