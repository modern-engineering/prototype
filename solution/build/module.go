package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/solution"
	"golang.org/x/mod/modfile"
)

func hostModule(ctx context.Context, artifact solution.Artifact, options Options) (solution.ModuleContext, error) {
	if err := ctx.Err(); err != nil {
		return solution.ModuleContext{}, err
	}
	module := artifact.Module.Clone()
	if err := module.Write(io.Discard, io.Discard); err != nil {
		return module, errors.Join(ErrModule, err)
	}
	declared := make(map[string]bool)
	for _, requirement := range module.Require {
		declared[requirement.Path] = true
	}
	if !declared[ToolkitModule] {
		return module, fmt.Errorf("%w: toolkit %q has no requirement", ErrModule, ToolkitModule)
	}
	for _, node := range artifact.Definition.Nodes {
		covered := false
		for path := range declared {
			if node.Definition.Package == path || strings.HasPrefix(node.Definition.Package, path+"/") {
				covered = true
				break
			}
		}
		if !covered {
			return module, fmt.Errorf("%w: catalogue package %q has no requirement", ErrModule, node.Definition.Package)
		}
	}
	base, err := filepath.Abs(options.Dir)
	if err != nil {
		return module, err
	}
	for i := range module.Replace {
		replacement := &module.Replace[i]
		if replacement.New.Version != "" {
			continue
		}
		if !options.Dev {
			return module, fmt.Errorf("%w: local replacement %q requires development mode", ErrModule, replacement.Old.Path)
		}
		if !filepath.IsAbs(replacement.New.Path) {
			replacement.New.Path = filepath.Join(base, replacement.New.Path)
		}
	}
	if options.Dev {
		if err := overlayDevelopment(&module, base, declared); err != nil {
			return module, err
		}
	}
	return module, nil
}

func writeModule(result *Result, module solution.ModuleContext) error {
	var mod, sum bytes.Buffer
	if err := module.Write(&mod, &sum); err != nil {
		return err
	}
	if err := os.WriteFile(result.ModFile, mod.Bytes(), 0o600); err != nil {
		return err
	}
	return os.WriteFile(result.SumFile, sum.Bytes(), 0o600)
}

// Local source substitutions belong only to this build. The workspace itself is
// never rewritten, and a local module cannot introduce a missing requirement.
func overlayDevelopment(module *solution.ModuleContext, base string, declared map[string]bool) error {
	var directories []string
	workspace := os.Getenv("GOWORK")
	if workspace == "" || workspace == "auto" {
		workspace = enclosingFile(base, "go.work")
	}
	if workspace != "" && workspace != "off" {
		if !filepath.IsAbs(workspace) {
			return fmt.Errorf("%w: GOWORK must be an absolute path", ErrModule)
		}
		data, err := os.ReadFile(workspace)
		if err != nil {
			return err
		}
		work, err := modfile.ParseWork(workspace, data, nil)
		if err != nil {
			return errors.Join(ErrModule, err)
		}
		for _, use := range work.Use {
			dir := use.Path
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(filepath.Dir(workspace), dir)
			}
			directories = append(directories, dir)
		}
		for _, replacement := range work.Replace {
			if !declared[replacement.Old.Path] {
				continue
			}
			next := solution.ModuleReplacement{Old: solution.Module{Path: replacement.Old.Path, Version: replacement.Old.Version}, New: solution.Module{Path: replacement.New.Path, Version: replacement.New.Version}}
			if next.New.Version == "" && !filepath.IsAbs(next.New.Path) {
				next.New.Path = filepath.Join(filepath.Dir(workspace), next.New.Path)
			}
			setReplacement(module, next)
		}
	}
	if file := enclosingFile(base, "go.mod"); file != "" {
		directories = append(directories, filepath.Dir(file))
	}
	seen := make(map[string]string)
	for _, dir := range directories {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			return err
		}
		manifest, err := modfile.Parse(filepath.Join(dir, "go.mod"), data, nil)
		if err != nil {
			return errors.Join(ErrModule, err)
		}
		if manifest.Module == nil {
			return fmt.Errorf("%w: %s has no module directive", ErrModule, dir)
		}
		path := manifest.Module.Mod.Path
		if !declared[path] {
			continue
		}
		if previous, exists := seen[path]; exists && previous != dir {
			return fmt.Errorf("%w: module %q occurs in multiple workspace directories", ErrModule, path)
		}
		seen[path] = dir
		setReplacement(module, solution.ModuleReplacement{Old: solution.Module{Path: path}, New: solution.Module{Path: dir}})
	}
	return nil
}

func enclosingFile(dir, name string) string {
	for {
		file := filepath.Join(dir, name)
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			return file
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func setReplacement(module *solution.ModuleContext, next solution.ModuleReplacement) {
	// A workspace's module-wide replacement supersedes version-specific ones.
	module.Replace = slices.DeleteFunc(module.Replace, func(previous solution.ModuleReplacement) bool {
		return previous.Old.Path == next.Old.Path && (next.Old.Version == "" || previous.Old.Version == next.Old.Version)
	})
	module.Replace = append(module.Replace, next)
}

// Module graph selection precedes package discovery. A missing catalogue package
// must fail at the selected version instead of discovering a newer release.
func selectedModules(ctx context.Context, work string, environment []string) (map[string]string, error) {
	return moduleGraph(ctx, work, environment, "-mod=readonly")
}

func moduleGraph(ctx context.Context, work string, environment []string, mode string) (map[string]string, error) {
	command := exec.CommandContext(ctx, "go", "list", mode, "-m", "-json", "all")
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
				return selected, nil
			}
			return nil, &Error{Stage: "dependencies", Err: err}
		}
		selected[module.Path] = module.Version
	}
}

func checkCatalogueModules(ctx context.Context, work string, environment []string, definition solution.Definition, module solution.ModuleContext, selected map[string]string) error {
	current, err := selectedModules(ctx, work, environment)
	if err != nil {
		return err
	}
	for path, version := range current {
		previous, present := selected[path]
		if present && version != previous {
			return &Error{Stage: "dependencies", Err: fmt.Errorf("%w: package discovery changed module %q from %q to %q", ErrModule, path, previous, version)}
		}
	}
	declared := make(map[string]bool)
	for _, requirement := range module.Require {
		declared[requirement.Path] = true
	}
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
		return commandError(ctx, "dependencies", errors.Join(ErrModule, err), nil)
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
