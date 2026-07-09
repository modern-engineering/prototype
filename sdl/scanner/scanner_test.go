// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package scanner_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
)

// A result records one scanned token with its position as "line:col".
type result struct {
	tok token.Token
	lit string
	pos string
}

// scanAll tokenizes src to EOF (inclusive) and returns the token stream
// and the sorted error list.
func scanAll(src string) ([]result, scanner.ErrorList) {
	var errs scanner.ErrorList
	var s scanner.Scanner
	s.Init("test.sdl", []byte(src), errs.Add)
	var out []result
	for {
		pos, tok, lit := s.Scan()
		out = append(out, result{tok, lit, fmt.Sprintf("%d:%d", pos.Line, pos.Column)})
		if tok == token.EOF {
			break
		}
	}
	errs.Sort()
	return out, errs
}

func assertStream(t *testing.T, src string, want []result) {
	t.Helper()
	got, errs := scanAll(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected scan errors: %v", errs)
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("token %d: got {%v %q %s}, want {%v %q %s}",
				i, got[i].tok, got[i].lit, got[i].pos, want[i].tok, want[i].lit, want[i].pos)
		}
	}
	if len(got) != len(want) {
		t.Errorf("token count: got %d, want %d\ngot: %v", len(got), len(want), got)
	}
}

func TestScanStatementStream(t *testing.T) {
	// Exercises keywords, qualified names, newline suppression after '{',
	// comment transparency for pending NEWLINEs, and blank-line folding.
	src := "solution sample\n\ndeploy ff.Ping as P1 { // open\n\tcount: -1 // endless\n\tok: true\n}\n"
	assertStream(t, src, []result{
		{token.SOLUTION, "solution", "1:1"},
		{token.IDENT, "sample", "1:10"},
		{token.NEWLINE, "\n", "1:16"},
		{token.DEPLOY, "deploy", "3:1"},
		{token.IDENT, "ff", "3:8"},
		{token.PERIOD, "", "3:10"},
		{token.IDENT, "Ping", "3:11"},
		{token.AS, "as", "3:16"},
		{token.IDENT, "P1", "3:19"},
		{token.LBRACE, "", "3:22"},
		{token.COMMENT, "// open", "3:24"},
		{token.IDENT, "count", "4:2"},
		{token.COLON, "", "4:7"},
		{token.INT, "-1", "4:9"},
		{token.COMMENT, "// endless", "4:12"},
		{token.NEWLINE, "\n", "4:22"},
		{token.IDENT, "ok", "5:2"},
		{token.COLON, "", "5:4"},
		{token.TRUE, "true", "5:6"},
		{token.NEWLINE, "\n", "5:10"},
		{token.RBRACE, "", "6:1"},
		{token.NEWLINE, "\n", "6:2"},
		{token.EOF, "", "6:3"},
	})
}

func TestScanParenFolding(t *testing.T) {
	// No NEWLINE after '(' and a NEWLINE after ')'.
	src := "import (\n\tff \"e/ff\"\n)\n"
	assertStream(t, src, []result{
		{token.IMPORT, "import", "1:1"},
		{token.LPAREN, "", "1:8"},
		{token.IDENT, "ff", "2:2"},
		{token.STRING, `"e/ff"`, "2:5"},
		{token.NEWLINE, "\n", "2:11"},
		{token.RPAREN, "", "3:1"},
		{token.NEWLINE, "\n", "3:2"},
		{token.EOF, "", "3:3"},
	})
}

func TestScanContinuationAndEOFNewline(t *testing.T) {
	// A line break after ':' does not terminate; a NEWLINE is synthesized
	// at EOF when the last token can end a statement.
	src := "x:\n1"
	assertStream(t, src, []result{
		{token.IDENT, "x", "1:1"},
		{token.COLON, "", "1:2"},
		{token.INT, "1", "2:1"},
		{token.NEWLINE, "\n", "2:2"},
		{token.EOF, "", "2:2"},
	})
}

func TestScanCommentOnlyLines(t *testing.T) {
	// Comment-only lines and a trailing comment produce no NEWLINEs.
	src := "// a\n// b\nx: 1\n// tail"
	assertStream(t, src, []result{
		{token.COMMENT, "// a", "1:1"},
		{token.COMMENT, "// b", "2:1"},
		{token.IDENT, "x", "3:1"},
		{token.COLON, "", "3:2"},
		{token.INT, "1", "3:4"},
		{token.NEWLINE, "\n", "3:5"},
		{token.COMMENT, "// tail", "4:1"},
		{token.EOF, "", "4:8"},
	})
}

func TestScanNumbersAndDurations(t *testing.T) {
	src := "a: 1s\nb: -1h30m\nc: 250ms\nd: +42\ne: -7\nf: 1.5h\ng: 250µs\n"
	tests := []struct {
		tok token.Token
		lit string
	}{
		{token.DURATION, "1s"},
		{token.DURATION, "-1h30m"},
		{token.DURATION, "250ms"},
		{token.INT, "+42"},
		{token.INT, "-7"},
		{token.DURATION, "1.5h"},
		{token.DURATION, "250µs"},
	}
	got, errs := scanAll(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected scan errors: %v", errs)
	}
	var values []result
	for _, r := range got {
		if r.tok == token.INT || r.tok == token.DURATION {
			values = append(values, r)
		}
	}
	if len(values) != len(tests) {
		t.Fatalf("value token count: got %d, want %d", len(values), len(tests))
	}
	for i, tt := range tests {
		if values[i].tok != tt.tok || values[i].lit != tt.lit {
			t.Errorf("value %d: got {%v %q}, want {%v %q}", i, values[i].tok, values[i].lit, tt.tok, tt.lit)
		}
		if values[i].pos != fmt.Sprintf("%d:4", i+1) {
			t.Errorf("value %d: position %s, want %d:4", i, values[i].pos, i+1)
		}
	}
}

func TestScanStrings(t *testing.T) {
	// Escaped quote, escape sequences, and non-ASCII text: one STRING
	// token whose lexeme unquotes to the interpreted value.
	src := "s: \"a\\\"b\\n\\x41\\u00e9 日本\""
	got, errs := scanAll(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected scan errors: %v", errs)
	}
	if len(got) != 5 { // IDENT COLON STRING NEWLINE EOF
		t.Fatalf("token count: got %d (%v), want 5", len(got), got)
	}
	str := got[2]
	if str.tok != token.STRING || str.pos != "1:4" {
		t.Fatalf("string token: got {%v %q %s}", str.tok, str.lit, str.pos)
	}
	unquoted, err := strconv.Unquote(str.lit)
	if err != nil {
		t.Fatalf("Unquote(%q): %v", str.lit, err)
	}
	if want := "a\"b\nAé 日本"; unquoted != want {
		t.Errorf("Unquote(%q) = %q, want %q", str.lit, unquoted, want)
	}
}

func TestScanErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		pos  string
		msg  string
	}{
		{"unterminated string at EOF", `x: "abc`, "1:4", "string literal not terminated"},
		{"unterminated string at newline", "x: \"abc\ny: 1", "1:4", "string literal not terminated"},
		{"unknown escape", `x: "a\qb"`, "1:7", "unknown escape sequence"},
		{"bad hex escape digit", `x: "\xg1"`, "1:7", "illegal character U+0067 'g' in escape sequence"},
		{"surrogate escape", `x: "\ud800"`, "1:6", "escape sequence is invalid Unicode code point"},
		{"stray char", "x: @1", "1:4", "illegal character U+0040 '@'"},
		{"lone slash", "x: / y", "1:4", "illegal character U+002F '/'"},
		{"lone minus", "x: - 1", "1:4", "illegal character U+002D '-'"},
		{"NUL", "x: \x001", "1:4", "illegal character NUL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := scanAll(tt.src)
			if len(errs) == 0 {
				t.Fatalf("no errors scanning %q", tt.src)
			}
			e := errs[0]
			if got := fmt.Sprintf("%d:%d", e.Pos.Line, e.Pos.Column); got != tt.pos {
				t.Errorf("first error at %s, want %s (%v)", got, tt.pos, errs)
			}
			if e.Msg != tt.msg {
				t.Errorf("first error msg %q, want %q", e.Msg, tt.msg)
			}
		})
	}
}

func TestScanContinuesAfterError(t *testing.T) {
	// An unterminated string on line 1 must not eat line 2.
	src := "x: \"abc\ny: 1\n"
	got, errs := scanAll(src)
	if len(errs) != 1 {
		t.Fatalf("error count: got %d (%v), want 1", len(errs), errs)
	}
	var idents []string
	for _, r := range got {
		if r.tok == token.IDENT {
			idents = append(idents, r.lit)
		}
	}
	if len(idents) != 2 || idents[0] != "x" || idents[1] != "y" {
		t.Errorf("idents after error: got %v, want [x y]", idents)
	}
}
