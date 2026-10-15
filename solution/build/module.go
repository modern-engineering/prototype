package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/solution"
)

func moduleArguments(ctx context.Context, definition solution.Definition, options Options) (requirements, replacements []string, err error) {
	modules := make(map[string]string)
	for _, module := range definition.Modules {
		if module.Path == "" || module.Version == "" {
			return nil, nil, fmt.Errorf("%w: empty requirement path or version", ErrModule)
		}
		if _, exists := modules[module.Path]; exists {
			return nil, nil, fmt.Errorf("%w: repeated requirement %q", ErrModule, module.Path)
		}
		modules[module.Path] = module.Version
		requirements = append(requirements, "-require="+module.Path+"@"+module.Version)
	}
	if _, declared := modules[ToolkitModule]; !declared {
		version := options.ToolkitVersion
		if options.ToolkitDir != "" {
			version = "v0.0.0"
		}
		if version == "" {
			return nil, nil, ErrToolkitVersion
		}
		modules[ToolkitModule] = version
		requirements = append(requirements, "-require="+ToolkitModule+"@"+version)
	}
	for _, node := range definition.Nodes {
		covered := false
		for module := range modules {
			if node.Definition.Package == module || strings.HasPrefix(node.Definition.Package, module+"/") {
				covered = true
				break
			}
		}
		if !covered {
			return nil, nil, fmt.Errorf("%w: catalogue package %q has no requirement", ErrModule, node.Definition.Package)
		}
	}
	explicit := append([]Replacement(nil), options.Replacements...)
	if options.ToolkitDir != "" {
		explicit = append(explicit, Replacement{Module: ToolkitModule, Dir: options.ToolkitDir})
	}
	seen := make(map[string]bool)
	for _, replacement := range explicit {
		if replacement.Module == "" || replacement.Dir == "" || seen[replacement.Module] {
			return nil, nil, fmt.Errorf("%w: empty or repeated replacement %q", ErrModule, replacement.Module)
		}
		seen[replacement.Module] = true
		dir, err := filepath.Abs(replacement.Dir)
		if err != nil {
			return nil, nil, err
		}
		if replacement.Module == ToolkitModule {
			// Reading -json does not modify the selected source manifest.
			command := exec.CommandContext(ctx, "go", "mod", "edit", "-json", filepath.Join(dir, "go.mod"))
			command.Env = buildEnvironment()
			output, err := command.Output()
			if err != nil {
				return nil, nil, commandError(ctx, "toolkit module", err, nil)
			}
			var manifest struct{ Module struct{ Path string } }
			if err := json.Unmarshal(output, &manifest); err != nil {
				return nil, nil, err
			}
			if manifest.Module.Path != ToolkitModule {
				return nil, nil, fmt.Errorf("%w: toolkit directory declares %q", ErrModule, manifest.Module.Path)
			}
		}
		replacements = append(replacements, "-replace="+replacement.Module+"="+dir)
	}
	return requirements, replacements, nil
}

// Tidy may discard requirements not used by this host, but they remain part of
// the artifact's selected module context. Preserve them without lowering any
// requirements increased by Go's module selection.
func retainRequirements(ctx context.Context, work string, environment, requirements []string) error {
	command := exec.CommandContext(ctx, "go", "mod", "edit", "-json")
	command.Dir, command.Env = work, environment
	output, err := command.CombinedOutput()
	if err != nil {
		return commandError(ctx, "module", err, output)
	}
	var manifest struct{ Require []struct{ Path string } }
	if err := json.Unmarshal(output, &manifest); err != nil {
		return &Error{Stage: "module", Err: err}
	}
	present := make(map[string]bool)
	for _, requirement := range manifest.Require {
		present[requirement.Path] = true
	}
	args := []string{"mod", "edit"}
	for _, requirement := range requirements {
		path, _, _ := strings.Cut(strings.TrimPrefix(requirement, "-require="), "@")
		if !present[path] {
			args = append(args, requirement)
		}
	}
	if len(args) == 2 {
		return nil
	}
	command = exec.CommandContext(ctx, "go", args...)
	command.Dir, command.Env = work, environment
	output, err = command.CombinedOutput()
	if err != nil {
		return commandError(ctx, "module", err, output)
	}
	return nil
}

// Module graph selection precedes package discovery. In particular, a missing
// catalogue package must not upgrade its containing module merely to find code.
func selectedModules(ctx context.Context, work string, environment []string) (map[string]string, error) {
	command := exec.CommandContext(ctx, "go", "list", "-mod=mod", "-m", "-json", "all")
	command.Dir, command.Env = work, environment
	output, err := command.Output()
	if err != nil {
		return nil, commandError(ctx, "dependencies", err, nil)
	}
	selected := make(map[string]string)
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var module struct{ Path, Version string }
		if err := decoder.Decode(&module); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, &Error{Stage: "dependencies", Err: err}
		}
		selected[module.Path] = module.Version
	}
	return selected, nil
}

func checkCatalogueModules(ctx context.Context, work string, environment []string, definition solution.Definition, selected map[string]string) error {
	current, err := selectedModules(ctx, work, environment)
	if err != nil {
		return err
	}
	pathsChanged := make([]string, 0, len(current))
	for path := range current {
		pathsChanged = append(pathsChanged, path)
	}
	slices.Sort(pathsChanged)
	for _, path := range pathsChanged {
		version := current[path]
		previous, present := selected[path]
		if !present || version != previous {
			return &Error{Stage: "dependencies", Err: fmt.Errorf("%w: package discovery changed module %q from %q to %q", ErrModule, path, previous, version)}
		}
	}
	declared := map[string]bool{ToolkitModule: true}
	for _, module := range definition.Modules {
		declared[module.Path] = true
	}
	// Include the host itself so a missing host API in an older toolkit cannot
	// silently select a newer toolkit release during tidy.
	paths := []string{ToolkitModule + "/solution/host"}
	seen := map[string]bool{paths[0]: true}
	for _, node := range definition.Nodes {
		path := node.Definition.Package
		if !seen[path] {
			paths = append(paths, path)
			seen[path] = true
		}
	}
	args := append([]string{"list", "-mod=readonly", "-json"}, paths...)
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir, command.Env = work, environment
	output, err := command.Output()
	if err != nil {
		return commandError(ctx, "dependencies", err, nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	for {
		var pkg struct {
			ImportPath string
			Module     *struct{ Path, Version string }
		}
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return &Error{Stage: "dependencies", Err: err}
		}
		if pkg.Module == nil || !declared[pkg.Module.Path] {
			return &Error{Stage: "dependencies", Err: fmt.Errorf("%w: package %q resolved outside declared modules", ErrModule, pkg.ImportPath)}
		}
		version, present := selected[pkg.Module.Path]
		if !present || version != pkg.Module.Version {
			return &Error{Stage: "dependencies", Err: fmt.Errorf("%w: package %q changed module selection from %q to %q", ErrModule, pkg.ImportPath, version, pkg.Module.Version)}
		}
	}
}
