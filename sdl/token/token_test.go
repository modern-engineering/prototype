// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package token_test

import (
	"testing"

	"github.com/modern-engineering/prototype/sdl/token"
)

func TestTokenString(t *testing.T) {
	tests := []struct {
		tok  token.Token
		want string
	}{
		{token.ILLEGAL, "ILLEGAL"},
		{token.EOF, "EOF"},
		{token.NEWLINE, "newline"},
		{token.COMMENT, "comment"},
		{token.IDENT, "IDENT"},
		{token.STRING, "STRING"},
		{token.INT, "INT"},
		{token.DURATION, "DURATION"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.COLON, ":"},
		{token.PERIOD, "."},
		{token.SOLUTION, "solution"},
		{token.IMPORT, "import"},
		{token.EXTERN, "extern"},
		{token.VAR, "var"},
		{token.DEFAULT, "default"},
		{token.DEPLOY, "deploy"},
		{token.PROVISION, "provision"},
		{token.AS, "as"},
		{token.TRUE, "true"},
		{token.FALSE, "false"},
		{token.Token(999), "token(999)"},
	}
	for _, tt := range tests {
		if got := tt.tok.String(); got != tt.want {
			t.Errorf("Token(%d).String() = %q, want %q", int(tt.tok), got, tt.want)
		}
	}
}

func TestLookup(t *testing.T) {
	keywords := map[string]token.Token{
		"solution":  token.SOLUTION,
		"import":    token.IMPORT,
		"extern":    token.EXTERN,
		"var":       token.VAR,
		"default":   token.DEFAULT,
		"deploy":    token.DEPLOY,
		"provision": token.PROVISION,
		"as":        token.AS,
		"true":      token.TRUE,
		"false":     token.FALSE,
	}
	for ident, want := range keywords {
		if got := token.Lookup(ident); got != want {
			t.Errorf("Lookup(%q) = %v, want %v", ident, got, want)
		}
	}
	for _, ident := range []string{"Ping", "solutions", "Deploy", "_", "trueish"} {
		if got := token.Lookup(ident); got != token.IDENT {
			t.Errorf("Lookup(%q) = %v, want IDENT", ident, got)
		}
	}
}

// TestKeywords pins the iterator against the two exported views of the
// same vocabulary: every yielded word must Lookup to a keyword token,
// and together the words must cover the keyword range exactly — one
// word per keyword token, in token order.
func TestKeywords(t *testing.T) {
	want := []string{"solution", "import", "extern", "var", "default", "deploy", "provision", "as", "true", "false"}
	var got []string
	for kw := range token.Keywords() {
		if tok := token.Lookup(kw); !tok.IsKeyword() {
			t.Errorf("Keywords() yielded %q, but Lookup(%q) = %v, not a keyword", kw, kw, tok)
		}
		got = append(got, kw)
	}
	if len(got) != len(want) {
		t.Fatalf("Keywords() yielded %d words %q, want %d", len(got), got, len(want))
	}
	for i, kw := range want {
		if got[i] != kw {
			t.Errorf("Keywords()[%d] = %q, want %q", i, got[i], kw)
		}
	}

	// An early break must stop the iteration, the iter.Seq contract.
	n := 0
	for range token.Keywords() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("break after the first word iterated %d times, want 1", n)
	}
}

func TestPredicates(t *testing.T) {
	tests := []struct {
		tok                        token.Token
		literal, operator, keyword bool
	}{
		{token.IDENT, true, false, false},
		{token.STRING, true, false, false},
		{token.DURATION, true, false, false},
		{token.COLON, false, true, false},
		{token.RPAREN, false, true, false},
		{token.AS, false, false, true},
		{token.TRUE, false, false, true},
		{token.NEWLINE, false, false, false},
		{token.EOF, false, false, false},
	}
	for _, tt := range tests {
		if got := tt.tok.IsLiteral(); got != tt.literal {
			t.Errorf("%v.IsLiteral() = %v, want %v", tt.tok, got, tt.literal)
		}
		if got := tt.tok.IsOperator(); got != tt.operator {
			t.Errorf("%v.IsOperator() = %v, want %v", tt.tok, got, tt.operator)
		}
		if got := tt.tok.IsKeyword(); got != tt.keyword {
			t.Errorf("%v.IsKeyword() = %v, want %v", tt.tok, got, tt.keyword)
		}
	}
}

func TestPositionString(t *testing.T) {
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

func TestFilePosition(t *testing.T) {
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
