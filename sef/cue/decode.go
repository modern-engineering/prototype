package cue

import (
	cueapi "cuelang.org/go/cue"

	"github.com/modern-engineering/prototype/solution"
)

// byteString avoids the JSON conversion path, which cannot preserve arbitrary
// byte strings. The schema has already constrained these fields to string|bytes.
type byteString string

func (s *byteString) UnmarshalCUE(value cueapi.Value) error {
	data, err := value.Bytes()
	if err == nil {
		*s = byteString(data)
	}
	return err
}

type wireArtifact struct {
	Version    string         `json:"version"`
	ID         byteString     `json:"id"`
	Revision   byteString     `json:"revision"`
	Definition wireDefinition `json:"definition"`
}

type wireDefinition struct {
	Modules  []wireModule  `json:"modules"`
	Nodes    []wireNode    `json:"nodes"`
	Bindings []wireBinding `json:"bindings"`
}

type wireModule struct {
	Path    byteString `json:"path"`
	Version byteString `json:"version"`
}

type wireSymbol struct {
	Package byteString `json:"package"`
	Name    byteString `json:"name"`
}

type wireNode struct {
	Name       byteString `json:"name"`
	Definition wireSymbol `json:"definition"`
}

type wireBinding struct {
	Node  byteString `json:"node"`
	Input byteString `json:"input"`
	Value wireValue  `json:"value"`
}

type wireValue struct {
	Kind      string         `json:"kind"`
	Text      byteString     `json:"text"`
	Reference *wireReference `json:"reference"`
}

type wireReference struct {
	Node   byteString `json:"node"`
	Output byteString `json:"output"`
}

func (wire wireArtifact) artifact() solution.Artifact {
	artifact := solution.Artifact{Version: wire.Version, ID: string(wire.ID), Revision: string(wire.Revision)}
	for _, module := range wire.Definition.Modules {
		artifact.Definition.Modules = append(artifact.Definition.Modules, solution.Module{Path: string(module.Path), Version: string(module.Version)})
	}
	for _, node := range wire.Definition.Nodes {
		artifact.Definition.Nodes = append(artifact.Definition.Nodes, solution.Node{
			Name: string(node.Name), Definition: solution.Symbol{Package: string(node.Definition.Package), Name: string(node.Definition.Name)},
		})
	}
	for _, binding := range wire.Definition.Bindings {
		var value solution.Value = solution.Literal(binding.Value.Text)
		if binding.Value.Kind == "reference" {
			value = solution.Reference{Node: string(binding.Value.Reference.Node), Output: string(binding.Value.Reference.Output)}
		}
		artifact.Definition.Bindings = append(artifact.Definition.Bindings, solution.Binding{Node: string(binding.Node), Input: string(binding.Input), Value: value})
	}
	return artifact
}
