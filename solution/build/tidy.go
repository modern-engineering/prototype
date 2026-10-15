package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/modern-engineering/prototype/solution"
)

// Tidy prepares requirements and checksums for the artifact's generated host.
// Callers authoring a bundle should include the descriptor references from all
// its artifacts in Definition. Every descriptor module and the toolkit must
// already have a requirement; Tidy does not discover unspecified dependencies
// for missing catalogue packages or upgrade previously selected module versions.
// Newly exposed transitive requirements are incorporated into the result.
//
// Tidy returns a new context and leaves the artifact and authoring files alone.
// It retains unused authored requirements, and development substitutions never
// leak into the returned manifest. Authored replacement paths remain relative
// to the authoring directory. Options.OutDir can retain intermediate files.
func Tidy(ctx context.Context, artifact solution.Artifact, options Options) (updatedContext solution.ModuleContext, err error) {
	result, module, err := prepare(ctx, artifact, options)
	if err != nil {
		return solution.ModuleContext{}, err
	}
	defer func() {
		if closeErr := result.Close(); closeErr != nil {
			updatedContext = solution.ModuleContext{}
			err = errors.Join(err, closeErr)
		}
	}()
	environment := buildEnvironment()
	selected, err := moduleGraph(ctx, result.dir, environment, "-mod=mod")
	if err != nil {
		return solution.ModuleContext{}, err
	}
	resolved, err := readModule(result)
	if err != nil {
		return solution.ModuleContext{}, &Error{Stage: "module", Err: err}
	}
	command := exec.CommandContext(ctx, "go", "mod", "tidy")
	command.Dir, command.Env = result.dir, environment
	output, err := command.CombinedOutput()
	if err != nil {
		return solution.ModuleContext{}, commandError(ctx, "dependencies", err, output)
	}
	updated, err := readModule(result)
	if err != nil {
		return solution.ModuleContext{}, &Error{Stage: "module", Err: err}
	}
	present := make(map[string]bool)
	for _, requirement := range updated.Require {
		present[requirement.Path] = true
	}
	for _, requirement := range module.Require {
		if !present[requirement.Path] {
			updated.Require = append(updated.Require, requirement)
		}
	}
	// Tidy can drop checksums for authored requirements it does not import.
	// Keep their authenticated manifests alongside the retained requirements.
	sums := make(map[[2]string]bool)
	for _, sum := range updated.Sum {
		sums[[2]string{sum.Path, sum.Version}] = true
	}
	for _, sum := range resolved.Sum {
		if !sums[[2]string{sum.Path, sum.Version}] {
			updated.Sum = append(updated.Sum, sum)
		}
	}
	if err := writeModule(result, updated); err != nil {
		return solution.ModuleContext{}, &Error{Stage: "module", Err: err}
	}
	if err := checkCatalogueModules(ctx, result.dir, environment, artifact.Definition, module, selected); err != nil {
		return solution.ModuleContext{}, err
	}
	// Only the dependency edits are authoring output. Runtime source overlays and
	// absolute paths are private to this invocation.
	updated.Replace = artifact.Module.Clone().Replace
	return updated, nil
}

func readModule(result *Result) (solution.ModuleContext, error) {
	mod, err := os.ReadFile(result.ModFile)
	if err != nil {
		return solution.ModuleContext{}, err
	}
	sum, err := os.ReadFile(result.SumFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return solution.ModuleContext{}, err
	}
	if sum == nil {
		return solution.ParseModule(bytes.NewReader(mod), nil)
	}
	return solution.ParseModule(bytes.NewReader(mod), bytes.NewReader(sum))
}
