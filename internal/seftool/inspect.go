package seftool

import (
	"bytes"
	"fmt"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

func (c Command) inspect(w *bytes.Buffer, artifact solution.Artifact, linked, inputs bool) error {
	if err := link.CheckStructure(artifact.Definition); err != nil {
		return err
	}
	var graph *link.Solution
	var instance *load.Instance
	if linked {
		var err error
		graph, err = link.Resolve(artifact.Definition, c.Registry)
		if err != nil {
			return err
		}
		if inputs {
			instance, err = load.Instantiate(graph)
			if err != nil {
				return err
			}
		}
	}
	_, _ = fmt.Fprintf(w, "solution %q revision %q schema %q\n", artifact.ID, artifact.Revision, artifact.Version)
	for _, module := range artifact.Definition.Modules {
		_, _ = fmt.Fprintf(w, "require %q %q\n", module.Path, module.Version)
	}
	for _, node := range artifact.Definition.Nodes {
		_, _ = fmt.Fprintf(w, "node %q = %s.%s\n", node.Name, node.Definition.Package, node.Definition.Name)
		if linked {
			occurrence, _ := graph.Lookup(node.Name)
			_, _ = fmt.Fprintf(w, "  descriptor %T\n", occurrence.Descriptor())
			for dependency := range occurrence.Dependencies() {
				_, _ = fmt.Fprintf(w, "  depends on %q\n", dependency.Name())
			}
		}
		if inputs {
			invocation, _ := instance.Lookup(node.Name)
			for field := range invocation.Inputs().All() {
				_, _ = fmt.Fprintf(w, "  input %q sensitive=%t help=%q\n", field.Name, field.Sensitive, field.Help)
			}
			for field := range invocation.Outputs().All() {
				_, _ = fmt.Fprintf(w, "  output %q sensitive=%t help=%q\n", field.Name, field.Sensitive, field.Help)
			}
		}
	}
	for _, binding := range artifact.Definition.Bindings {
		_, _ = fmt.Fprintf(w, "bind %q.%q <- ", binding.Node, binding.Input)
		switch value := binding.Value.(type) {
		case solution.Literal, *solution.Literal:
			_, _ = fmt.Fprintln(w, "literal")
		case solution.Reference:
			_, _ = fmt.Fprintf(w, "%q.%q\n", value.Node, value.Output)
		case *solution.Reference:
			_, _ = fmt.Fprintf(w, "%q.%q\n", value.Node, value.Output)
		}
	}
	return nil
}
