package text

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// Encode writes the canonical textual representation of an artifact. The three
// header directives and module context precede legacy require, node, and bind blocks;
// authored order within each collection is retained. It validates version and
// binding forms before writing, but does not vet solution relationships. It leaves
// w open. A write failure may leave partial output.
func Encode(w io.Writer, artifact solution.Artifact) error {
	if artifact.Version != sef.Version {
		return fmt.Errorf("%w: %q", sef.ErrVersion, artifact.Version)
	}
	var output bytes.Buffer
	statement(&output, "sef", []string{artifact.Version}, false)
	statement(&output, "solution", []string{artifact.ID}, false)
	statement(&output, "revision", []string{artifact.Revision}, false)
	m := artifact.Module
	if m.Path != "" {
		statement(&output, "module", []string{m.Path}, false)
	}
	if m.Go != "" {
		statement(&output, "go", []string{m.Go}, false)
	}
	if m.Toolchain != "" {
		statement(&output, "toolchain", []string{m.Toolchain}, false)
	}
	for _, entry := range m.Require {
		statement(&output, "module-require", moduleArguments(entry), false)
	}
	for _, entry := range m.Exclude {
		statement(&output, "module-exclude", moduleArguments(entry), false)
	}
	for _, entry := range m.Replace {
		args := []string{entry.Old.Path, entry.Old.Version, entry.New.Path, entry.New.Version}
		if entry.Old.Indirect || entry.New.Indirect {
			args = append(args, strconv.FormatBool(entry.Old.Indirect), strconv.FormatBool(entry.New.Indirect))
		}
		statement(&output, "module-replace", args, false)
	}
	for _, entry := range m.Sum {
		statement(&output, "module-sum", []string{entry.Path, entry.Version, entry.Hash}, false)
	}
	if len(artifact.Definition.Modules) > 0 {
		output.WriteString("\nrequire (\n")
		for _, module := range artifact.Definition.Modules {
			statement(&output, "require", moduleArguments(module), true)
		}
		output.WriteString(")\n")
	}
	if len(artifact.Definition.Nodes) > 0 {
		output.WriteString("\nnode (\n")
		for _, node := range artifact.Definition.Nodes {
			statement(&output, "node", []string{node.Name, node.Definition.Package, node.Definition.Name}, true)
		}
		output.WriteString(")\n")
	}
	if len(artifact.Definition.Bindings) > 0 {
		output.WriteString("\nbind (\n")
		for i, binding := range artifact.Definition.Bindings {
			args, err := bindingArguments(binding)
			if err != nil {
				return fmt.Errorf("binding %d: %w", i, err)
			}
			statement(&output, "bind", args, true)
		}
		output.WriteString(")\n")
	}
	_, err := output.WriteTo(w)
	return err
}

func bindingArguments(binding solution.Binding) ([]string, error) {
	prefix := []string{binding.Node, binding.Input}
	switch value := binding.Value.(type) {
	case solution.Literal:
		return append(prefix, "literal", string(value)), nil
	case solution.Reference:
		return append(prefix, "reference", value.Node, value.Output), nil
	case *solution.Literal:
		if value != nil {
			binding.Value = *value
			return bindingArguments(binding)
		}
	case *solution.Reference:
		if value != nil {
			binding.Value = *value
			return bindingArguments(binding)
		}
	}
	return nil, fmt.Errorf("%w: %T", sef.ErrValue, binding.Value)
}

func statement(output *bytes.Buffer, directive string, args []string, inBlock bool) {
	if inBlock {
		output.WriteByte('\t')
	} else {
		output.WriteString(directive)
		output.WriteByte(' ')
	}
	output.WriteString(arguments(directive, args))
	output.WriteByte('\n')
}

func arguments(directive string, args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = strconv.Quote(arg)
		if directive == "bind" && i == 2 {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}

func moduleArguments(m solution.Module) []string {
	args := []string{m.Path, m.Version}
	if m.Indirect {
		args = append(args, "true")
	}
	return args
}
