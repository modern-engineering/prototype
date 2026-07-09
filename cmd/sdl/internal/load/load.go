// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package load reads a solution directory into the inputs of the build
// pipeline: the unit sources, the solution name, and the set of
// imported package paths. It runs the checks that must fail fast
// before the Go toolchain is invoked — syntax and import-path shape —
// and nothing more: linking semantics (solution-clause agreement,
// reference resolution, per-unit import scope) belong to
// solution.MainCompile inside the generated compiler.
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
	// clause; a loaded solution always has one, since a clause-less
	// unit is a load diagnostic. Whether the other units agree is the
	// linker's business.
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
// syntax errors and import paths that cannot be package-load patterns
// come back together in one *base.DiagnosticsError. Import names are
// not checked at all — reference names scope to their declaring unit,
// so binding them is the linker's business; the load set is the union
// of the units' import paths, deduplicated, which is exactly what
// discovery and code generation consume.
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
	paths := make(map[string]token.Position) // import path -> first spec
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
			// A unit that failed to parse may carry a partial AST — an
			// "import ff" with no path leaves an ImportSpec whose Path
			// is nil — so its import walk is skipped, mirroring the
			// generated compiler, where syntax gates linking. The skip
			// is per unit, not per load: the parse diagnostics already
			// position every fault in this file, and the healthy units'
			// import checks keep flowing so one broken unit does not
			// hide another's conflicts.
			continue
		}
		if f.Solution == nil {
			// Only an empty or comment-only unit parses without a
			// clause (the parser demands one ahead of any declaration).
			// Diagnosing it here keeps build and fmt agreeing that the
			// unit is not well-formed, and keeps the fault the
			// author's: without this, a clause-less lone unit would
			// surface as the generated compiler's config error, exit 2.
			diags.Add(token.Position{Filename: name, Line: 1, Column: 1}, "unit declares no solution clause")
		} else if sol.Name == "" {
			sol.Name = f.Solution.Name.Name
		}
		for _, decl := range f.Imports {
			for _, spec := range decl.Specs {
				addImport(&diags, paths, spec)
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

// addImport records one import spec: the path joins the load set,
// deduplicated by path with the first spec keeping the position. The
// spec's reference name is not load's business — import scope is the
// unit, so the same alias may name different packages in different
// units, and only the linker sees the units one table at a time.
func addImport(diags *scanner.ErrorList, paths map[string]token.Position, spec *ast.ImportSpec) {
	path, pos := spec.Path.Value, spec.Pos()
	if err := checkImportPath(path); err != nil {
		diags.Add(pos, fmt.Sprintf("import %q: %v", path, err))
		return
	}
	if _, ok := paths[path]; !ok {
		paths[path] = pos
	}
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
