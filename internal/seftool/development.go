package seftool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// development adds one local source module, without inheriting the caller's
// requirements or replacements. The artifact itself remains unchanged.
func development(ctx context.Context, definition solution.Definition, options build.Options) (solution.Definition, build.Options, error) {
	query := exec.CommandContext(ctx, "go", "env", "GOMOD")
	query.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "GO111MODULE=on", "GOTOOLCHAIN=local")
	output, err := query.Output()
	if err != nil {
		return definition, options, errors.Join(err, ctx.Err())
	}
	filename := strings.TrimSpace(string(output))
	if filename == "" || filename == os.DevNull {
		return definition, options, fmt.Errorf("%w: -dev requires a caller Go module", build.ErrModule)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return definition, options, err
	}
	caller, err := modfile.ParseLax(filename, data, nil)
	if err != nil {
		return definition, options, err
	}
	if caller.Module == nil {
		return definition, options, fmt.Errorf("%w: %s has no module declaration", build.ErrModule, filename)
	}
	path := caller.Module.Mod.Path
	directory := filepath.Dir(filename)
	const toolkit = "github.com/modern-engineering/prototype"
	explicitReplacement := slices.ContainsFunc(options.Replacements, func(r build.Replacement) bool { return r.Module == path })
	if path == toolkit && options.ToolkitDir == "" {
		if !explicitReplacement {
			options.ToolkitDir = directory
		} else if options.ToolkitVersion == "" {
			options.ToolkitVersion = "v0.0.0"
		}
	}
	referenced := false
	for _, node := range definition.Nodes {
		if !withinModule(node.Definition.Package, path) {
			continue
		}
		// A declared nested module owns its packages independently of the
		// directory containing the caller's module.
		nested := slices.ContainsFunc(definition.Modules, func(m solution.Module) bool {
			return m.Path != path && withinModule(m.Path, path) && withinModule(node.Definition.Package, m.Path)
		})
		if !nested {
			referenced = true
			break
		}
	}
	if !referenced || path == toolkit {
		return definition, options, nil
	}
	if !slices.ContainsFunc(definition.Modules, func(m solution.Module) bool { return m.Path == path }) {
		_, major, ok := module.SplitPathVersion(path)
		if !ok {
			return definition, options, fmt.Errorf("%w: invalid caller module %q", build.ErrModule, path)
		}
		prefix := module.PathMajorPrefix(major)
		if prefix == "" {
			prefix = "v0"
		}
		definition.Modules = append(slices.Clone(definition.Modules), solution.Module{Path: path, Version: prefix + ".0.0"})
	}
	if !explicitReplacement {
		options.Replacements = append(slices.Clone(options.Replacements), build.Replacement{Module: path, Dir: directory})
	}
	return definition, options, nil
}

func withinModule(pkg, module string) bool {
	return pkg == module || strings.HasPrefix(pkg, module+"/")
}
