// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package load reads a solution directory into the inputs of the build
// pipeline: the unit sources, the solution name, and the union import
// table. It runs the checks that must fail fast before the Go toolchain
// is invoked — syntax and import-table conflicts — and nothing more:
// linking semantics (solution-clause agreement, reference resolution)
// belong to solution.MainCompile inside the generated compiler.
package load

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
)

// A Unit is one solution unit: its base filename and full source text.
type Unit struct {
	Name   string
	Source string
}

// An Import is one imported package path together with the position of
// the first import spec naming it; package-load failures are reported
// there.
type Import struct {
	Path string
	Pos  token.Position
}

// A Solution is a loaded solution directory, ready for discovery and
// code generation.
type Solution struct {
	// Name is the solution name declared by the first unit's solution
	// clause. Whether the other units agree is the linker's business.
	Name string

	// Dir is the absolute solution directory.
	Dir string

	// Units are the solution's units, sorted by filename.
	Units []Unit

	// Imports are the distinct imported package paths, sorted by path.
	Imports []Import
}

// Dir loads the solution in dir (empty means the current directory).
// All units are parsed and every fault is collected before failing:
// syntax errors, import paths that cannot be package-load patterns, and
// one import name bound to two different paths all come back together
// in one *base.DiagnosticsError.
func Dir(dir string) (*Solution, error) {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %v", dir, err)
	}
	names, err := unitNames(abs)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, &base.DiagnosticsError{Lines: []string{fmt.Sprintf("no .sdl files in %s", abs)}}
	}

	sol := &Solution{Dir: abs}
	var diags scanner.ErrorList
	imports := make(map[string]importBinding) // reference name -> binding
	paths := make(map[string]token.Position)  // import path -> first spec
	for _, name := range names {
		src, err := os.ReadFile(filepath.Join(abs, name))
		if err != nil {
			return nil, fmt.Errorf("reading unit: %v", err)
		}
		sol.Units = append(sol.Units, Unit{Name: name, Source: string(src)})

		f, err := parser.ParseFile(name, src)
		if err != nil {
			var list scanner.ErrorList
			if errors.As(err, &list) {
				diags = append(diags, list...)
			} else {
				diags.Add(token.Position{Filename: name}, err.Error())
			}
		}
		if sol.Name == "" && f.Solution != nil {
			sol.Name = f.Solution.Name.Name
		}
		for _, decl := range f.Imports {
			for _, spec := range decl.Specs {
				addImport(&diags, imports, paths, spec)
			}
		}
	}
	if len(diags) > 0 {
		diags.Sort()
		lines := make([]string, len(diags))
		for i, e := range diags {
			lines[i] = e.Error()
		}
		return nil, &base.DiagnosticsError{Lines: lines}
	}

	for path, pos := range paths {
		sol.Imports = append(sol.Imports, Import{Path: path, Pos: pos})
	}
	// Sorted paths keep discovery, code generation, and the emitted
	// catalogue deterministic.
	slices.SortFunc(sol.Imports, func(a, b Import) int {
		return strings.Compare(a.Path, b.Path)
	})
	return sol, nil
}

// unitNames lists the .sdl files of dir, sorted by filename (the unit
// order of the whole pipeline).
func unitNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading solution directory: %v", err)
	}
	var names []string // ReadDir returns entries sorted by filename
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sdl") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// An importBinding is one entry of the union import table.
type importBinding struct {
	path string
	pos  token.Position
}

// addImport records one import spec: the path joins the load set, and an
// aliased spec binds its alias in the union table, where one name bound
// to two different paths is a conflict reported at the later spec.
// Unaliased specs bind the imported package's own name, which only
// discovery can supply; their conflicts are the linker's to find.
func addImport(diags *scanner.ErrorList, imports map[string]importBinding, paths map[string]token.Position, spec *ast.ImportSpec) {
	path, pos := spec.Path.Value, spec.Pos()
	if err := checkImportPath(path); err != nil {
		diags.Add(pos, fmt.Sprintf("import %q: %v", path, err))
		return
	}
	if _, ok := paths[path]; !ok {
		paths[path] = pos
	}
	if spec.Alias == nil {
		return
	}
	alias := spec.Alias.Name
	if prev, ok := imports[alias]; ok {
		if prev.path != path {
			diags.Add(pos, fmt.Sprintf("import name %s already bound to %q (first imported at %s)", alias, prev.path, prev.pos))
		}
		return
	}
	imports[alias] = importBinding{path: path, pos: pos}
}

// checkImportPath rejects strings that cannot be a single package's
// import path before they reach the Go toolchain as load patterns:
// wildcards, meta-patterns, and shell-hostile shapes would either fan
// out to several packages or be misread as something other than a path.
func checkImportPath(path string) error {
	switch {
	case path == "":
		return errors.New("empty import path")
	case path == "all" || path == "std" || path == "cmd":
		return errors.New("not a package path")
	case strings.Contains(path, "..."):
		return errors.New("import path must not contain wildcards")
	case strings.HasPrefix(path, "-"):
		return errors.New("import path must not begin with -")
	case strings.ContainsAny(path, " \t\r\n\"'`"):
		return errors.New("import path contains invalid characters")
	}
	return nil
}
