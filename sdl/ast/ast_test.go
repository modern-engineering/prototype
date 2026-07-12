// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package ast_test

import (
	"testing"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/token"
)

// The sealed unions cover exactly the node kinds the grammar admits.
var (
	_ ast.Decl = (*ast.ExternDecl)(nil)
	_ ast.Decl = (*ast.VarDecl)(nil)
	_ ast.Decl = (*ast.DefaultDecl)(nil)
	_ ast.Decl = (*ast.DeployDecl)(nil)
	_ ast.Decl = (*ast.ProvisionDecl)(nil)

	_ ast.BodyItem = (*ast.Param)(nil)
	_ ast.BodyItem = (*ast.Section)(nil)

	_ ast.Value = (*ast.BadValue)(nil)
	_ ast.Value = (*ast.StringLit)(nil)
	_ ast.Value = (*ast.IntLit)(nil)
	_ ast.Value = (*ast.DurationLit)(nil)
	_ ast.Value = (*ast.BoolLit)(nil)
	_ ast.Value = (*ast.RefExpr)(nil)

	_ ast.Node = (*ast.File)(nil)
	_ ast.Node = (*ast.SolutionClause)(nil)
	_ ast.Node = (*ast.ImportDecl)(nil)
	_ ast.Node = (*ast.ImportSpec)(nil)
	_ ast.Node = (*ast.ExternSpec)(nil)
	_ ast.Node = (*ast.VarSpec)(nil)
	_ ast.Node = (*ast.DeploySpec)(nil)
	_ ast.Node = (*ast.ProvisionSpec)(nil)
	_ ast.Node = (*ast.Body)(nil)
	_ ast.Node = (*ast.Comment)(nil)
	_ ast.Node = (*ast.CommentGroup)(nil)
	_ ast.Node = (*ast.TypeRef)(nil)
	_ ast.Node = (*ast.Ident)(nil)
)

func TestPosAnchors(t *testing.T) {
	at := func(line, col int) token.Position {
		return token.Position{Filename: "f.sdl", Line: line, Column: col}
	}
	pkg := &ast.Ident{NamePos: at(1, 8), Name: "ff"}
	name := &ast.Ident{NamePos: at(1, 11), Name: "Ping"}

	tests := []struct {
		name string
		node ast.Node
		want token.Position
	}{
		{"Ident", name, at(1, 11)},
		{"qualified TypeRef", &ast.TypeRef{Pkg: pkg, Name: name}, at(1, 8)},
		{"bare TypeRef", &ast.TypeRef{Name: name}, at(1, 11)},
		{"RefExpr", &ast.RefExpr{X: pkg, Sel: name}, at(1, 8)},
		{"StringLit", &ast.StringLit{ValuePos: at(2, 3)}, at(2, 3)},
		{"Param", &ast.Param{Key: name}, at(1, 11)},
		{"Section", &ast.Section{Name: name}, at(1, 11)},
		{"qualified Section", &ast.Section{Name: pkg, Qualifier: name}, at(1, 8)},
		{"DeploySpec", &ast.DeploySpec{Type: &ast.TypeRef{Pkg: pkg, Name: name}}, at(1, 8)},
		{"ImportSpec with alias", &ast.ImportSpec{Alias: pkg, Path: &ast.StringLit{ValuePos: at(1, 11)}}, at(1, 8)},
		{"ImportSpec bare", &ast.ImportSpec{Path: &ast.StringLit{ValuePos: at(1, 11)}}, at(1, 11)},
		{"CommentGroup", &ast.CommentGroup{List: []*ast.Comment{{Slash: at(3, 1)}}}, at(3, 1)},
		{"File without clause", &ast.File{}, token.Position{}},
	}
	for _, tt := range tests {
		if got := tt.node.Pos(); got != tt.want {
			t.Errorf("%s: Pos() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
