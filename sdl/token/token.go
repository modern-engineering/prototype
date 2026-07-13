// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package token

import (
	"iter"
	"strconv"
)

// Token is the set of lexical tokens of the SDL language.
type Token int

// The list of tokens.
const (
	// Special tokens
	ILLEGAL Token = iota
	EOF
	NEWLINE // statement-terminating line break
	COMMENT // // line comment

	literal_beg
	// Identifiers and basic type literals
	IDENT    // Ping1
	STRING   // "com.acme.Echo"
	INT      // -1
	DURATION // 1h30m
	literal_end

	operator_beg
	// Punctuation
	LBRACE // {
	RBRACE // }
	LPAREN // (
	RPAREN // )
	COLON  // :
	PERIOD // .
	operator_end

	keyword_beg
	// Keywords
	SOLUTION  // solution
	IMPORT    // import
	EXTERN    // extern
	VAR       // var
	DEFAULT   // default
	DEPLOY    // deploy
	PROVISION // provision
	AS        // as
	TRUE      // true
	FALSE     // false
	keyword_end
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	NEWLINE: "newline",
	COMMENT: "comment",

	IDENT:    "IDENT",
	STRING:   "STRING",
	INT:      "INT",
	DURATION: "DURATION",

	LBRACE: "{",
	RBRACE: "}",
	LPAREN: "(",
	RPAREN: ")",
	COLON:  ":",
	PERIOD: ".",

	SOLUTION:  "solution",
	IMPORT:    "import",
	EXTERN:    "extern",
	VAR:       "var",
	DEFAULT:   "default",
	DEPLOY:    "deploy",
	PROVISION: "provision",
	AS:        "as",
	TRUE:      "true",
	FALSE:     "false",
}

// String returns the string corresponding to the token tok.
// For punctuation and keywords the string is the actual token character
// sequence (e.g., for the token COLON, the string is ":"). For all other
// tokens the string corresponds to the token constant name (e.g. for the
// token IDENT, the string is "IDENT").
func (tok Token) String() string {
	s := ""
	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}
	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}
	return s
}

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token, keyword_end-(keyword_beg+1))
	for i := keyword_beg + 1; i < keyword_end; i++ {
		keywords[tokens[i]] = i
	}
}

// Keywords yields the keyword lexemes of the language in token order.
// It exists for tooling generated from the grammar — syntax
// highlighters, completion — which needs the whole keyword vocabulary,
// not the one-word question [Lookup] answers.
func Keywords() iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := keyword_beg + 1; tok < keyword_end; tok++ {
			if !yield(tokens[tok]) {
				return
			}
		}
	}
}

// Lookup maps an identifier to its keyword token or [IDENT] if it is not
// a keyword.
func Lookup(ident string) Token {
	if tok, isKeyword := keywords[ident]; isKeyword {
		return tok
	}
	return IDENT
}

// IsLiteral returns true for tokens corresponding to identifiers and
// basic literals; it returns false otherwise.
func (tok Token) IsLiteral() bool { return literal_beg < tok && tok < literal_end }

// IsOperator returns true for tokens corresponding to punctuation;
// it returns false otherwise.
func (tok Token) IsOperator() bool { return operator_beg < tok && tok < operator_end }

// IsKeyword returns true for tokens corresponding to keywords;
// it returns false otherwise.
func (tok Token) IsKeyword() bool { return keyword_beg < tok && tok < keyword_end }
