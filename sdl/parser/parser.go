// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package parser

import (
	"fmt"
	"strconv"
	"time"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
)

// ParseFile parses the source text of a single SDL solution unit and
// returns the corresponding [ast.File]. The filename is used only when
// recording positions.
//
// ParseFile always returns a non-nil file; if the source could not be
// parsed cleanly, the file holds the statements that did parse. The error
// is nil for a clean parse and a sorted [scanner.ErrorList] otherwise.
func ParseFile(filename string, src []byte) (*ast.File, error) {
	p := new(parser)
	p.scanner.Init(filename, src, func(pos token.Position, msg string) {
		p.errors.Add(pos, msg)
	})
	p.next() // prime the first token
	f := p.parseFile()
	p.errors.Sort()
	return f, p.errors.Err()
}

// The parser structure holds the parser's internal state.
type parser struct {
	scanner scanner.Scanner
	errors  scanner.ErrorList

	// current token
	pos token.Position
	tok token.Token
	lit string

	// comment collection
	suffix  *ast.Comment        // unclaimed trailing comment on the previous token's line
	pending []*ast.CommentGroup // unclaimed full-line comment groups

	hasImports bool // the unit declares imports; TypeRefs must qualify
}

// ----------------------------------------------------------------------------
// Token flow and comment collection

// next advances to the next non-comment token, collecting comments on the
// way: a comment on the same line as the token just left becomes the
// pending suffix comment, full-line comments accumulate into pending
// groups (adjacent lines form one group, a blank line starts a new one).
func (p *parser) next() {
	prevLine := p.pos.Line
	var group *ast.CommentGroup
	groupEnd := 0
	flush := func() {
		if group != nil {
			p.pending = append(p.pending, group)
			group = nil
		}
	}
	for {
		pos, tok, lit := p.scanner.Scan()
		if tok != token.COMMENT {
			p.pos, p.tok, p.lit = pos, tok, lit
			flush()
			return
		}
		c := &ast.Comment{Slash: pos, Text: lit}
		switch {
		case pos.Line == prevLine && prevLine > 0:
			p.suffix = c
		case group != nil && pos.Line == groupEnd+1:
			group.List = append(group.List, c)
			groupEnd = pos.Line
		default:
			flush()
			group = &ast.CommentGroup{List: []*ast.Comment{c}}
			groupEnd = pos.Line
		}
	}
}

// takePending returns the pending comment groups and clears them.
func (p *parser) takePending() []*ast.CommentGroup {
	g := p.pending
	p.pending = nil
	return g
}

// restorePending puts unclaimed groups back at the front of the pending
// list (used when a statement fails before producing a node).
func (p *parser) restorePending(groups []*ast.CommentGroup) {
	if len(groups) > 0 {
		p.pending = append(groups, p.pending...)
	}
}

// attachSuffix hands the pending suffix comment, if any, to c. If c's
// Suffix slot is already taken, the comment is kept as a pending group of
// its own so it is displaced rather than lost.
func (p *parser) attachSuffix(c *ast.Comments) {
	if p.suffix == nil {
		return
	}
	if c.Suffix == nil {
		c.Suffix = p.suffix
	} else {
		p.pending = append(p.pending, &ast.CommentGroup{List: []*ast.Comment{p.suffix}})
	}
	p.suffix = nil
}

// finishLine terminates a statement: it claims the trailing comment for c
// and consumes the statement-terminating NEWLINE. A closing ")" or "}" or
// EOF also terminates the statement (without being consumed), so the last
// statement of a group needs no line break of its own.
func (p *parser) finishLine(c *ast.Comments) {
	p.attachSuffix(c)
	switch p.tok {
	case token.NEWLINE:
		p.next()
	case token.EOF, token.RBRACE, token.RPAREN:
		// The closing token terminates the statement and stays.
	default:
		p.errorExpected(p.pos, "newline")
		p.advance()
	}
}

// ----------------------------------------------------------------------------
// Errors and resynchronization

func (p *parser) error(pos token.Position, msg string) {
	p.errors.Add(pos, msg)
}

// errorExpected reports that msg was expected at pos; if pos is the
// current token's position, the report names the token that was found
// instead.
func (p *parser) errorExpected(pos token.Position, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		switch {
		case p.tok == token.NEWLINE:
			msg += ", found newline"
		case p.tok.IsLiteral() || p.tok.IsKeyword():
			msg += ", found " + p.lit
		default:
			msg += ", found '" + p.tok.String() + "'"
		}
	}
	p.error(pos, msg)
}

// expect consumes the current token if it has kind tok and reports an
// error otherwise (leaving the token in place for resynchronization). It
// returns the position the token was expected at.
func (p *parser) expect(tok token.Token) token.Position {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
		return pos
	}
	p.next()
	return pos
}

// advance resynchronizes after a statement-level error: it skips to just
// past the next NEWLINE at the nesting depth where the error occurred,
// swallowing any nested blocks on the way, and stops (without consuming)
// at a closing token of the enclosing block or at EOF. Any trailing
// comment collected inside the skipped region is dropped; full-line
// groups stay pending and attach to the next statement.
func (p *parser) advance() {
	depth := 0
	for {
		switch p.tok {
		case token.EOF:
			p.suffix = nil
			return
		case token.NEWLINE:
			if depth == 0 {
				p.next()
				p.suffix = nil
				return
			}
		case token.LPAREN, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACE:
			if depth == 0 {
				p.suffix = nil
				return
			}
			depth--
		}
		p.next()
	}
}

// ----------------------------------------------------------------------------
// Identifiers, type references, and values

// parseIdent parses an identifier. On failure it reports an error and
// returns a placeholder without consuming the offending token.
func (p *parser) parseIdent() *ast.Ident {
	if p.tok == token.IDENT {
		id := &ast.Ident{NamePos: p.pos, Name: p.lit}
		p.next()
		return id
	}
	p.errorExpected(p.pos, "identifier")
	return &ast.Ident{NamePos: p.pos, Name: "_"}
}

// parseTypeRef parses a package-qualified type reference. The caller must
// have checked that the current token is an identifier. The bare form is
// tolerated in units without imports; see the package documentation.
func (p *parser) parseTypeRef() *ast.TypeRef {
	first := p.parseIdent()
	if p.tok == token.PERIOD {
		p.next()
		return &ast.TypeRef{Pkg: first, Name: p.parseIdent()}
	}
	if p.hasImports {
		p.error(first.NamePos, fmt.Sprintf(
			"unqualified type reference %s: element references must be package-qualified through an import (e.g. pkg.%s)",
			first.Name, first.Name))
	}
	return &ast.TypeRef{Name: first}
}

// parseValue parses a parameter value: a literal or a symbol reference.
// On failure it reports an error and returns an [ast.BadValue] without
// consuming the offending token.
func (p *parser) parseValue() ast.Value {
	pos := p.pos
	switch p.tok {
	case token.STRING:
		raw := p.lit
		value, err := strconv.Unquote(raw)
		if err != nil {
			value = "" // the scanner reported the malformed literal
		}
		p.next()
		return &ast.StringLit{ValuePos: pos, Raw: raw, Value: value}
	case token.INT:
		raw := p.lit
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			p.error(pos, fmt.Sprintf("invalid integer literal %s: value out of range", raw))
		}
		p.next()
		return &ast.IntLit{ValuePos: pos, Raw: raw, Value: value}
	case token.DURATION:
		raw := p.lit
		value, err := time.ParseDuration(raw)
		if err != nil {
			p.error(pos, fmt.Sprintf("invalid duration literal %s", raw))
		}
		p.next()
		return &ast.DurationLit{ValuePos: pos, Raw: raw, Value: value}
	case token.TRUE, token.FALSE:
		value := p.tok == token.TRUE
		p.next()
		return &ast.BoolLit{ValuePos: pos, Value: value}
	case token.IDENT:
		ref := &ast.RefExpr{X: p.parseIdent()}
		if p.tok == token.PERIOD {
			p.next()
			ref.Sel = p.parseIdent()
		}
		return ref
	}
	p.errorExpected(pos, "value")
	return &ast.BadValue{From: pos}
}

func isLiteral(v ast.Value) bool {
	switch v.(type) {
	case *ast.StringLit, *ast.IntLit, *ast.DurationLit, *ast.BoolLit, *ast.BadValue:
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// Bodies

// parseBody parses a braced body. The caller must have checked that the
// current token is '{'. A trailing comment on the '{' line attaches to
// owner, the node whose statement the body belongs to.
func (p *parser) parseBody(owner *ast.Comments) *ast.Body {
	b := &ast.Body{Lbrace: p.expect(token.LBRACE)}
	p.attachSuffix(owner)
	seen := make(map[string]bool)
	for p.tok != token.RBRACE && p.tok != token.EOF {
		if p.tok != token.IDENT {
			p.errorExpected(p.pos, "parameter or section")
			p.advance()
			continue
		}
		before := p.takePending()
		name := p.parseIdent()
		switch p.tok {
		case token.COLON:
			p.next()
			param := &ast.Param{Key: name, Value: p.parseValue()}
			param.Before = before
			if seen[name.Name] {
				p.error(name.NamePos, fmt.Sprintf("duplicate parameter key %s", name.Name))
			}
			seen[name.Name] = true
			p.finishLine(&param.Comments)
			b.Items = append(b.Items, param)
		case token.LBRACE:
			section := &ast.Section{Name: name}
			section.Before = before
			section.Body = p.parseBody(&section.Comments)
			p.finishLine(&section.Comments)
			b.Items = append(b.Items, section)
		default:
			p.errorExpected(p.pos, "':' or '{'")
			p.restorePending(before)
			p.advance()
		}
	}
	b.After = p.takePending()
	b.Rbrace = p.pos
	p.expect(token.RBRACE)
	return b
}

// ----------------------------------------------------------------------------
// Clauses and declarations

func (p *parser) parseSolutionClause() *ast.SolutionClause {
	c := &ast.SolutionClause{Keyword: p.pos}
	c.Before = p.takePending()
	p.next() // consume "solution"
	c.Name = p.parseIdent()
	p.finishLine(&c.Comments)
	return c
}

// parseSpecGroup parses the single-or-factored spec production shared by
// import, extern, var, deploy, and provision declarations. The spec
// callback parses one spec and reports whether it produced a node; in the
// factored form it is called once per line. dc receives the group-level
// comments; single receives the comment groups above a single-form
// declaration, whose line belongs to the spec.
func (p *parser) parseSpecGroup(dc *ast.Comments, spec func(before []*ast.CommentGroup) bool) (lparen token.Position, after []*ast.CommentGroup, rparen token.Position) {
	before := p.takePending()
	p.next() // consume the verb keyword
	if p.tok != token.LPAREN {
		if !spec(before) {
			p.restorePending(before)
		}
		return
	}
	dc.Before = before
	lparen = p.pos
	p.next()
	p.attachSuffix(dc)
	for p.tok != token.RPAREN && p.tok != token.EOF {
		spec(p.takePending())
	}
	after = p.takePending()
	rparen = p.pos
	p.expect(token.RPAREN)
	p.finishLine(dc)
	return
}

func (p *parser) parseImportDecl() *ast.ImportDecl {
	p.hasImports = true
	d := &ast.ImportDecl{Keyword: p.pos}
	d.Lparen, d.After, d.Rparen = p.parseSpecGroup(&d.Comments, func(before []*ast.CommentGroup) bool {
		s := p.parseImportSpec()
		if s == nil {
			return false
		}
		s.Before = append(before, s.Before...)
		d.Specs = append(d.Specs, s)
		return true
	})
	return d
}

func (p *parser) parseImportSpec() *ast.ImportSpec {
	if p.tok != token.IDENT && p.tok != token.STRING {
		p.errorExpected(p.pos, "import spec")
		p.advance()
		return nil
	}
	s := new(ast.ImportSpec)
	if p.tok == token.IDENT {
		s.Alias = p.parseIdent()
	}
	if p.tok != token.STRING {
		p.errorExpected(p.pos, "import path string")
		p.advance()
		return s
	}
	path, err := strconv.Unquote(p.lit)
	if err != nil {
		path = "" // the scanner reported the malformed literal
	}
	s.Path = &ast.StringLit{ValuePos: p.pos, Raw: p.lit, Value: path}
	p.next()
	p.finishLine(&s.Comments)
	return s
}

func (p *parser) parseExternDecl() *ast.ExternDecl {
	d := &ast.ExternDecl{Keyword: p.pos}
	d.Lparen, d.After, d.Rparen = p.parseSpecGroup(&d.Comments, func(before []*ast.CommentGroup) bool {
		s := p.parseExternSpec()
		if s == nil {
			return false
		}
		s.Before = append(before, s.Before...)
		d.Specs = append(d.Specs, s)
		return true
	})
	return d
}

func (p *parser) parseExternSpec() *ast.ExternSpec {
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "extern symbol name")
		p.advance()
		return nil
	}
	s := &ast.ExternSpec{Name: p.parseIdent()}
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "type reference")
		p.advance()
		return s
	}
	s.Type = p.parseTypeRef()
	p.finishLine(&s.Comments)
	return s
}

func (p *parser) parseVarDecl() *ast.VarDecl {
	d := &ast.VarDecl{Keyword: p.pos}
	d.Lparen, d.After, d.Rparen = p.parseSpecGroup(&d.Comments, func(before []*ast.CommentGroup) bool {
		s := p.parseVarSpec()
		if s == nil {
			return false
		}
		s.Before = append(before, s.Before...)
		d.Specs = append(d.Specs, s)
		return true
	})
	return d
}

func (p *parser) parseVarSpec() *ast.VarSpec {
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "var symbol name")
		p.advance()
		return nil
	}
	s := &ast.VarSpec{Name: p.parseIdent()}
	if p.tok == token.COLON {
		p.next()
	} else {
		p.errorExpected(p.pos, "':'")
	}
	s.Value = p.parseValue()
	if !isLiteral(s.Value) {
		p.error(s.Value.Pos(), "var value must be a literal (string, int, duration, or bool)")
	}
	p.finishLine(&s.Comments)
	return s
}

func (p *parser) parseDefaultDecl() *ast.DefaultDecl {
	d := &ast.DefaultDecl{Keyword: p.pos}
	d.Before = p.takePending()
	p.next() // consume "default"
	switch p.tok {
	case token.DEPLOY, token.PROVISION:
		d.Target = &ast.Ident{NamePos: p.pos, Name: p.lit}
		p.next()
	case token.IDENT:
		d.Target = p.parseTypeRef()
	case token.LPAREN:
		// "default" deliberately does not factor (CP-A); a factored
		// form is a recorded door, not a grammar gap.
		p.error(p.pos, "default does not take a factored block")
		p.advance()
		return d
	default:
		p.errorExpected(p.pos, "type reference, 'deploy', or 'provision'")
	}
	if p.tok == token.LBRACE {
		d.Body = p.parseBody(&d.Comments)
	} else {
		p.errorExpected(p.pos, "'{'")
	}
	p.finishLine(&d.Comments)
	return d
}

func (p *parser) parseDeployDecl() *ast.DeployDecl {
	d := &ast.DeployDecl{Keyword: p.pos}
	d.Lparen, d.After, d.Rparen = p.parseSpecGroup(&d.Comments, func(before []*ast.CommentGroup) bool {
		s := p.parseDeploySpec()
		if s == nil {
			return false
		}
		s.Before = append(before, s.Before...)
		d.Specs = append(d.Specs, s)
		return true
	})
	return d
}

func (p *parser) parseDeploySpec() *ast.DeploySpec {
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "type reference")
		p.advance()
		return nil
	}
	s := &ast.DeploySpec{Type: p.parseTypeRef()}
	if p.tok == token.AS {
		p.next()
	} else {
		p.errorExpected(p.pos, "'as'")
	}
	s.Name = p.parseIdent()
	if p.tok == token.LBRACE {
		s.Body = p.parseBody(&s.Comments)
	}
	p.finishLine(&s.Comments)
	return s
}

func (p *parser) parseProvisionDecl() *ast.ProvisionDecl {
	d := &ast.ProvisionDecl{Keyword: p.pos}
	d.Lparen, d.After, d.Rparen = p.parseSpecGroup(&d.Comments, func(before []*ast.CommentGroup) bool {
		s := p.parseProvisionSpec()
		if s == nil {
			return false
		}
		s.Before = append(before, s.Before...)
		d.Specs = append(d.Specs, s)
		return true
	})
	return d
}

func (p *parser) parseProvisionSpec() *ast.ProvisionSpec {
	if p.tok != token.IDENT {
		p.errorExpected(p.pos, "type reference")
		p.advance()
		return nil
	}
	s := &ast.ProvisionSpec{Type: p.parseTypeRef()}
	if p.tok == token.IDENT {
		s.Kind = p.parseIdent()
	}
	if p.tok == token.AS {
		p.next()
	} else {
		p.errorExpected(p.pos, "'as'")
	}
	s.Name = p.parseIdent()
	if p.tok == token.LBRACE {
		s.Body = p.parseBody(&s.Comments)
	}
	p.finishLine(&s.Comments)
	return s
}

// ----------------------------------------------------------------------------
// Files

func (p *parser) parseFile() *ast.File {
	f := new(ast.File)
	if p.tok == token.SOLUTION {
		f.Solution = p.parseSolutionClause()
	} else if p.tok != token.EOF {
		p.errorExpected(p.pos, "'solution'")
	}
	for p.tok != token.EOF {
		switch p.tok {
		case token.IMPORT:
			d := p.parseImportDecl()
			if len(f.Decls) > 0 {
				p.error(d.Keyword, "import declarations must precede other declarations")
			}
			f.Imports = append(f.Imports, d)
		case token.EXTERN:
			f.Decls = append(f.Decls, p.parseExternDecl())
		case token.VAR:
			f.Decls = append(f.Decls, p.parseVarDecl())
		case token.DEFAULT:
			f.Decls = append(f.Decls, p.parseDefaultDecl())
		case token.DEPLOY:
			f.Decls = append(f.Decls, p.parseDeployDecl())
		case token.PROVISION:
			f.Decls = append(f.Decls, p.parseProvisionDecl())
		case token.SOLUTION:
			p.error(p.pos, "duplicate solution clause (one per unit)")
			p.advance()
		case token.NEWLINE:
			p.next() // stray line break after error recovery
		default:
			p.errorExpected(p.pos, "declaration")
			p.advance()
		}
	}
	f.After = p.takePending()
	return f
}
