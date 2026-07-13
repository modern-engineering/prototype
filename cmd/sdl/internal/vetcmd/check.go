// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package vetcmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/image"
)

// A checker accumulates the findings of one unit set. The section and
// top-level-field checks judge against the checker's captured
// vocabulary — the very value the linker enforces — so those
// memberships cannot drift from the compiler; see the package
// documentation for the drift risk the remaining checks carry.
type checker struct {
	vocab solution.Vocab
	diags scanner.ErrorList
}

// newChecker returns a checker over the linker's body vocabulary.
func newChecker() *checker {
	return &checker{vocab: solution.Vocabulary()}
}

// errorf records one positioned finding.
func (c *checker) errorf(pos token.Position, format string, args ...any) {
	c.diags.Add(pos, fmt.Sprintf(format, args...))
}

// CheckSource parses one unit and returns its findings, sorted: the
// syntax errors when the parse is unclean, otherwise the unit-scope
// checks — the statement bodies and this one file's declarations. The
// unused-symbol check deliberately does not run: a unit is one member
// of its solution's unit set, and a sibling unit may reference a
// symbol this unit declares, so single-file scope cannot judge
// unusedness without false positives. sdl vet runs the set-scope
// checks over each directory it visits; a language server serving
// per-file diagnostics is CheckSource's intended second caller.
func CheckSource(filename string, src []byte) scanner.ErrorList {
	c := newChecker()
	f, err := parser.ParseFile(filename, src)
	if err != nil {
		c.syntax(filename, err)
	} else {
		c.unit(f)
		c.declarations([]*ast.File{f})
	}
	c.diags.Sort()
	return c.diags
}

// syntax records a failed parse's findings: the positioned error list
// exactly as fmt reports it, or the error text anchored to the file
// for a failure that carries no positions.
func (c *checker) syntax(filename string, err error) {
	var list scanner.ErrorList
	if errors.As(err, &list) {
		c.diags = append(c.diags, list...)
		return
	}
	c.diags.Add(token.Position{Filename: filename}, err.Error())
}

// unit runs the single-unit checks: every statement body validates
// against the section vocabulary and its verb's closed top-level
// scheme, mirroring the linker's per-statement route.
func (c *checker) unit(f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.DeployDecl:
			for _, spec := range d.Specs {
				c.body(spec.Body, image.VerbDeploy)
			}
		case *ast.ProvisionDecl:
			for _, spec := range d.Specs {
				c.body(spec.Body, image.VerbProvision)
			}
		case *ast.DefaultDecl:
			// A verb-scoped default names its own top-level scheme; a
			// type-scoped one takes the verb its element's kind
			// implies, which is catalogue knowledge vet does not
			// have, so its top-level fields pass unjudged — the
			// linker's own posture for an unresolved element.
			verb := ""
			if t, isVerb := d.Target.(*ast.Ident); isVerb {
				verb = t.Name
			}
			c.body(d.Body, verb)
		}
	}
}

// body checks one statement body: top-level fields against the verb's
// closed scheme, section heads against the vocabulary and their
// words' policies, and metadata items against the compartment's
// string constraint. The linker's route is the original; with verb
// empty no scheme is known and the top-level fields pass unjudged.
func (c *checker) body(body *ast.Body, verb string) {
	if body == nil {
		return
	}
	seen := make(map[string]token.Position, 3)
	for _, item := range body.Items {
		switch it := item.(type) {
		case *ast.Param:
			c.rootField(it, verb)
		case *ast.Section:
			c.section(it, seen)
		}
	}
}

// rootField checks one top-level field's key against the verb's
// closed scheme. The key check is element-independent — under the
// D-12 anatomy no bare root key can be a catalogue parameter — so
// catalogue-free vet performs it in full; only the linker's teaching
// hint for a misplaced catalogue parameter needs the element and is
// not reproduced. Value shapes stay the linker's call.
func (c *checker) rootField(it *ast.Param, verb string) {
	if verb == "" || slices.Contains(c.vocab.RootFields[verb], it.Key.Name) {
		return
	}
	c.errorf(it.Key.NamePos, "unknown top-level field %s: %s", it.Key.Name, c.rootScheme(verb))
}

// section checks one section: vocabulary membership, the word's
// qualifier policy, per-body dedup, and the metadata items. The
// linker's routeSection is the original, mirrored fault for fault: a
// failed check returns before the next, so a misqualified section
// never claims a dedup slot, and the retired mockup-5 word on keeps
// its teaching migration message ahead of the vocabulary check.
func (c *checker) section(sec *ast.Section, seen map[string]token.Position) {
	name := sec.Name.Name
	switch {
	case name == "on":
		c.errorf(sec.Name.NamePos, "unknown section on: deployment intent moved to top-level fields, controller schemes to with <qualifier> stanzas")
		return
	case !slices.Contains(c.vocab.Sections, name):
		c.errorf(sec.Name.NamePos, "unknown section %s: sections are %s", name, wordList(c.vocab.Sections))
		return
	}
	switch name {
	case "params", "metadata":
		if sec.Qualifier != nil {
			c.errorf(sec.Qualifier.NamePos, "%s takes no qualifier", name)
			return
		}
	case "with":
		if sec.Qualifier == nil {
			c.errorf(sec.Name.NamePos, "with requires a qualifier, e.g. with k8s.pod")
			return
		}
		name += " " + sec.Qualifier.Name
	}
	if first, dup := seen[name]; dup {
		c.errorf(sec.Name.NamePos, "duplicate %s section (first declared at %s)", name, first)
		return
	}
	seen[name] = sec.Name.NamePos

	if sec.Name.Name == "metadata" {
		c.metadata(sec.Body)
	}
}

// metadata checks a metadata section's values: string literals only,
// the compartment's portability constraint. A value the parse already
// faulted stays quiet, and nested sections — the linker's "parameters
// only" fault, outside the v0 list — pass unjudged.
func (c *checker) metadata(body *ast.Body) {
	for _, item := range body.Items {
		it, isParam := item.(*ast.Param)
		if !isParam {
			continue
		}
		switch it.Value.(type) {
		case *ast.StringLit:
		case *ast.BadValue: // the parse already reported it
		default:
			c.errorf(it.Value.Pos(), "metadata values must be string literals")
		}
	}
}

// A declaration is one name's first owner in the unit set's flat
// namespace.
type declaration struct {
	pos    token.Position
	symbol bool // a var or extern symbol: the unused check's population
}

// declarations enters every declared name of every unit — instance
// names, var symbols, extern symbols — into one flat namespace in
// unit-then-statement order, diagnosing each redeclaration at the
// later site, the linker's collect discipline. The first owners come
// back for the unused pass.
func (c *checker) declarations(files []*ast.File) map[string]declaration {
	decls := make(map[string]declaration)
	declare := func(name *ast.Ident, symbol bool) {
		if first, dup := decls[name.Name]; dup {
			c.errorf(name.NamePos, "duplicate symbol %s (first declared at %s)", name.Name, first.pos)
			return
		}
		decls[name.Name] = declaration{pos: name.NamePos, symbol: symbol}
	}
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.DeployDecl:
				for _, spec := range d.Specs {
					declare(spec.Name, false)
				}
			case *ast.ProvisionDecl:
				for _, spec := range d.Specs {
					declare(spec.Name, false)
				}
			case *ast.ExternDecl:
				for _, spec := range d.Specs {
					declare(spec.Name, true)
				}
			case *ast.VarDecl:
				for _, spec := range d.Specs {
					declare(spec.Name, true)
				}
			}
		}
	}
	return decls
}

// unused reports the var and extern symbols no unit of the set
// references. The build accepts them — the image pins every declared
// symbol — which is exactly the mistake: an unused extern makes every
// deployment site bind a value nothing reads. A reference is a
// params-section value only, bare or the head of a dotted output
// reference; in the deployment, with, and metadata compartments a
// bare identifier is an opaque token or a fault, never a symbol
// reference, so a token spelling a symbol's name marks nothing used.
func (c *checker) unused(files []*ast.File, decls map[string]declaration) {
	used := make(map[string]bool)
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.DeployDecl:
				for _, spec := range d.Specs {
					markUses(used, spec.Body)
				}
			case *ast.ProvisionDecl:
				for _, spec := range d.Specs {
					markUses(used, spec.Body)
				}
			case *ast.DefaultDecl:
				markUses(used, d.Body)
			}
		}
	}
	for name, d := range decls {
		if d.symbol && !used[name] {
			c.errorf(d.pos, "symbol %s declared and not used", name)
		}
	}
}

// markUses records the symbols a body's params sections reference.
// Every section literally named params counts, qualified or
// duplicated or not: those faults carry their own findings, and
// counting the uses anyway keeps a misshapen section from stacking
// bogus unused findings on top of them.
func markUses(used map[string]bool, body *ast.Body) {
	if body == nil {
		return
	}
	for _, item := range body.Items {
		sec, isSection := item.(*ast.Section)
		if !isSection || sec.Name.Name != "params" {
			continue
		}
		for _, pitem := range sec.Body.Items {
			if it, isParam := pitem.(*ast.Param); isParam {
				if ref, isRef := it.Value.(*ast.RefExpr); isRef {
					used[ref.X.Name] = true
				}
			}
		}
	}
}

// rootScheme renders a verb's top-level scheme the way the linker's
// diagnostics teach it, straight from the vocabulary's sorted field
// lists.
func (c *checker) rootScheme(verb string) string {
	keys := c.vocab.RootFields[verb]
	if len(keys) == 0 {
		return verb + " takes no top-level fields"
	}
	return verb + " takes " + strings.Join(keys, ", ")
}

// wordList renders a word enumeration the way the linker's
// diagnostics teach a closed vocabulary: "a", "a and b", "a, b, and
// c".
func wordList(words []string) string {
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	case 2:
		return words[0] + " and " + words[1]
	}
	return strings.Join(words[:len(words)-1], ", ") + ", and " + words[len(words)-1]
}
