// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package printer

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/token"
)

// Fprint writes the canonical form of f to w.
func Fprint(w io.Writer, f *ast.File) error {
	src, err := Source(f)
	if err != nil {
		return err
	}
	_, err = w.Write(src)
	return err
}

// Source returns the canonical form of f. The error reports a
// structural hole in the tree — a missing solution clause, a
// declaration without specs, a [ast.BadValue] — never a formatting
// choice; a tree that parsed cleanly always prints.
func Source(f *ast.File) ([]byte, error) {
	var p printer
	if err := p.file(f); err != nil {
		return nil, err
	}
	return p.buf.Bytes(), nil
}

// A printer accumulates the canonical output. Blank lines are decided
// one item ahead: every printed item — a comment group or a statement
// line — first settles whether a blank line separates it from the item
// before, from the source line delta when positions are known and from
// the force and snug switches at section and block boundaries.
type printer struct {
	buf   bytes.Buffer
	depth int // indentation depth, one tab per level

	printed bool // some line has been written
	lastEnd int  // source line where the previous item ended; 0 unknown
	force   bool // the next item is separated by exactly one blank line
	snug    bool // the next item follows its block opener directly
}

// writeLine writes one line at the current indentation.
func (p *printer) writeLine(text string) {
	for range p.depth {
		p.buf.WriteByte('\t')
	}
	p.buf.WriteString(text)
	p.buf.WriteByte('\n')
	p.printed = true
}

// startItem decides the blank line owed before an item whose first
// token sits on the given source line. Unknown lines (zero) fall back
// to blankWhenUnknown, the synthesized-tree policy of the caller.
func (p *printer) startItem(line int, blankWhenUnknown bool) {
	force, snug := p.force, p.snug
	p.force, p.snug = false, false
	switch {
	case !p.printed || snug:
		return
	case force:
		// owed exactly one blank line
	case line > 0 && p.lastEnd > 0:
		if line-p.lastEnd < 2 {
			return
		}
	case !blankWhenUnknown:
		return
	}
	p.buf.WriteByte('\n')
}

// comments prints full-line comment groups and reports whether any line
// was written; the blank-when-unknown policy applies to the first group
// only, later groups separate by their own source gaps.
func (p *printer) comments(groups []*ast.CommentGroup, blankWhenUnknown bool) bool {
	printed := false
	for _, g := range groups {
		if g == nil || len(g.List) == 0 {
			continue
		}
		p.startItem(g.List[0].Slash.Line, blankWhenUnknown)
		blankWhenUnknown = false
		for _, c := range g.List {
			p.writeLine(c.Text)
		}
		p.lastEnd = g.List[len(g.List)-1].Slash.Line
		printed = true
	}
	return printed
}

// suffix renders a trailing comment for the end of a line.
func suffix(c *ast.Comment) string {
	if c == nil {
		return ""
	}
	return " " + c.Text
}

// splitSuffix places a multi-line statement's trailing comment: on the
// opening line when the author wrote it beside the opener, on the
// closing line otherwise (and always there when open and close share a
// line, or when the comment carries no position).
func splitSuffix(c *ast.Comment, open, close int) (onOpen, onClose string) {
	if c == nil {
		return "", ""
	}
	if c.Slash.Line > 0 && c.Slash.Line == open && open != close {
		return suffix(c), ""
	}
	return "", suffix(c)
}

// stmt prints one statement: its Before groups, the head line, and the
// body when there is one. start is the source line of the statement's
// first token; top selects the synthesized-tree blank policy for
// top-level statements.
func (p *printer) stmt(c *ast.Comments, head string, body *ast.Body, start int, top bool) error {
	unknown := top
	if p.comments(c.Before, unknown) {
		unknown = false
	}
	p.startItem(start, unknown)
	if body == nil {
		p.writeLine(head + suffix(c.Suffix))
		p.lastEnd = start
		return nil
	}
	open, cls := splitSuffix(c.Suffix, body.Lbrace.Line, body.Rbrace.Line)
	if len(body.Items) == 0 && len(body.After) == 0 {
		p.writeLine(head + " {}" + open + cls)
		p.lastEnd = body.Rbrace.Line
		return nil
	}
	p.writeLine(head + " {" + open)
	p.depth++
	p.lastEnd = body.Lbrace.Line
	p.snug = true
	if err := p.bodyItems(body); err != nil {
		return err
	}
	p.comments(body.After, false)
	p.depth--
	p.snug = false
	p.writeLine("}" + cls)
	p.lastEnd = body.Rbrace.Line
	return nil
}

// bodyItems prints a body's parameters and sections in order.
func (p *printer) bodyItems(b *ast.Body) error {
	for _, item := range b.Items {
		switch it := item.(type) {
		case *ast.Param:
			if it.Key == nil {
				return fmt.Errorf("printer: parameter has no key")
			}
			text, err := valueText(it.Value)
			if err != nil {
				return err
			}
			if err := p.stmt(&it.Comments, it.Key.Name+": "+text, nil, it.Key.NamePos.Line, false); err != nil {
				return err
			}
		case *ast.Section:
			if it.Name == nil || it.Body == nil {
				return fmt.Errorf("printer: section is missing its name or body")
			}
			head := it.Name.Name
			if it.Qualifier != nil {
				head += " " + it.Qualifier.Name
			}
			if err := p.stmt(&it.Comments, head, it.Body, it.Name.NamePos.Line, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("printer: cannot print body item %T", item)
		}
	}
	return nil
}

// declGroup prints a single-or-factored declaration. printSpec prints
// spec i as a full statement, prefixed for the single form where the
// keyword line is the spec line; a non-empty prefix therefore also
// marks the spec as top level. A multi-spec declaration prints factored
// even without a recorded "(" position — the only form that can hold
// it — so synthesized trees need no fabricated positions.
//
// A single-form declaration carries no comments of its own (the parser
// attaches them to the spec, see the ast package documentation), so the
// single path prints only the spec.
func (p *printer) declGroup(keyword string, start int, c *ast.Comments, lparen token.Position, n int, printSpec func(i int, prefix string) error, after []*ast.CommentGroup, rparen token.Position) error {
	if !lparen.IsValid() && n == 1 {
		return printSpec(0, keyword+" ")
	}
	if !lparen.IsValid() && n == 0 {
		return fmt.Errorf("printer: %s declaration has no specs", keyword)
	}
	unknown := !p.comments(c.Before, true)
	p.startItem(start, unknown)
	open, cls := splitSuffix(c.Suffix, lparen.Line, rparen.Line)
	if n == 0 && len(after) == 0 {
		p.writeLine(keyword + " ()" + open + cls)
		p.lastEnd = rparen.Line
		return nil
	}
	p.writeLine(keyword + " (" + open)
	p.depth++
	p.lastEnd = lparen.Line
	p.snug = true
	for i := range n {
		if err := printSpec(i, ""); err != nil {
			return err
		}
	}
	p.comments(after, false)
	p.depth--
	p.snug = false
	p.writeLine(")" + cls)
	p.lastEnd = rparen.Line
	return nil
}

// file prints one solution unit.
func (p *printer) file(f *ast.File) error {
	if f == nil || f.Solution == nil || f.Solution.Name == nil {
		return fmt.Errorf("printer: file has no solution clause")
	}
	sol := f.Solution
	if err := p.stmt(&sol.Comments, "solution "+sol.Name.Name, nil, sol.Keyword.Line, false); err != nil {
		return err
	}
	p.force = true
	for _, d := range f.Imports {
		if err := p.importDecl(d); err != nil {
			return err
		}
	}
	p.force = true
	for _, decl := range f.Decls {
		var err error
		switch d := decl.(type) {
		case *ast.ExternDecl:
			err = p.externDecl(d)
		case *ast.VarDecl:
			err = p.varDecl(d)
		case *ast.DefaultDecl:
			err = p.defaultDecl(d)
		case *ast.DeployDecl:
			err = p.deployDecl(d)
		case *ast.ProvisionDecl:
			err = p.provisionDecl(d)
		default:
			err = fmt.Errorf("printer: cannot print declaration %T", decl)
		}
		if err != nil {
			return err
		}
	}
	p.comments(f.After, true)
	return nil
}

func (p *printer) importDecl(d *ast.ImportDecl) error {
	return p.declGroup("import", d.Keyword.Line, &d.Comments, d.Lparen, len(d.Specs), func(i int, prefix string) error {
		return p.importSpec(d.Specs[i], prefix)
	}, d.After, d.Rparen)
}

func (p *printer) importSpec(s *ast.ImportSpec, prefix string) error {
	if s.Path == nil {
		return fmt.Errorf("printer: import spec has no path")
	}
	head := prefix
	if s.Alias != nil {
		head += s.Alias.Name + " "
	}
	head += stringText(s.Path)
	return p.stmt(&s.Comments, head, nil, s.Pos().Line, prefix != "")
}

func (p *printer) externDecl(d *ast.ExternDecl) error {
	return p.declGroup("extern", d.Keyword.Line, &d.Comments, d.Lparen, len(d.Specs), func(i int, prefix string) error {
		return p.externSpec(d.Specs[i], prefix)
	}, d.After, d.Rparen)
}

func (p *printer) externSpec(s *ast.ExternSpec, prefix string) error {
	if s.Name == nil {
		return fmt.Errorf("printer: extern spec has no name")
	}
	t, err := typeRefText(s.Type)
	if err != nil {
		return err
	}
	return p.stmt(&s.Comments, prefix+s.Name.Name+" "+t, nil, s.Name.NamePos.Line, prefix != "")
}

func (p *printer) varDecl(d *ast.VarDecl) error {
	return p.declGroup("var", d.Keyword.Line, &d.Comments, d.Lparen, len(d.Specs), func(i int, prefix string) error {
		return p.varSpec(d.Specs[i], prefix)
	}, d.After, d.Rparen)
}

func (p *printer) varSpec(s *ast.VarSpec, prefix string) error {
	if s.Name == nil {
		return fmt.Errorf("printer: var spec has no name")
	}
	text, err := valueText(s.Value)
	if err != nil {
		return err
	}
	return p.stmt(&s.Comments, prefix+s.Name.Name+": "+text, nil, s.Name.NamePos.Line, prefix != "")
}

func (p *printer) defaultDecl(d *ast.DefaultDecl) error {
	var target string
	switch t := d.Target.(type) {
	case *ast.TypeRef:
		text, err := typeRefText(t)
		if err != nil {
			return err
		}
		target = text
	case *ast.Ident:
		target = t.Name
	default:
		return fmt.Errorf("printer: default declaration has no target")
	}
	if d.Body == nil {
		return fmt.Errorf("printer: default declaration has no body")
	}
	return p.stmt(&d.Comments, "default "+target, d.Body, d.Keyword.Line, true)
}

func (p *printer) deployDecl(d *ast.DeployDecl) error {
	return p.declGroup("deploy", d.Keyword.Line, &d.Comments, d.Lparen, len(d.Specs), func(i int, prefix string) error {
		return p.deploySpec(d.Specs[i], prefix)
	}, d.After, d.Rparen)
}

func (p *printer) deploySpec(s *ast.DeploySpec, prefix string) error {
	t, err := typeRefText(s.Type)
	if err != nil {
		return err
	}
	if s.Name == nil {
		return fmt.Errorf("printer: deploy spec has no instance name")
	}
	return p.stmt(&s.Comments, prefix+t+" as "+s.Name.Name, s.Body, s.Pos().Line, prefix != "")
}

func (p *printer) provisionDecl(d *ast.ProvisionDecl) error {
	return p.declGroup("provision", d.Keyword.Line, &d.Comments, d.Lparen, len(d.Specs), func(i int, prefix string) error {
		return p.provisionSpec(d.Specs[i], prefix)
	}, d.After, d.Rparen)
}

func (p *printer) provisionSpec(s *ast.ProvisionSpec, prefix string) error {
	t, err := typeRefText(s.Type)
	if err != nil {
		return err
	}
	if s.Name == nil {
		return fmt.Errorf("printer: provision spec has no instance name")
	}
	head := prefix + t
	if s.Kind != nil {
		head += " " + s.Kind.Name
	}
	head += " as " + s.Name.Name
	return p.stmt(&s.Comments, head, s.Body, s.Pos().Line, prefix != "")
}

// valueText renders a parameter value: preserved lexemes for parsed
// literals, canonical lexemes for synthesized ones (see the package
// documentation).
func valueText(v ast.Value) (string, error) {
	switch v := v.(type) {
	case *ast.StringLit:
		return stringText(v), nil
	case *ast.IntLit:
		if v.Raw != "" {
			return v.Raw, nil
		}
		return strconv.FormatInt(v.Value, 10), nil
	case *ast.DurationLit:
		if v.Raw != "" {
			return v.Raw, nil
		}
		return v.Value.String(), nil
	case *ast.BoolLit:
		return strconv.FormatBool(v.Value), nil
	case *ast.RefExpr:
		if v.X == nil {
			return "", fmt.Errorf("printer: reference has no symbol")
		}
		if v.Sel != nil {
			return v.X.Name + "." + v.Sel.Name, nil
		}
		return v.X.Name, nil
	case nil:
		return "", fmt.Errorf("printer: missing value")
	}
	return "", fmt.Errorf("printer: cannot print value %T", v)
}

// stringText renders a string literal: the author's lexeme when
// recorded, canonical quoting otherwise.
func stringText(s *ast.StringLit) string {
	if s.Raw != "" {
		return s.Raw
	}
	return strconv.Quote(s.Value)
}

// typeRefText renders a type reference. The bare unqualified form is
// printed as parsed; whether it resolves is the linker's business.
func typeRefText(t *ast.TypeRef) (string, error) {
	if t == nil || t.Name == nil {
		return "", fmt.Errorf("printer: missing type reference")
	}
	if t.Pkg != nil {
		return t.Pkg.Name + "." + t.Name.Name, nil
	}
	return t.Name.Name, nil
}
