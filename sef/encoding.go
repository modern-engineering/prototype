package sef

import (
	"errors"
	"fmt"

	"github.com/modern-engineering/prototype/solution"
)

// Version identifies the experimental information schema supported by the codecs.
// It is shared by all encodings; it does not imply a stability guarantee.
const Version = "0.1"

var (
	// ErrVersion means a document has an absent or unsupported schema version.
	ErrVersion = errors.New("sef: unsupported version")
	// ErrValue means a binding does not contain a supported structural value.
	ErrValue = errors.New("sef: invalid binding value")
	// ErrTrailingData means a reader contains data after its SEF document.
	ErrTrailingData = errors.New("sef: trailing data")
)

// Wire records contain no interfaces, so every encoding carries the same forms
// without registering Go types. Kind preserves an explicit empty literal even
// when an encoding omits zero-valued fields.
type wireDocument struct {
	Version    string         `json:"version"`
	ID         string         `json:"id"`
	Revision   string         `json:"revision"`
	Definition wireDefinition `json:"definition"`
}

type wireDefinition struct {
	Modules  []wireModule  `json:"modules,omitempty"`
	Nodes    []wireNode    `json:"nodes,omitempty"`
	Bindings []wireBinding `json:"bindings,omitempty"`
}

type wireModule struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type wireSymbol struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}

type wireNode struct {
	Name       string     `json:"name"`
	Definition wireSymbol `json:"definition"`
}

type wireBinding struct {
	Node  string    `json:"node"`
	Input string    `json:"input"`
	Value wireValue `json:"value"`
}

type wireValue struct {
	Kind      string         `json:"kind"`
	Text      string         `json:"text,omitempty"`
	Reference *wireReference `json:"reference,omitempty"`
}

type wireReference struct {
	Node   string `json:"node"`
	Output string `json:"output"`
}

func marshalDocument(document solution.Artifact) (wireDocument, error) {
	if document.Version != Version {
		return wireDocument{}, fmt.Errorf("%w: %q", ErrVersion, document.Version)
	}
	wire := wireDocument{Version: document.Version, ID: document.ID, Revision: document.Revision}
	for _, module := range document.Definition.Modules {
		wire.Definition.Modules = append(wire.Definition.Modules, wireModule(module))
	}
	for _, node := range document.Definition.Nodes {
		wire.Definition.Nodes = append(wire.Definition.Nodes, wireNode{
			Name: node.Name, Definition: wireSymbol(node.Definition),
		})
	}
	for i, binding := range document.Definition.Bindings {
		value, err := marshalValue(binding.Value)
		if err != nil {
			return wireDocument{}, fmt.Errorf("binding %d (%s.%s): %w", i, binding.Node, binding.Input, err)
		}
		wire.Definition.Bindings = append(wire.Definition.Bindings, wireBinding{
			Node: binding.Node, Input: binding.Input, Value: value,
		})
	}
	return wire, nil
}

func marshalValue(value solution.Value) (wireValue, error) {
	switch value := value.(type) {
	case solution.Literal:
		return wireValue{Kind: "literal", Text: string(value)}, nil
	case *solution.Literal:
		if value != nil {
			return marshalValue(*value)
		}
	case solution.Reference:
		return wireValue{Kind: "reference", Reference: &wireReference{Node: value.Node, Output: value.Output}}, nil
	case *solution.Reference:
		if value != nil {
			return marshalValue(*value)
		}
	}
	return wireValue{}, fmt.Errorf("%w: %T", ErrValue, value)
}

func unmarshalDocument(wire wireDocument) (solution.Artifact, error) {
	if wire.Version != Version {
		return solution.Artifact{}, fmt.Errorf("%w: %q", ErrVersion, wire.Version)
	}
	document := solution.Artifact{Version: wire.Version, ID: wire.ID, Revision: wire.Revision}
	for _, module := range wire.Definition.Modules {
		document.Definition.Modules = append(document.Definition.Modules, solution.Module(module))
	}
	for _, node := range wire.Definition.Nodes {
		document.Definition.Nodes = append(document.Definition.Nodes, solution.Node{
			Name: node.Name, Definition: solution.Symbol(node.Definition),
		})
	}
	for i, binding := range wire.Definition.Bindings {
		var value solution.Value
		switch binding.Value.Kind {
		case "literal":
			if binding.Value.Reference == nil {
				value = solution.Literal(binding.Value.Text)
			}
		case "reference":
			if binding.Value.Reference != nil && binding.Value.Text == "" {
				value = solution.Reference(*binding.Value.Reference)
			}
		}
		if value == nil {
			return solution.Artifact{}, fmt.Errorf("binding %d (%s.%s): %w", i, binding.Node, binding.Input, ErrValue)
		}
		document.Definition.Bindings = append(document.Definition.Bindings, solution.Binding{
			Node: binding.Node, Input: binding.Input, Value: value,
		})
	}
	return document, nil
}
