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
	"os"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/cmd/sdl/internal/load"
)

// descriptorPath is the package whose Descriptor type marks a catalogue
// citizen.
const descriptorPath = "github.com/modern-engineering/prototype/application"

// A Package is one discovered catalogue package: its import path, its Go
// package name, and the exported identifiers of its citizens.
type Package struct {
	Path string
	Name string

	// Citizens are the exported package-level vars of type
	// *application.Descriptor, in types.Scope.Names order (sorted).
	// A package may have none; it is registered empty, and the linker
	// reports any reference into it as an unknown element.
	Citizens []string
}

// Discover resolves the solution's imports in dir's module context and
// scans each package's type surface for catalogue citizens. Packages
// that fail to load become positioned diagnostics at their import specs
// (a *base.DiagnosticsError); citizen-less packages and value-typed
// descriptor vars are warnings on warn, not errors. The result is
// sorted by path.
//
// The module context matches the build driver's: GOWORK=off (workspace
// mode is a later rung) and an emptied GOFLAGS, so discovery and the
// generated compiler resolve packages identically.
func Discover(dir string, imports []load.Import, warn io.Writer) ([]Package, error) {
	if len(imports) == 0 {
		return nil, nil
	}
	patterns := make([]string, len(imports))
	specs := make(map[string]load.Import, len(imports))
	for i, imp := range imports {
		patterns[i] = imp.Path
		specs[imp.Path] = imp
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
		Dir:  dir,
		Env:  append(os.Environ(), "GOWORK=off", "GOFLAGS="),
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
// a pointer-to-Descriptor var is a citizen; a value-typed Descriptor var
// earns a warning naming the pointer-style fix; a package with no
// citizens at all earns a warning and an empty registration.
func scan(pkg *packages.Package, warn io.Writer) Package {
	scope := pkg.Types.Scope()
	var citizens []string
	for _, name := range scope.Names() { // Names is sorted
		v, ok := scope.Lookup(name).(*types.Var)
		if !ok || !v.Exported() {
			continue
		}
		t := types.Unalias(v.Type())
		if ptr, ok := t.(*types.Pointer); ok {
			if isDescriptor(types.Unalias(ptr.Elem())) {
				citizens = append(citizens, name)
			}
			continue
		}
		if isDescriptor(t) {
			fmt.Fprintf(warn, "sdl: package %q: var %s is an application.Descriptor value; declare it as a pointer (var %s = &application.Descriptor{...}) to register it\n",
				pkg.PkgPath, name, name)
		}
	}
	if len(citizens) == 0 {
		fmt.Fprintf(warn, "sdl: package %q exports no catalogue elements\n", pkg.PkgPath)
	}
	return Package{Path: pkg.PkgPath, Name: pkg.Types.Name(), Citizens: citizens}
}

// isDescriptor reports whether t is the named type application.Descriptor.
func isDescriptor(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Name() == "Descriptor" &&
		obj.Pkg() != nil && obj.Pkg().Path() == descriptorPath
}
