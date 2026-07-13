// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package highlightcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// vscodeLiterals records the TextMate rendering decided for every
// literal token: the repository rule that colors it, or "" where the
// literal deliberately stays plain text. checkLiterals holds the
// emitter to this table's completeness.
var vscodeLiterals = map[string]string{
	"IDENT":    "", // references and keys stay plain
	"STRING":   "string",
	"INT":      "int",
	"DURATION": "duration",
}

// A tmRule is one TextMate grammar rule: a named match, a begin/end
// region with inner patterns, a rule with captures, or an include
// reference into the repository.
type tmRule struct {
	Name     string            `json:"name,omitempty"`
	Match    string            `json:"match,omitempty"`
	Begin    string            `json:"begin,omitempty"`
	End      string            `json:"end,omitempty"`
	Captures map[string]tmName `json:"captures,omitempty"`
	Patterns []tmRule          `json:"patterns,omitempty"`
	Include  string            `json:"include,omitempty"`
}

// tmName names a scope inside a captures table.
type tmName struct {
	Name string `json:"name"`
}

// tmGrammar is the TextMate grammar document Visual Studio Code
// loads; the struct field order is the emission order.
type tmGrammar struct {
	Schema     string            `json:"$schema"`
	Info       []string          `json:"information_for_contributors"`
	Name       string            `json:"name"`
	ScopeName  string            `json:"scopeName"`
	FileTypes  []string          `json:"fileTypes"`
	Patterns   []tmRule          `json:"patterns"`
	Repository map[string]tmRule `json:"repository"`
}

// identRe is the ASCII approximation of the scanner's identifier rule
// (sdl/scanner accepts Unicode letters; a highlighter's regex keeps to
// the portable subset).
const identRe = `[A-Za-z_][A-Za-z0-9_]*`

// emitVSCode renders the grammar as a TextMate tmLanguage document in
// JSON. The word lists come straight from the grammar value; the
// literal shapes are this emitter's transcription of the scanner's
// lexical rules (sdl/scanner). Rules guard their own boundaries — the
// numeric shapes refuse word or dotted context on either side — so
// their order in the patterns list is cosmetic except for durations
// before integers, mirroring the scanner's longest-match rule.
func emitVSCode(w io.Writer, g grammar) error {
	if err := checkLiterals("vscode", vscodeLiterals); err != nil {
		return err
	}
	class, err := charClass(g.operators)
	if err != nil {
		return fmt.Errorf("vscode: %v", err)
	}

	repo := map[string]tmRule{
		"comment": {Name: "comment.line.double-slash.sdl", Match: `//.*$`},
		"string": {
			Name:  "string.quoted.double.sdl",
			Begin: `"`,
			End:   `"|$`, // a string never crosses its line, the scanner's rule
			Patterns: []tmRule{
				{Name: "constant.character.escape.sdl", Match: `\\(?:[abfnrtv\\"]|[0-7]{3}|x[0-9A-Fa-f]{2}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})`},
				{Name: "invalid.illegal.escape.sdl", Match: `\\.`},
			},
		},
		"duration": {Name: "constant.numeric.duration.sdl", Match: `(?<![\w.])[+-]?\d+[A-Za-z_.][A-Za-z0-9_.]*`},
		"int":      {Name: "constant.numeric.integer.sdl", Match: `(?<![\w.])[+-]?\d+(?![\w.])`},
	}
	patterns := []tmRule{{Include: "#comment"}, {Include: "#string"}}
	if len(g.sections) > 0 {
		// A section word may carry a dotted qualifier (with k8s.pod);
		// which sections require or refuse one is the linker's policy,
		// not the grammar's, so the shape is uniform here.
		repo["section"] = tmRule{
			Match: `\b(` + strings.Join(g.sections, `|`) + `)\b(?:\s+(` + identRe + `(?:\.` + identRe + `)*))?`,
			Captures: map[string]tmName{
				"1": {Name: "keyword.other.section.sdl"},
				"2": {Name: "entity.name.namespace.qualifier.sdl"},
			},
		}
		patterns = append(patterns, tmRule{Include: "#section"})
	}
	if len(g.keywords) > 0 {
		repo["keyword"] = tmRule{Name: "keyword.control.sdl", Match: word(g.keywords)}
		patterns = append(patterns, tmRule{Include: "#keyword"})
	}
	if len(g.booleans) > 0 {
		repo["boolean"] = tmRule{Name: "constant.language.boolean.sdl", Match: word(g.booleans)}
		patterns = append(patterns, tmRule{Include: "#boolean"})
	}
	if len(g.fields) > 0 {
		// The colon lookahead keeps the field words out of value
		// position; a params key spelling a field word still colors,
		// the approximation a regex highlighter affords.
		repo["field"] = tmRule{Name: "support.type.property-name.sdl", Match: word(g.fields) + `(?=\s*:)`}
		patterns = append(patterns, tmRule{Include: "#field"})
	}
	patterns = append(patterns, tmRule{Include: "#duration"}, tmRule{Include: "#int"})
	if class != "" {
		repo["punctuation"] = tmRule{Name: "punctuation.other.sdl", Match: `[` + class + `]`}
		patterns = append(patterns, tmRule{Include: "#punctuation"})
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // regexes carry < and > verbatim
	enc.SetIndent("", "  ")
	return enc.Encode(tmGrammar{
		Schema: "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json",
		Info: []string{
			"Generated by sdl highlight vscode from the toolchain's grammar code;",
			"regenerate instead of editing, or the file drifts from the grammar.",
		},
		Name:       "SDL",
		ScopeName:  "source.sdl",
		FileTypes:  []string{"sdl"},
		Patterns:   patterns,
		Repository: repo,
	})
}

// word renders words as a whole-word regular-expression alternation.
func word(words []string) string {
	return `\b(?:` + strings.Join(words, `|`) + `)\b`
}
