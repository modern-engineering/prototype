// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package token_test

import (
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/sdl/token"
)

// One scenario holds the lexical vocabulary together: every token
// renders through String and answers exactly its class's predicates,
// each keyword's rendering is the identifier Lookup maps back to the
// token, Keywords yields precisely those lexemes in token order, and
// any other identifier stays IDENT.
func TestVocabulary(t *testing.T) {
	const (
		special = iota // answers none of the class predicates
		literal
		operator
		keyword
	)
	vocabulary := []struct {
		tok   token.Token
		str   string
		class int
	}{
		{token.ILLEGAL, "ILLEGAL", special},
		{token.EOF, "EOF", special},
		{token.NEWLINE, "newline", special},
		{token.COMMENT, "comment", special},
		{token.IDENT, "IDENT", literal},
		{token.STRING, "STRING", literal},
		{token.INT, "INT", literal},
		{token.DURATION, "DURATION", literal},
		{token.LBRACE, "{", operator},
		{token.RBRACE, "}", operator},
		{token.LPAREN, "(", operator},
		{token.RPAREN, ")", operator},
		{token.COLON, ":", operator},
		{token.PERIOD, ".", operator},
		{token.SOLUTION, "solution", keyword},
		{token.IMPORT, "import", keyword},
		{token.EXTERN, "extern", keyword},
		{token.VAR, "var", keyword},
		{token.DEFAULT, "default", keyword},
		{token.DEPLOY, "deploy", keyword},
		{token.PROVISION, "provision", keyword},
		{token.AS, "as", keyword},
		{token.TRUE, "true", keyword},
		{token.FALSE, "false", keyword},
	}
	var keywords []string
	for _, tt := range vocabulary {
		if got := tt.tok.String(); got != tt.str {
			t.Errorf("Token(%d).String() = %q, want %q", int(tt.tok), got, tt.str)
		}
		if got := tt.tok.IsLiteral(); got != (tt.class == literal) {
			t.Errorf("%v.IsLiteral() = %v, want %v", tt.tok, got, tt.class == literal)
		}
		if got := tt.tok.IsOperator(); got != (tt.class == operator) {
			t.Errorf("%v.IsOperator() = %v, want %v", tt.tok, got, tt.class == operator)
		}
		if got := tt.tok.IsKeyword(); got != (tt.class == keyword) {
			t.Errorf("%v.IsKeyword() = %v, want %v", tt.tok, got, tt.class == keyword)
		}
		if tt.class == keyword {
			if got := token.Lookup(tt.str); got != tt.tok {
				t.Errorf("Lookup(%q) = %v, want %v", tt.str, got, tt.tok)
			}
			keywords = append(keywords, tt.str)
		}
	}

	if got := slices.Collect(token.Keywords()); !slices.Equal(got, keywords) {
		t.Errorf("Keywords() yielded %q, want the vocabulary's keyword rows %q", got, keywords)
	}

	// An early break must stop the iteration, the iter.Seq contract.
	n := 0
	for range token.Keywords() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("break after the first keyword iterated %d times, want 1", n)
	}

	// Identifiers outside the keyword rows — case variants and
	// extensions of real keywords included — stay IDENT.
	for _, ident := range []string{"Ping", "solutions", "Deploy", "_", "trueish"} {
		if got := token.Lookup(ident); got != token.IDENT {
			t.Errorf("Lookup(%q) = %v, want IDENT", ident, got)
		}
	}

	// A value outside the vocabulary still renders diagnosably.
	if got := token.Token(999).String(); got != "token(999)" {
		t.Errorf(`Token(999).String() = %q, want "token(999)"`, got)
	}
}

func TestPositionRendering(t *testing.T) {
	tests := []struct {
		pos  token.Position
		want string
	}{
		{token.Position{Filename: "f.sdl", Line: 3, Column: 7}, "f.sdl:3:7"},
		{token.Position{Line: 3, Column: 7}, "3:7"},
		{token.Position{Line: 3}, "3"},
		{token.Position{Filename: "f.sdl"}, "f.sdl"},
		{token.Position{}, "-"},
	}
	for _, tt := range tests {
		if got := tt.pos.String(); got != tt.want {
			t.Errorf("%#v.String() = %q, want %q", tt.pos, got, tt.want)
		}
	}
	valid := token.Position{Line: 1, Column: 1}
	if !valid.IsValid() {
		t.Error("Position{Line: 1, Column: 1}.IsValid() = false, want true")
	}
	invalid := token.Position{}
	if invalid.IsValid() {
		t.Error("Position{}.IsValid() = true, want false")
	}
}

func TestFileOffsetMapping(t *testing.T) {
	// Source: "ab\ncd\n" — two lines, six bytes.
	f := token.NewFile("f.sdl", 6)
	f.AddLine(3) // second line starts at offset 3
	f.AddLine(3) // duplicate is ignored
	f.AddLine(6) // at file size: ignored
	if got := f.Name(); got != "f.sdl" {
		t.Errorf("Name() = %q, want %q", got, "f.sdl")
	}
	if got := f.Size(); got != 6 {
		t.Errorf("Size() = %d, want 6", got)
	}
	if got := f.LineCount(); got != 2 {
		t.Errorf("LineCount() = %d, want 2", got)
	}
	tests := []struct {
		offset     int
		line, col  int
		wantOffset int
	}{
		{0, 1, 1, 0},
		{2, 1, 3, 2},  // the first '\n'
		{3, 2, 1, 3},  // 'c'
		{4, 2, 2, 4},  // 'd'
		{6, 2, 4, 6},  // end of file
		{-1, 1, 1, 0}, // clamped low
		{99, 2, 4, 6}, // clamped high
	}
	for _, tt := range tests {
		pos := f.Position(tt.offset)
		if pos.Line != tt.line || pos.Column != tt.col || pos.Offset != tt.wantOffset || pos.Filename != "f.sdl" {
			t.Errorf("Position(%d) = %v (offset %d), want %d:%d (offset %d)",
				tt.offset, pos, pos.Offset, tt.line, tt.col, tt.wantOffset)
		}
	}
}
