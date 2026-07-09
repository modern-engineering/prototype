// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package echocmd implements sdl echo, the read side of the image
// plumbing: it renders a desired-state image back as one canonical
// solution unit, the IR made visible. Echo is canonical re-rendering,
// never a byte round trip of the original sources: the image stores
// values, not lexemes or layout, so every literal prints in its
// canonical spelling — where sdl fmt would preserve "0x10", echo has
// only the 16. Building the echoed unit yields an image [image.Equal]
// to the input; that equivalence is the round-trip contract, proven end
// to end in the cmd/sdl tests.
package echocmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/printer"
	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution/image"
)

// CmdEcho is the sdl echo command.
var CmdEcho = &base.Command{
	UsageLine: "sdl echo [image]",
	Short:     "render a desired-state image as canonical SDL",
	Long: `Echo reads a desired-state image — from the named file, or from
standard input when no file is given — and prints it to standard output
as one canonical solution unit: the solution clause, one import per
catalogue package sorted by path, the symbol table as extern and var
declarations in image order, and one single-form deploy or provision
statement per record in image order, parameters included. A provision
statement always writes its kind word — the compiler resolved any
omission, and the canonical form keeps the record explicit.

The unit is a re-rendering, not the original sources: images store
canonical values rather than the author's lexemes or layout, so echo
prints every value in its canonical spelling (strings quoted like Go,
integers in decimal, durations as Go renders them, booleans as true or
false, symbol references as bare identifiers, output references as
instance.output pairs) and emits nothing the image does not carry.
Building the echoed unit against the same catalogue reproduces an
equal image.

Imports are aliased so references resolve exactly as compiled: the
reference name is the package's registered name, spelled out when the
import path's last element would not suggest it, and disambiguated with
a numeric suffix (name, name2, ...) when several packages share one
name.

Exit status 0 means the unit was printed; 2 reports an unreadable or
undecodable image, or one that this rung cannot render.`,
}

func init() {
	CmdEcho.Run = runEcho
}

func runEcho(ctx context.Context, cmd *base.Command, args []string) error {
	var in io.Reader = os.Stdin
	switch len(args) {
	case 0:
	case 1:
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		// The image is only read; a close fault has nothing to add to
		// the decode's own verdict.
		defer func() { _ = f.Close() }()
		in = f
	default:
		return &base.UsageError{Msg: fmt.Sprintf("echo takes at most one image argument, got %d", len(args))}
	}
	return echo(in, os.Stdout)
}

// echo decodes one image and prints its canonical unit. Every error is
// a fault in the image or in the streams — never in a solution's text —
// so callers surface them all as exit code 2.
func echo(r io.Reader, w io.Writer) error {
	img, err := image.Decode(r)
	if err != nil {
		return err
	}
	f, err := reconstruct(img)
	if err != nil {
		return err
	}
	return printer.Fprint(w, f)
}

// reconstruct builds the canonical unit for an image: the solution
// clause, the import block, the symbol table as an extern and a var
// declaration, and one single-form deploy or provision declaration per
// record. Factoring within a declaration follows the printer's rule —
// one spec prints single-form, several print as one factored block —
// since the image records no authoring layout.
func reconstruct(img *image.Image) (*ast.File, error) {
	if err := checkIdent("solution name", img.Solution); err != nil {
		return nil, err
	}
	f := &ast.File{Solution: &ast.SolutionClause{Name: &ast.Ident{Name: img.Solution}}}

	refs, specs, err := imports(img.Catalogue)
	if err != nil {
		return nil, err
	}
	if len(specs) > 0 {
		f.Imports = []*ast.ImportDecl{{Specs: specs}}
	}

	externs, vars, err := symbols(img.Symbols, refs)
	if err != nil {
		return nil, err
	}
	if externs != nil {
		f.Decls = append(f.Decls, externs)
	}
	if vars != nil {
		f.Decls = append(f.Decls, vars)
	}

	for i, rec := range img.Records {
		var decl ast.Decl
		var err error
		switch rec.Verb {
		case image.VerbDeploy:
			decl, err = deploy(rec, refs)
		case image.VerbProvision:
			decl, err = provision(rec, refs)
		default:
			err = fmt.Errorf("cannot render verb %q", rec.Verb)
		}
		if err != nil {
			return nil, fmt.Errorf("record %d (%s): %w", i, rec.Name, err)
		}
		f.Decls = append(f.Decls, decl)
	}
	return f, nil
}

// symbols renders the image's symbol table: the externs as one extern
// declaration and the vars as one var declaration, each in image
// (name) order, extern block first. Extern types print qualified
// through the same reference names the records resolve by.
func symbols(defs []image.SymbolDef, refs map[string]string) (externs *ast.ExternDecl, vars *ast.VarDecl, err error) {
	for _, def := range defs {
		if err := checkIdent(fmt.Sprintf("symbol name %q", def.Name), def.Name); err != nil {
			return nil, nil, err
		}
		switch def.Class {
		case image.ClassExtern:
			if def.Type == nil {
				return nil, nil, fmt.Errorf("extern symbol %s has no type", def.Name)
			}
			ref, ok := refs[def.Type.Package]
			if !ok {
				return nil, nil, fmt.Errorf("symbol %s: type package %q is not pinned in the catalogue", def.Name, def.Type.Package)
			}
			if err := checkIdent(fmt.Sprintf("symbol %s: type name %q", def.Name, def.Type.Name), def.Type.Name); err != nil {
				return nil, nil, err
			}
			if externs == nil {
				externs = new(ast.ExternDecl)
			}
			externs.Specs = append(externs.Specs, &ast.ExternSpec{
				Name: &ast.Ident{Name: def.Name},
				Type: &ast.TypeRef{Pkg: &ast.Ident{Name: ref}, Name: &ast.Ident{Name: def.Type.Name}},
			})
		case image.ClassVar:
			value, err := valueNode(def.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("var symbol %s: %w", def.Name, err)
			}
			if vars == nil {
				vars = new(ast.VarDecl)
			}
			vars.Specs = append(vars.Specs, &ast.VarSpec{
				Name:  &ast.Ident{Name: def.Name},
				Value: value,
			})
		default:
			return nil, nil, fmt.Errorf("symbol %s: cannot render class %q", def.Name, def.Class)
		}
	}
	return externs, vars, nil
}

// imports derives one import spec per catalogue package, in catalogue
// (path) order, and the reference-name table the records resolve
// through. The reference name is the package's registered Go name; when
// several packages share a name, later ones (by path order) take a
// deterministic numeric suffix. The alias is written out whenever the
// unit would otherwise mislead: when the name was disambiguated, or
// when the path's last element differs from it.
func imports(catalogue []image.Package) (refs map[string]string, specs []*ast.ImportSpec, err error) {
	refs = make(map[string]string, len(catalogue))
	used := make(map[string]bool, len(catalogue))
	for _, pkg := range catalogue {
		if err := checkIdent(fmt.Sprintf("package %q: name %q", pkg.Path, pkg.Name), pkg.Name); err != nil {
			return nil, nil, err
		}
		ref := pkg.Name
		for n := 2; used[ref]; n++ {
			ref = pkg.Name + strconv.Itoa(n)
		}
		used[ref] = true
		refs[pkg.Path] = ref

		spec := &ast.ImportSpec{Path: &ast.StringLit{Value: pkg.Path}}
		if ref != pkg.Name || ref != lastElement(pkg.Path) {
			spec.Alias = &ast.Ident{Name: ref}
		}
		specs = append(specs, spec)
	}
	return refs, specs, nil
}

// lastElement returns the final path element, the name a reader would
// guess a package by.
func lastElement(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// deploy renders one record as a single-form deploy declaration.
func deploy(rec image.Record, refs map[string]string) (*ast.DeployDecl, error) {
	if rec.Kind != "" {
		return nil, fmt.Errorf("deploy record carries provision kind %q", rec.Kind)
	}
	typeRef, name, body, err := recordSpec(rec, refs)
	if err != nil {
		return nil, err
	}
	spec := &ast.DeploySpec{Type: typeRef, Name: name, Body: body}
	return &ast.DeployDecl{Specs: []*ast.DeploySpec{spec}}, nil
}

// provision renders one record as a single-form provision declaration.
// The kind word is always written out: the compiler resolved any
// omission at build time, and the canonical unit keeps the record
// explicit rather than re-deriving what a fresh catalogue might
// resolve differently.
func provision(rec image.Record, refs map[string]string) (*ast.ProvisionDecl, error) {
	if rec.Kind != image.KindSlice && rec.Kind != image.KindAttach {
		return nil, fmt.Errorf("cannot render provision kind %q", rec.Kind)
	}
	typeRef, name, body, err := recordSpec(rec, refs)
	if err != nil {
		return nil, err
	}
	spec := &ast.ProvisionSpec{Type: typeRef, Kind: &ast.Ident{Name: rec.Kind}, Name: name, Body: body}
	return &ast.ProvisionDecl{Specs: []*ast.ProvisionSpec{spec}}, nil
}

// recordSpec renders the spec material shared by both record verbs:
// the qualified type reference, the instance name, and the parameter
// body (nil for a parameterless record).
func recordSpec(rec image.Record, refs map[string]string) (*ast.TypeRef, *ast.Ident, *ast.Body, error) {
	ref, ok := refs[rec.Element.Package]
	if !ok {
		return nil, nil, nil, fmt.Errorf("element package %q is not pinned in the catalogue", rec.Element.Package)
	}
	if err := checkIdent(fmt.Sprintf("element name %q", rec.Element.Name), rec.Element.Name); err != nil {
		return nil, nil, nil, err
	}
	if err := checkIdent(fmt.Sprintf("instance name %q", rec.Name), rec.Name); err != nil {
		return nil, nil, nil, err
	}
	var body *ast.Body
	if len(rec.Params) > 0 {
		body = new(ast.Body)
		for _, b := range rec.Params {
			if err := checkIdent(fmt.Sprintf("parameter key %q", b.Key), b.Key); err != nil {
				return nil, nil, nil, err
			}
			value, err := bindingValue(b)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("parameter %s: %w", b.Key, err)
			}
			body.Items = append(body.Items, &ast.Param{
				Key:   &ast.Ident{Name: b.Key},
				Value: value,
			})
		}
	}
	typeRef := &ast.TypeRef{Pkg: &ast.Ident{Name: ref}, Name: &ast.Ident{Name: rec.Element.Name}}
	return typeRef, &ast.Ident{Name: rec.Name}, body, nil
}

// bindingValue lifts one binding's payload into a value node: a symbol
// reference prints as its bare identifier, an output reference as its
// instance.output pair, a literal in its canonical spelling.
func bindingValue(b image.Binding) (ast.Value, error) {
	if b.Ref != nil {
		if b.Value != nil {
			return nil, fmt.Errorf("carries both a value and a reference")
		}
		if err := checkIdent(fmt.Sprintf("referenced symbol %q", b.Ref.Symbol), b.Ref.Symbol); err != nil {
			return nil, err
		}
		ref := &ast.RefExpr{X: &ast.Ident{Name: b.Ref.Symbol}}
		if b.Ref.Output != "" {
			if err := checkIdent(fmt.Sprintf("referenced output %q", b.Ref.Output), b.Ref.Output); err != nil {
				return nil, err
			}
			ref.Sel = &ast.Ident{Name: b.Ref.Output}
		}
		return ref, nil
	}
	return valueNode(b.Value)
}

// valueNode lifts an image value into a lexeme-free literal node; the
// printer renders those canonically (see sdl/printer).
func valueNode(v *image.Value) (ast.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("binding has no value")
	}
	switch v.Kind {
	case image.KindString:
		return &ast.StringLit{Value: v.Str}, nil
	case image.KindInt:
		return &ast.IntLit{Value: v.Int}, nil
	case image.KindBool:
		return &ast.BoolLit{Value: v.Bool}, nil
	case image.KindDuration:
		return &ast.DurationLit{Value: v.Dur}, nil
	}
	return nil, fmt.Errorf("cannot render value kind %q", v.Kind)
}

// checkIdent verifies that a name from the image can be spelled as an
// SDL identifier — the scanner's identifier rule minus the keywords.
// Names beyond it (a flag named "log-level", say) have no SDL rendering
// yet, and a loud fault beats printing a unit that cannot parse.
func checkIdent(what, name string) error {
	ok := name != "" && token.Lookup(name) == token.IDENT
	for i, r := range name {
		letter := unicode.IsLetter(r) || r == '_'
		if !letter && (i == 0 || !unicode.IsDigit(r)) {
			ok = false
		}
	}
	if !ok {
		return fmt.Errorf("%s: cannot render as an SDL identifier", what)
	}
	return nil
}
