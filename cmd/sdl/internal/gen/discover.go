// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package gen turns a loaded solution into the source of its generated
// compiler: discovery resolves the imported Go packages and finds their
// catalogue citizens, and Source emits the package main that embeds the
// units and registers the catalogue for solution.MainCompile.
package gen

import (
	"fmt"
	"go/types"
	"io"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/load"
)

// The packages whose named types mark catalogue citizens.
const (
	descriptorPath = "github.com/modern-engineering/prototype/application"
	solutionPath   = "github.com/modern-engineering/prototype/solution"
)

// The element kinds discovery recognizes, keyed by the pointee type of
// the exported var.
const (
	KindComponent = "component" // *application.Descriptor, packaged by solution.App
	KindProvision = "provision" // *solution.ProvisionType, packaged by solution.Provision
	KindSymbol    = "symbol"    // *solution.SymbolType, packaged by solution.Symbol
)

// A Citizen is one discovered catalogue element: the exported var's Go
// identifier — the reference name solutions link against — and the
// element kind its type selects.
type Citizen struct {
	Name string
	Kind string
}

// A Package is one discovered catalogue package: its import path, its Go
// package name, and its citizens.
type Package struct {
	Path string
	Name string

	// Citizens are the exported package-level vars whose pointee type
	// marks an element kind, in types.Scope.Names order (sorted). A
	// package may have none; it is registered empty, and the linker
	// reports any reference into it as an unknown element.
	Citizens []Citizen
}

// Discover resolves the solution's imports in dir's module context and
// scans each package's type surface for catalogue citizens. Packages
// that fail to load become positioned diagnostics at their import specs
// (a *base.DiagnosticsError); citizen-less packages and value-typed
// descriptor vars are warnings on warn, not errors. The result is
// sorted by path.
//
// Internal packages are rejected up front, positioned at their import
// specs: the generated compiler builds as its own synthesized module
// (sdl.invalid/solmain), outside every internal boundary, so the Go
// toolchain would refuse the import anyway — the diagnostic here names
// the limitation instead of relaying a build failure. A later rung
// could lift it by synthesizing the work module under the solution
// module's own path; that is a recorded door, not a grammar gap.
//
// The env is the module-context environment of the enclosing build
// (work.Context.Env), so discovery and the generated compiler resolve
// packages identically in every module mode — workspace mode included.
// Loading passes -mod=readonly explicitly, so a vendored solution
// module resolves through its module graph exactly as the synthesized
// build will (the work module carries no vendor tree).
func Discover(dir string, imports []load.Import, env []string, warn io.Writer) ([]Package, error) {
	if len(imports) == 0 {
		return nil, nil
	}
	patterns := make([]string, len(imports))
	specs := make(map[string]load.Import, len(imports))
	var internal []string
	for i, imp := range imports {
		patterns[i] = imp.Path
		specs[imp.Path] = imp
		if internalPath(imp.Path) {
			internal = append(internal, fmt.Sprintf(
				"%s: import %q: internal package: the generated compiler builds outside the package's internal boundary and cannot import it; export the catalogue package",
				imp.Pos, imp.Path))
		}
	}
	if len(internal) > 0 {
		return nil, &base.DiagnosticsError{Lines: internal}
	}
	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedTypes,
		Dir:        dir,
		Env:        env,
		BuildFlags: []string{"-mod=readonly"},
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("loading catalogue packages: %v", err)
	}

	var diags []string
	var out []Package
	for _, pkg := range pkgs {
		imp, known := specs[pkg.PkgPath]
		if !known {
			imp, known = specs[pkg.ID]
		}
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				// One diagnostic, one line: go's errors may span
				// lines ("to add it:\n\tgo get ..."), so fold the
				// message's whitespace runs.
				msg := strings.Join(strings.Fields(e.Msg), " ")
				if known {
					diags = append(diags, fmt.Sprintf("%s: import %q: %s", imp.Pos, imp.Path, msg))
				} else {
					diags = append(diags, fmt.Sprintf("import %q: %s", pkg.PkgPath, msg))
				}
			}
			continue
		}
		if pkg.Types == nil {
			// Loading reported no error yet produced no type
			// information: a driver fault, not a solution fault.
			return nil, fmt.Errorf("loading catalogue packages: no type information for %q", pkg.PkgPath)
		}
		out = append(out, scan(pkg, warn))
	}
	if len(diags) > 0 {
		return nil, &base.DiagnosticsError{Lines: diags}
	}
	slices.SortFunc(out, func(a, b Package) int {
		return strings.Compare(a.Path, b.Path)
	})
	return out, nil
}

// scan reads one loaded package's exported vars into its registration:
// a pointer var whose pointee marks an element kind is a citizen; a
// value-typed var of such a type earns a warning naming the
// pointer-style fix; a package with no citizens at all earns a warning
// and an empty registration.
func scan(pkg *packages.Package, warn io.Writer) Package {
	scope := pkg.Types.Scope()
	var citizens []Citizen
	for _, name := range scope.Names() { // Names is sorted
		v, ok := scope.Lookup(name).(*types.Var)
		if !ok || !v.Exported() {
			continue
		}
		t := types.Unalias(v.Type())
		if ptr, ok := t.(*types.Pointer); ok {
			if kind, _, ok := citizenKind(types.Unalias(ptr.Elem())); ok {
				citizens = append(citizens, Citizen{Name: name, Kind: kind})
			}
			continue
		}
		if _, typeName, ok := citizenKind(t); ok {
			printf(warn, "sdl: package %q: var %s is a value of type %s; declare it as a pointer (var %s = &%s{...}) to register it\n",
				pkg.PkgPath, name, typeName, name, typeName)
		}
	}
	if len(citizens) == 0 {
		printf(warn, "sdl: package %q exports no catalogue elements\n", pkg.PkgPath)
	}
	return Package{Path: pkg.PkgPath, Name: pkg.Types.Name(), Citizens: citizens}
}

// printf writes one warning line. The write is best effort: a warning
// writer that fails has nowhere better to hear about it, and warnings
// never change the discovery result, so the write error is deliberately
// discarded — here, once, rather than at every call site.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// citizenKind classifies a candidate pointee type, returning the
// element kind it marks and the qualified type name for messages.
func citizenKind(t types.Type) (kind, typeName string, ok bool) {
	switch {
	case isNamed(t, descriptorPath, "Descriptor"):
		return KindComponent, "application.Descriptor", true
	case isNamed(t, solutionPath, "ProvisionType"):
		return KindProvision, "solution.ProvisionType", true
	case isNamed(t, solutionPath, "SymbolType"):
		return KindSymbol, "solution.SymbolType", true
	}
	return "", "", false
}

// internalPath reports whether path lies inside an internal directory
// — the Go import-visibility boundary, cmd/go's own element rule.
func internalPath(path string) bool {
	return path == "internal" ||
		strings.HasPrefix(path, "internal/") ||
		strings.HasSuffix(path, "/internal") ||
		strings.Contains(path, "/internal/")
}

// isNamed reports whether t is the named type path.name.
func isNamed(t types.Type, path, name string) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Name() == name &&
		obj.Pkg() != nil && obj.Pkg().Path() == path
}
