// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package highlightcmd

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution"
)

// A grammar is the language surface the emitters render, collected
// from the same code the toolchain scans and links with: the keyword
// and punctuation vocabulary of sdl/token and the body vocabulary of
// solution.Vocabulary. Collecting at run time is the point of the
// verb — an emitted syntax file reflects the grammar of the very sdl
// binary that wrote it, never a hand-maintained copy.
type grammar struct {
	keywords  []string // declaration keywords, sorted
	booleans  []string // value keywords, the boolean literals, sorted
	operators []string // punctuation lexemes, in token order
	sections  []string // section words a statement body may open, sorted
	fields    []string // union of the per-verb top-level fields, sorted
}

// collect assembles the grammar. The keyword vocabulary splits into
// declaration words and value words: true and false are keyword
// tokens lexically, but an editor colors a boolean like the literal
// it denotes.
func collect() grammar {
	var g grammar
	for kw := range token.Keywords() {
		switch token.Lookup(kw) {
		case token.TRUE, token.FALSE:
			g.booleans = append(g.booleans, kw)
		default:
			g.keywords = append(g.keywords, kw)
		}
	}
	slices.Sort(g.keywords)
	slices.Sort(g.booleans)
	for _, tok := range tokensWhere(token.Token.IsOperator) {
		g.operators = append(g.operators, tok.String())
	}
	vocab := solution.Vocabulary()
	g.sections = slices.Sorted(slices.Values(vocab.Sections))
	set := make(map[string]bool)
	for _, fields := range vocab.RootFields {
		for _, field := range fields {
			set[field] = true
		}
	}
	g.fields = slices.Sorted(maps.Keys(set))
	return g
}

// tokensWhere collects the tokens satisfying pred. The token
// constants are not iterable from outside sdl/token, but the
// predicates are exported and the constants are small ints, so a
// generous scan finds every one without touching the package's
// numbering.
func tokensWhere(pred func(token.Token) bool) []token.Token {
	var toks []token.Token
	for tok := token.Token(0); tok < 256; tok++ {
		if pred(tok) {
			toks = append(toks, tok)
		}
	}
	return toks
}

// checkLiterals verifies that an emitter decided a rendering for
// every literal token the scanner produces; decided maps the token
// name to the emitter's rule for it, empty for a deliberate "stays
// plain text". A literal missing from the table fails generation
// loudly — the anti-drift stance of the whole verb — instead of
// silently leaving a new literal unstyled.
func checkLiterals(target string, decided map[string]string) error {
	for _, tok := range tokensWhere(token.Token.IsLiteral) {
		if _, ok := decided[tok.String()]; !ok {
			return fmt.Errorf("%s: no rendering decided for literal token %s", target, tok)
		}
	}
	return nil
}

// charClass renders lexemes as the body of a regular-expression
// character class, syntax shared between Vim's collections and
// TextMate's Oniguruma classes. Multi-character lexemes have no place
// in a class; a future one must teach the emitters its rendering
// before it can ride along.
func charClass(lexemes []string) (string, error) {
	var b strings.Builder
	for _, lex := range lexemes {
		if len(lex) != 1 {
			return "", fmt.Errorf("operator %q is not a single character; teach the emitters its rendering", lex)
		}
		if strings.ContainsAny(lex, `]\^-`) {
			b.WriteByte('\\')
		}
		b.WriteString(lex)
	}
	return b.String(), nil
}
