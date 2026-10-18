package cue

import (
	"fmt"
	"io"
	"unicode/utf8"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
	"cuelang.org/go/cue/token"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// Encode writes canonical concrete CUE data for an artifact. It retains current
// collection order, but carries no source comments, expressions, or definitions.
// It validates the version and binding forms before writing any bytes. Invalid
// UTF-8 string data is written as CUE bytes instead of being replaced. It leaves
// w open; a write failure may leave partial output.
func Encode(w io.Writer, artifact solution.Artifact) error {
	if artifact.Version != sef.Version {
		return fmt.Errorf("%w: %q", sef.ErrVersion, artifact.Version)
	}
	var modules, nodes, bindings []ast.Expr
	for _, module := range artifact.Definition.Modules {
		modules = append(modules, ast.NewStruct("path", text(module.Path), "version", text(module.Version)))
	}
	for _, node := range artifact.Definition.Nodes {
		nodes = append(nodes, ast.NewStruct("name", text(node.Name), "definition", ast.NewStruct(
			"package", text(node.Definition.Package), "name", text(node.Definition.Name),
		)))
	}
	for i, binding := range artifact.Definition.Bindings {
		value, err := valueExpression(binding.Value)
		if err != nil {
			return fmt.Errorf("binding %d: %w", i, err)
		}
		bindings = append(bindings, ast.NewStruct("node", text(binding.Node), "input", text(binding.Input), "value", value))
	}
	root := ast.NewStruct(
		"version", ast.NewString(artifact.Version), "id", text(artifact.ID), "revision", text(artifact.Revision),
		"definition", ast.NewStruct("modules", ast.NewList(modules...), "nodes", ast.NewList(nodes...), "bindings", ast.NewList(bindings...)),
	)
	return writeSource(w, &ast.File{Decls: root.Elts})
}

func text(value string) ast.Expr {
	if utf8.ValidString(value) {
		return ast.NewString(value)
	}
	return ast.NewLit(token.STRING, literal.Bytes.Quote(value))
}

func valueExpression(value solution.Value) (ast.Expr, error) {
	switch value := value.(type) {
	case solution.Literal:
		return ast.NewStruct("kind", ast.NewString("literal"), "text", text(string(value))), nil
	case solution.Reference:
		return ast.NewStruct("kind", ast.NewString("reference"), "reference", ast.NewStruct("node", text(value.Node), "output", text(value.Output))), nil
	case *solution.Literal:
		if value != nil {
			return valueExpression(*value)
		}
	case *solution.Reference:
		if value != nil {
			return valueExpression(*value)
		}
	}
	return nil, fmt.Errorf("%w: %T", sef.ErrValue, value)
}
