// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package ast

import (
	"time"

	"github.com/modern-engineering/prototype/sdl/token"
)

// ----------------------------------------------------------------------------
// Interfaces

// All node types implement the Node interface.
type Node interface {
	Pos() token.Position // position of the first token belonging to the node
}

// All top-level declaration nodes implement the Decl interface.
// Import declarations are not Decls; they live in [File.Imports].
type Decl interface {
	Node
	declNode()
}

// All body item nodes ([Param], [Section]) implement the BodyItem
// interface.
type BodyItem interface {
	Node
	bodyItemNode()
}

// All value nodes implement the Value interface.
type Value interface {
	Node
	valueNode()
}

// ----------------------------------------------------------------------------
// Comments

// A Comment node represents a single // line comment.
type Comment struct {
	Slash token.Position // position of the leading '/'
	Text  string         // comment text, including the "//", excluding the line break
}

// Pos returns the position of the comment's leading '/'.
func (c *Comment) Pos() token.Position { return c.Slash }

// A CommentGroup represents a sequence of full-line comments on adjacent
// lines, with no blank line or other token between them.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

// Pos returns the position of the group's first comment.
func (g *CommentGroup) Pos() token.Position { return g.List[0].Slash }

// Comments carries the comment slots shared by all statement-shaped
// nodes, following the x/mod modfile model. See the package documentation
// for the attachment rules.
type Comments struct {
	Before []*CommentGroup // full-line comment groups above the statement
	Suffix *Comment        // trailing comment on the statement's last line, or nil
}

// ----------------------------------------------------------------------------
// Names, references, and values

// An Ident node represents an identifier.
type Ident struct {
	NamePos token.Position // identifier position
	Name    string         // identifier name
}

// Pos returns the position of the identifier.
func (x *Ident) Pos() token.Position { return x.NamePos }

// A TypeRef node represents a reference to a catalogue element type, such
// as ff.Ping. The design grammar requires the package qualifier; Pkg is
// nil only for the bare legacy-sketch form, which the parser tolerates in
// units that declare no imports (resolution then fails at link time).
type TypeRef struct {
	Pkg  *Ident // package qualifier; nil for the bare legacy form
	Name *Ident // element name
}

// Pos returns the position of the qualifier if present, otherwise of the
// name.
func (x *TypeRef) Pos() token.Position {
	if x.Pkg != nil {
		return x.Pkg.NamePos
	}
	return x.Name.NamePos
}

// A BadValue node is a placeholder for a value containing syntax errors
// for which no correct value node could be created.
type BadValue struct {
	From token.Position // position where the value was expected
}

// A StringLit node represents a Go interpreted string literal.
type StringLit struct {
	ValuePos token.Position // literal position
	Raw      string         // literal text as it appears in the source, including quotes
	Value    string         // interpreted value (strconv.Unquote of Raw)
}

// An IntLit node represents a decimal integer literal with an optional
// sign.
type IntLit struct {
	ValuePos token.Position // literal position
	Raw      string         // literal text as it appears in the source
	Value    int64          // integer value
}

// A DurationLit node represents a Go duration literal such as 1h30m.
type DurationLit struct {
	ValuePos token.Position // literal position
	Raw      string         // literal text as it appears in the source
	Value    time.Duration  // parsed value (time.ParseDuration of Raw)
}

// A BoolLit node represents one of the literals true or false.
type BoolLit struct {
	ValuePos token.Position // literal position
	Value    bool           // literal value
}

// A RefExpr node represents a symbol reference in value position: a bare
// identifier (var, extern, or an opaque token inside an "on" section) or
// a dotted pair naming a provision instance output such as
// natsAccount.config. The distinction is semantic, not syntactic.
type RefExpr struct {
	X   *Ident // referenced symbol
	Sel *Ident // output selector; nil for a bare reference
}

// Pos positions of the value nodes.
func (x *BadValue) Pos() token.Position    { return x.From }
func (x *StringLit) Pos() token.Position   { return x.ValuePos }
func (x *IntLit) Pos() token.Position      { return x.ValuePos }
func (x *DurationLit) Pos() token.Position { return x.ValuePos }
func (x *BoolLit) Pos() token.Position     { return x.ValuePos }
func (x *RefExpr) Pos() token.Position     { return x.X.NamePos }

// ----------------------------------------------------------------------------
// Bodies

// A Body node represents a braced statement body attached to a deploy or
// provision spec, a default declaration, or a section.
type Body struct {
	Lbrace token.Position  // position of '{'
	Items  []BodyItem      // parameters and sections, in source order
	After  []*CommentGroup // comment groups before the closing '}'
	Rbrace token.Position  // position of '}'
}

// Pos returns the position of the opening brace.
func (b *Body) Pos() token.Position { return b.Lbrace }

// A Param node represents a "key: value" body item.
type Param struct {
	Comments
	Key   *Ident // parameter key
	Value Value  // parameter value
}

// Pos returns the position of the parameter key.
func (p *Param) Pos() token.Position { return p.Key.NamePos }

// A Section node represents a colon-less named sub-body such as
// "on { ... }" or "metadata { ... }".
type Section struct {
	Comments
	Name *Ident // section name
	Body *Body  // section contents
}

// Pos returns the position of the section name.
func (s *Section) Pos() token.Position { return s.Name.NamePos }

// ----------------------------------------------------------------------------
// Clauses and declarations

// A SolutionClause node represents the "solution name" clause that opens
// every unit.
type SolutionClause struct {
	Comments
	Keyword token.Position // position of the "solution" keyword
	Name    *Ident         // solution name
}

// Pos returns the position of the "solution" keyword.
func (c *SolutionClause) Pos() token.Position { return c.Keyword }

// An ImportDecl node represents an import declaration, single or
// factored.
type ImportDecl struct {
	Comments
	Keyword token.Position  // position of the "import" keyword
	Lparen  token.Position  // position of '(', if factored (IsValid reports which)
	Specs   []*ImportSpec   // one spec, or the specs of the factored block
	After   []*CommentGroup // comment groups before ')', if factored
	Rparen  token.Position  // position of ')', if factored
}

// Pos returns the position of the "import" keyword.
func (d *ImportDecl) Pos() token.Position { return d.Keyword }

// An ImportSpec node represents a single package import.
type ImportSpec struct {
	Comments
	Alias *Ident     // local package name, or nil
	Path  *StringLit // import path
}

// Pos returns the position of the alias if present, otherwise of the
// path.
func (s *ImportSpec) Pos() token.Position {
	if s.Alias != nil {
		return s.Alias.NamePos
	}
	return s.Path.ValuePos
}

// An ExternDecl node represents an extern declaration, single or
// factored.
type ExternDecl struct {
	Comments
	Keyword token.Position  // position of the "extern" keyword
	Lparen  token.Position  // position of '(', if factored
	Specs   []*ExternSpec   // one spec, or the specs of the factored block
	After   []*CommentGroup // comment groups before ')', if factored
	Rparen  token.Position  // position of ')', if factored
}

// Pos returns the position of the "extern" keyword.
func (d *ExternDecl) Pos() token.Position { return d.Keyword }

// An ExternSpec node represents a single extern symbol declaration,
// "name pkg.Type" (no colon).
type ExternSpec struct {
	Comments
	Name *Ident   // symbol name
	Type *TypeRef // symbol type
}

// Pos returns the position of the symbol name.
func (s *ExternSpec) Pos() token.Position { return s.Name.NamePos }

// A VarDecl node represents a var declaration, single or factored.
type VarDecl struct {
	Comments
	Keyword token.Position  // position of the "var" keyword
	Lparen  token.Position  // position of '(', if factored
	Specs   []*VarSpec      // one spec, or the specs of the factored block
	After   []*CommentGroup // comment groups before ')', if factored
	Rparen  token.Position  // position of ')', if factored
}

// Pos returns the position of the "var" keyword.
func (d *VarDecl) Pos() token.Position { return d.Keyword }

// A VarSpec node represents a single var symbol declaration,
// "name: literal". The grammar admits literal values only.
type VarSpec struct {
	Comments
	Name  *Ident // symbol name
	Value Value  // literal default value
}

// Pos returns the position of the symbol name.
func (s *VarSpec) Pos() token.Position { return s.Name.NamePos }

// A DefaultDecl node represents a default declaration. Default
// declarations do not factor.
type DefaultDecl struct {
	Comments
	Keyword token.Position // position of the "default" keyword
	Target  Node           // *TypeRef, or *Ident for the verbs "deploy" and "provision"
	Body    *Body          // defaulted parameters
}

// Pos returns the position of the "default" keyword.
func (d *DefaultDecl) Pos() token.Position { return d.Keyword }

// A DeployDecl node represents a deploy declaration, single or factored.
type DeployDecl struct {
	Comments
	Keyword token.Position  // position of the "deploy" keyword
	Lparen  token.Position  // position of '(', if factored
	Specs   []*DeploySpec   // one spec, or the specs of the factored block
	After   []*CommentGroup // comment groups before ')', if factored
	Rparen  token.Position  // position of ')', if factored
}

// Pos returns the position of the "deploy" keyword.
func (d *DeployDecl) Pos() token.Position { return d.Keyword }

// A DeploySpec node represents a single deployment,
// "pkg.Type as name { ... }".
type DeploySpec struct {
	Comments
	Type *TypeRef // component type
	Name *Ident   // instance name
	Body *Body    // parameter bindings, or nil
}

// Pos returns the position of the component type.
func (s *DeploySpec) Pos() token.Position { return s.Type.Pos() }

// A ProvisionDecl node represents a provision declaration, single or
// factored.
type ProvisionDecl struct {
	Comments
	Keyword token.Position   // position of the "provision" keyword
	Lparen  token.Position   // position of '(', if factored
	Specs   []*ProvisionSpec // one spec, or the specs of the factored block
	After   []*CommentGroup  // comment groups before ')', if factored
	Rparen  token.Position   // position of ')', if factored
}

// Pos returns the position of the "provision" keyword.
func (d *ProvisionDecl) Pos() token.Position { return d.Keyword }

// A ProvisionSpec node represents a single provision,
// "pkg.Type kind as name { ... }". The kind word (slice or attach) may be
// omitted while the type registers exactly one kind; validation is the
// linker's job.
type ProvisionSpec struct {
	Comments
	Type *TypeRef // provision type
	Kind *Ident   // provision kind word, or nil
	Name *Ident   // instance name
	Body *Body    // parameter bindings, or nil
}

// Pos returns the position of the provision type.
func (s *ProvisionSpec) Pos() token.Position { return s.Type.Pos() }

// ----------------------------------------------------------------------------
// Files

// A File node represents one SDL solution unit.
type File struct {
	Solution *SolutionClause // the unit's solution clause, or nil on error
	Imports  []*ImportDecl   // import declarations, in source order
	Decls    []Decl          // remaining declarations, in source order
	After    []*CommentGroup // comment groups after the last statement
}

// Pos returns the position of the solution clause, or an invalid position
// if the clause is missing.
func (f *File) Pos() token.Position {
	if f.Solution != nil {
		return f.Solution.Keyword
	}
	return token.Position{}
}

// ----------------------------------------------------------------------------
// Sealed marker implementations

func (*ExternDecl) declNode()    {}
func (*VarDecl) declNode()       {}
func (*DefaultDecl) declNode()   {}
func (*DeployDecl) declNode()    {}
func (*ProvisionDecl) declNode() {}

func (*Param) bodyItemNode()   {}
func (*Section) bodyItemNode() {}

func (*BadValue) valueNode()    {}
func (*StringLit) valueNode()   {}
func (*IntLit) valueNode()      {}
func (*DurationLit) valueNode() {}
func (*BoolLit) valueNode()     {}
func (*RefExpr) valueNode()     {}
