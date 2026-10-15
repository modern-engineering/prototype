package host

import (
	"bytes"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/program"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// Inspect writes structural relationships and, when registry is non-nil, linked
// descriptor information. inputs additionally calls Make and validates actual
// I/O fields. It never calls Set, Get, or Run and omits authored literal values.
// A nil registry selects structural inspection; inputs then returns an error.
// Validation finishes before output is written. Writer failures are preserved.
func Inspect(output io.Writer, artifact solution.Artifact, registry *program.Registry, inputs bool) error {
	if inputs && registry == nil {
		return fmt.Errorf("I/O inspection requires a descriptor registry")
	}
	var buffer bytes.Buffer
	w := &buffer
	linked := registry != nil
	if err := link.CheckStructure(artifact.Definition); err != nil {
		return err
	}
	var graph *link.Solution
	var instance *load.Instance
	if linked {
		var err error
		graph, err = link.Resolve(artifact.Definition, registry)
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
	if artifact.Module.Path != "" {
		_, _ = fmt.Fprintf(w, "module %q go %q toolchain %q\n", artifact.Module.Path, artifact.Module.Go, artifact.Module.Toolchain)
	}
	for _, module := range artifact.Module.Require {
		_, _ = fmt.Fprintf(w, "require %q %q indirect=%t\n", module.Path, module.Version, module.Indirect)
	}
	for _, module := range artifact.Module.Exclude {
		_, _ = fmt.Fprintf(w, "exclude %q %q\n", module.Path, module.Version)
	}
	for _, replacement := range artifact.Module.Replace {
		_, _ = fmt.Fprintf(w, "replace %q %q => %q %q\n", replacement.Old.Path, replacement.Old.Version, replacement.New.Path, replacement.New.Version)
	}
	if len(artifact.Module.Sum) > 0 {
		_, _ = fmt.Fprintf(w, "checksums %d\n", len(artifact.Module.Sum))
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
	_, err := io.Copy(output, &buffer)
	return err
}
