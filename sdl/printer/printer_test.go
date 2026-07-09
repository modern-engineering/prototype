// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package printer_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/printer"
)

var update = flag.Bool("update", false, "rewrite golden files from the observed output")

// parse parses src and fails the test on any error: the printer's
// contract only covers trees that parsed cleanly.
func parse(t *testing.T, name, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(name, []byte(src))
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", name, err)
	}
	return f
}

func mustSource(t *testing.T, name string, f *ast.File) []byte {
	t.Helper()
	out, err := printer.Source(f)
	if err != nil {
		t.Fatalf("Source(%s): %v", name, err)
	}
	return out
}

func inputFiles(t *testing.T) []string {
	t.Helper()
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.input"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no testdata inputs")
	}
	return inputs
}

// TestGolden pins the canonical form of the corpus: each testdata input
// prints to exactly its .golden neighbour.
func TestGolden(t *testing.T) {
	for _, input := range inputFiles(t) {
		name := strings.TrimSuffix(filepath.Base(input), ".input")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			got := mustSource(t, input, parse(t, input, string(src)))

			golden := strings.TrimSuffix(input, ".input") + ".golden"
			if *update {
				if err := os.WriteFile(golden, got, 0o666); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("canonical form differs from %s\n--- got ---\n%s", golden, got)
			}
		})
	}
}

// adversarialSources are clean-parsing sources built to stress the
// layout and comment rules beyond what the corpus covers.
var adversarialSources = []string{
	// A suffix on the opening brace AND one on the closing brace: the
	// second is displaced into the next statement's Before groups.
	"solution s\n\nimport ff \"example.com/ff\"\n\ndeploy ff.Ping as A { // on the brace\n\tcount: 1\n} // displaced\ndeploy ff.Pong as B\n",
	// The displaced closing-paren comment of a factored block.
	"solution s\n\nimport ( // on the paren\n\tff \"example.com/ff\"\n) // displaced to the next statement\n\ndeploy ff.Ping as A\n",
	// One-line everything.
	"solution s\nimport (ff \"example.com/ff\")\ndeploy (ff.Ping as A { count: 1 })\n",
	// Empty factored block and empty body.
	"solution s\n\nimport ff \"example.com/ff\"\n\nvar ()\n\ndeploy ff.Ping as A {}\n",
	// No trailing newline.
	"solution s\n\nimport ff \"example.com/ff\"\n\ndeploy ff.Ping as A",
	// Comment groups only, after the clause.
	"solution s\n\n// alpha\n\n// beta\n// beta continued\n",
	// Excess blank lines everywhere.
	"solution s\n\n\n\n\nimport ff \"example.com/ff\"\n\n\n\ndeploy ff.Ping as A {\n\n\tcount: 1\n\n\n\ttarget: \"x\"\n\n}\n",
}

// sources returns every corpus file plus the adversarial cases, keyed
// for subtests.
func sources(t *testing.T) map[string]string {
	t.Helper()
	all := make(map[string]string)
	for _, input := range inputFiles(t) {
		src, err := os.ReadFile(input)
		if err != nil {
			t.Fatal(err)
		}
		all[filepath.Base(input)] = string(src)
	}
	for i, src := range adversarialSources {
		all[fmt.Sprintf("adversarial%d", i)] = src
	}
	return all
}

// TestIdempotent is the fixed point property: printing a printed unit
// changes nothing.
func TestIdempotent(t *testing.T) {
	for name, src := range sources(t) {
		t.Run(name, func(t *testing.T) {
			once := mustSource(t, name, parse(t, name, src))
			twice := mustSource(t, name, parse(t, name, string(once)))
			if !bytes.Equal(once, twice) {
				t.Errorf("printing is not idempotent\n--- first ---\n%s--- second ---\n%s", once, twice)
			}
		})
	}
}

// TestReparse is the identity property: the canonical form parses back
// into a structurally equal tree — same statements, same lexemes, same
// comment texts in the same slots.
func TestReparse(t *testing.T) {
	for name, src := range sources(t) {
		t.Run(name, func(t *testing.T) {
			before := parse(t, name, src)
			out := mustSource(t, name, before)
			after, err := parser.ParseFile(name, out)
			if err != nil {
				t.Fatalf("canonical form does not parse: %v\n--- printed ---\n%s", err, out)
			}
			if err := cmpFile(before, after); err != nil {
				t.Errorf("reparse differs: %v\n--- printed ---\n%s", err, out)
			}
		})
	}
}

// adversarialStrings stresses string canonicalization: quotes,
// backslashes, control characters, and non-ASCII text.
var adversarialStrings = []string{
	"",
	`plain`,
	`say "hi"`,
	`back\slash`,
	"tab\tand\nnewline",
	"carriage\rreturn",
	"nul\x00byte",
	"π ≠ 3.14, 中文, emoji 🎄",
	`mixed "quotes" and \backslashes\ and
line breaks`,
}

// TestStringRoundTrip drives adversarial string values through both
// printer paths: an authored literal keeps its lexeme, and a
// synthesized literal (no lexeme, the echo path) canonicalizes through
// strconv.Quote — both must parse back to the same value.
func TestStringRoundTrip(t *testing.T) {
	for i, s := range adversarialStrings {
		name := fmt.Sprintf("string%d", i)
		t.Run("authored/"+name, func(t *testing.T) {
			src := "solution s\n\nvar v: " + strconv.Quote(s) + "\n"
			before := parse(t, name, src)
			out := mustSource(t, name, before)
			after := parse(t, name, string(out))
			if err := cmpFile(before, after); err != nil {
				t.Fatalf("reparse differs: %v\n--- printed ---\n%s", err, out)
			}
			if got := varValue(t, after); got != s {
				t.Errorf("value round-trip: got %q, want %q", got, s)
			}
		})
		t.Run("synthesized/"+name, func(t *testing.T) {
			f := &ast.File{
				Solution: &ast.SolutionClause{Name: &ast.Ident{Name: "s"}},
				Decls: []ast.Decl{&ast.VarDecl{Specs: []*ast.VarSpec{{
					Name:  &ast.Ident{Name: "v"},
					Value: &ast.StringLit{Value: s},
				}}}},
			}
			out := mustSource(t, name, f)
			after := parse(t, name, string(out))
			if got := varValue(t, after); got != s {
				t.Errorf("value round-trip: got %q, want %q\n--- printed ---\n%s", got, s, out)
			}
		})
	}
}

// varValue extracts the sole var declaration's string value.
func varValue(t *testing.T, f *ast.File) string {
	t.Helper()
	for _, d := range f.Decls {
		if v, ok := d.(*ast.VarDecl); ok {
			lit, ok := v.Specs[0].Value.(*ast.StringLit)
			if !ok {
				t.Fatalf("var value is %T, want *ast.StringLit", v.Specs[0].Value)
			}
			return lit.Value
		}
	}
	t.Fatal("no var declaration in the reparsed file")
	return ""
}

// TestSynthesized pins the layout of position-free trees, the shape sdl
// echo builds: blank lines between top-level statements, snug block
// items, factored rendering for multi-spec declarations without a
// recorded paren, and canonical lexemes for every value kind.
func TestSynthesized(t *testing.T) {
	f := &ast.File{
		Solution: &ast.SolutionClause{Name: &ast.Ident{Name: "synth"}},
		Imports: []*ast.ImportDecl{{Specs: []*ast.ImportSpec{
			{Alias: &ast.Ident{Name: "ff"}, Path: &ast.StringLit{Value: "example.com/ff-go"}},
			{Path: &ast.StringLit{Value: "example.com/util"}},
		}}},
		Decls: []ast.Decl{
			&ast.DeployDecl{Specs: []*ast.DeploySpec{{
				Type: &ast.TypeRef{Pkg: &ast.Ident{Name: "ff"}, Name: &ast.Ident{Name: "Ping"}},
				Name: &ast.Ident{Name: "P1"},
				Body: &ast.Body{Items: []ast.BodyItem{
					&ast.Param{Key: &ast.Ident{Name: "count"}, Value: &ast.IntLit{Value: -1}},
					&ast.Param{Key: &ast.Ident{Name: "interval"}, Value: &ast.DurationLit{Value: 90 * time.Minute}},
					&ast.Param{Key: &ast.Ident{Name: "loud"}, Value: &ast.BoolLit{Value: true}},
					&ast.Param{Key: &ast.Ident{Name: "target"}, Value: &ast.StringLit{Value: `say "hi"`}},
				}},
			}}},
			&ast.DeployDecl{Specs: []*ast.DeploySpec{{
				Type: &ast.TypeRef{Pkg: &ast.Ident{Name: "util"}, Name: &ast.Ident{Name: "Pong"}},
				Name: &ast.Ident{Name: "P2"},
			}}},
		},
	}
	want := `solution synth

import (
	ff "example.com/ff-go"
	"example.com/util"
)

deploy ff.Ping as P1 {
	count: -1
	interval: 1h30m0s
	loud: true
	target: "say \"hi\""
}

deploy util.Pong as P2
`
	got := mustSource(t, "synth", f)
	if string(got) != want {
		t.Errorf("synthesized layout:\n--- got ---\n%s--- want ---\n%s", got, want)
	}
	// The synthesized form is already canonical: parsing and reprinting
	// it changes nothing.
	again := mustSource(t, "synth", parse(t, "synth", string(got)))
	if !bytes.Equal(got, again) {
		t.Errorf("synthesized output is not canonical\n--- reprinted ---\n%s", again)
	}
}

// TestErrors covers the structural holes the printer refuses: trees no
// clean parse produces.
func TestErrors(t *testing.T) {
	sol := &ast.SolutionClause{Name: &ast.Ident{Name: "s"}}
	cases := map[string]*ast.File{
		"NilFile":    nil,
		"NoSolution": {},
		"EmptyDecl":  {Solution: sol, Decls: []ast.Decl{&ast.DeployDecl{}}},
		"NilValue":   {Solution: sol, Decls: []ast.Decl{&ast.VarDecl{Specs: []*ast.VarSpec{{Name: &ast.Ident{Name: "v"}}}}}},
		"BadValue":   {Solution: sol, Decls: []ast.Decl{&ast.VarDecl{Specs: []*ast.VarSpec{{Name: &ast.Ident{Name: "v"}, Value: &ast.BadValue{}}}}}},
		"NoBody":     {Solution: sol, Decls: []ast.Decl{&ast.DefaultDecl{Target: &ast.Ident{Name: "deploy"}}}},
		"NoImportPath": {Solution: sol, Imports: []*ast.ImportDecl{{Specs: []*ast.ImportSpec{{
			Alias: &ast.Ident{Name: "ff"},
		}}}}},
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := printer.Source(f); err == nil {
				t.Error("Source succeeded, want an error")
			}
		})
	}
}

// ----------------------------------------------------------------------------
// Structural comparison
//
// cmpFile reports the first difference between two syntax trees that
// canonical printing promises to preserve: statement structure, lexemes
// (Raw text), values, and comment texts in their slots. Positions are
// the one thing printing legitimately changes, so they are ignored.

func cmpFile(a, b *ast.File) error {
	if err := cmpComments("solution", &a.Solution.Comments, &b.Solution.Comments); err != nil {
		return err
	}
	if a.Solution.Name.Name != b.Solution.Name.Name {
		return fmt.Errorf("solution: %q != %q", a.Solution.Name.Name, b.Solution.Name.Name)
	}
	if len(a.Imports) != len(b.Imports) {
		return fmt.Errorf("imports: %d != %d declarations", len(a.Imports), len(b.Imports))
	}
	for i := range a.Imports {
		if err := cmpImportDecl(fmt.Sprintf("import[%d]", i), a.Imports[i], b.Imports[i]); err != nil {
			return err
		}
	}
	if len(a.Decls) != len(b.Decls) {
		return fmt.Errorf("decls: %d != %d declarations", len(a.Decls), len(b.Decls))
	}
	for i := range a.Decls {
		if err := cmpDecl(fmt.Sprintf("decl[%d]", i), a.Decls[i], b.Decls[i]); err != nil {
			return err
		}
	}
	return cmpGroups("file after", a.After, b.After)
}

func cmpImportDecl(at string, a, b *ast.ImportDecl) error {
	if err := cmpGroupShape(at, &a.Comments, a.Lparen.IsValid(), len(a.Specs), a.After, &b.Comments, b.Lparen.IsValid(), len(b.Specs), b.After); err != nil {
		return err
	}
	for i := range a.Specs {
		as, bs := a.Specs[i], b.Specs[i]
		at := fmt.Sprintf("%s spec[%d]", at, i)
		if err := cmpComments(at, &as.Comments, &bs.Comments); err != nil {
			return err
		}
		if err := cmpIdent(at+" alias", as.Alias, bs.Alias); err != nil {
			return err
		}
		if as.Path.Raw != bs.Path.Raw || as.Path.Value != bs.Path.Value {
			return fmt.Errorf("%s path: %q != %q", at, as.Path.Raw, bs.Path.Raw)
		}
	}
	return nil
}

func cmpDecl(at string, a, b ast.Decl) error {
	switch a := a.(type) {
	case *ast.ExternDecl:
		b, ok := b.(*ast.ExternDecl)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpGroupShape(at, &a.Comments, a.Lparen.IsValid(), len(a.Specs), a.After, &b.Comments, b.Lparen.IsValid(), len(b.Specs), b.After); err != nil {
			return err
		}
		for i := range a.Specs {
			as, bs := a.Specs[i], b.Specs[i]
			at := fmt.Sprintf("%s spec[%d]", at, i)
			if err := cmpComments(at, &as.Comments, &bs.Comments); err != nil {
				return err
			}
			if err := cmpIdent(at+" name", as.Name, bs.Name); err != nil {
				return err
			}
			if err := cmpTypeRef(at+" type", as.Type, bs.Type); err != nil {
				return err
			}
		}
	case *ast.VarDecl:
		b, ok := b.(*ast.VarDecl)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpGroupShape(at, &a.Comments, a.Lparen.IsValid(), len(a.Specs), a.After, &b.Comments, b.Lparen.IsValid(), len(b.Specs), b.After); err != nil {
			return err
		}
		for i := range a.Specs {
			as, bs := a.Specs[i], b.Specs[i]
			at := fmt.Sprintf("%s spec[%d]", at, i)
			if err := cmpComments(at, &as.Comments, &bs.Comments); err != nil {
				return err
			}
			if err := cmpIdent(at+" name", as.Name, bs.Name); err != nil {
				return err
			}
			if err := cmpValue(at+" value", as.Value, bs.Value); err != nil {
				return err
			}
		}
	case *ast.DefaultDecl:
		b, ok := b.(*ast.DefaultDecl)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpComments(at, &a.Comments, &b.Comments); err != nil {
			return err
		}
		switch target := a.Target.(type) {
		case *ast.TypeRef:
			bt, ok := b.Target.(*ast.TypeRef)
			if !ok {
				return fmt.Errorf("%s target: %T != %T", at, a.Target, b.Target)
			}
			if err := cmpTypeRef(at+" target", target, bt); err != nil {
				return err
			}
		case *ast.Ident:
			bt, ok := b.Target.(*ast.Ident)
			if !ok {
				return fmt.Errorf("%s target: %T != %T", at, a.Target, b.Target)
			}
			if err := cmpIdent(at+" target", target, bt); err != nil {
				return err
			}
		}
		if err := cmpBody(at+" body", a.Body, b.Body); err != nil {
			return err
		}
	case *ast.DeployDecl:
		b, ok := b.(*ast.DeployDecl)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpGroupShape(at, &a.Comments, a.Lparen.IsValid(), len(a.Specs), a.After, &b.Comments, b.Lparen.IsValid(), len(b.Specs), b.After); err != nil {
			return err
		}
		for i := range a.Specs {
			as, bs := a.Specs[i], b.Specs[i]
			at := fmt.Sprintf("%s spec[%d]", at, i)
			if err := cmpComments(at, &as.Comments, &bs.Comments); err != nil {
				return err
			}
			if err := cmpTypeRef(at+" type", as.Type, bs.Type); err != nil {
				return err
			}
			if err := cmpIdent(at+" name", as.Name, bs.Name); err != nil {
				return err
			}
			if err := cmpBody(at+" body", as.Body, bs.Body); err != nil {
				return err
			}
		}
	case *ast.ProvisionDecl:
		b, ok := b.(*ast.ProvisionDecl)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpGroupShape(at, &a.Comments, a.Lparen.IsValid(), len(a.Specs), a.After, &b.Comments, b.Lparen.IsValid(), len(b.Specs), b.After); err != nil {
			return err
		}
		for i := range a.Specs {
			as, bs := a.Specs[i], b.Specs[i]
			at := fmt.Sprintf("%s spec[%d]", at, i)
			if err := cmpComments(at, &as.Comments, &bs.Comments); err != nil {
				return err
			}
			if err := cmpTypeRef(at+" type", as.Type, bs.Type); err != nil {
				return err
			}
			if err := cmpIdent(at+" kind", as.Kind, bs.Kind); err != nil {
				return err
			}
			if err := cmpIdent(at+" name", as.Name, bs.Name); err != nil {
				return err
			}
			if err := cmpBody(at+" body", as.Body, bs.Body); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("%s: unexpected declaration %T", at, a)
	}
	return nil
}

// cmpGroupShape compares the declaration-level parts shared by every
// single-or-factored declaration.
func cmpGroupShape(at string, ac *ast.Comments, afactored bool, an int, aafter []*ast.CommentGroup, bc *ast.Comments, bfactored bool, bn int, bafter []*ast.CommentGroup) error {
	if err := cmpComments(at, ac, bc); err != nil {
		return err
	}
	if afactored != bfactored {
		return fmt.Errorf("%s: factored %v != %v", at, afactored, bfactored)
	}
	if an != bn {
		return fmt.Errorf("%s: %d != %d specs", at, an, bn)
	}
	return cmpGroups(at+" after", aafter, bafter)
}

func cmpBody(at string, a, b *ast.Body) error {
	if (a == nil) != (b == nil) {
		return fmt.Errorf("%s: presence %v != %v", at, a != nil, b != nil)
	}
	if a == nil {
		return nil
	}
	if len(a.Items) != len(b.Items) {
		return fmt.Errorf("%s: %d != %d items", at, len(a.Items), len(b.Items))
	}
	for i := range a.Items {
		at := fmt.Sprintf("%s item[%d]", at, i)
		switch ai := a.Items[i].(type) {
		case *ast.Param:
			bi, ok := b.Items[i].(*ast.Param)
			if !ok {
				return fmt.Errorf("%s: %T != %T", at, a.Items[i], b.Items[i])
			}
			if err := cmpComments(at, &ai.Comments, &bi.Comments); err != nil {
				return err
			}
			if err := cmpIdent(at+" key", ai.Key, bi.Key); err != nil {
				return err
			}
			if err := cmpValue(at+" value", ai.Value, bi.Value); err != nil {
				return err
			}
		case *ast.Section:
			bi, ok := b.Items[i].(*ast.Section)
			if !ok {
				return fmt.Errorf("%s: %T != %T", at, a.Items[i], b.Items[i])
			}
			if err := cmpComments(at, &ai.Comments, &bi.Comments); err != nil {
				return err
			}
			if err := cmpIdent(at+" name", ai.Name, bi.Name); err != nil {
				return err
			}
			if err := cmpBody(at, ai.Body, bi.Body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: unexpected item %T", at, a.Items[i])
		}
	}
	return cmpGroups(at+" after", a.After, b.After)
}

func cmpValue(at string, a, b ast.Value) error {
	switch a := a.(type) {
	case *ast.StringLit:
		b, ok := b.(*ast.StringLit)
		if !ok || a.Raw != b.Raw || a.Value != b.Value {
			return fmt.Errorf("%s: string %#v != %#v", at, a, b)
		}
	case *ast.IntLit:
		b, ok := b.(*ast.IntLit)
		if !ok || a.Raw != b.Raw || a.Value != b.Value {
			return fmt.Errorf("%s: int %#v != %#v", at, a, b)
		}
	case *ast.DurationLit:
		b, ok := b.(*ast.DurationLit)
		if !ok || a.Raw != b.Raw || a.Value != b.Value {
			return fmt.Errorf("%s: duration %#v != %#v", at, a, b)
		}
	case *ast.BoolLit:
		b, ok := b.(*ast.BoolLit)
		if !ok || a.Value != b.Value {
			return fmt.Errorf("%s: bool %#v != %#v", at, a, b)
		}
	case *ast.RefExpr:
		b, ok := b.(*ast.RefExpr)
		if !ok {
			return fmt.Errorf("%s: %T != %T", at, a, b)
		}
		if err := cmpIdent(at+" symbol", a.X, b.X); err != nil {
			return err
		}
		if err := cmpIdent(at+" selector", a.Sel, b.Sel); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s: unexpected value %T", at, a)
	}
	return nil
}

func cmpTypeRef(at string, a, b *ast.TypeRef) error {
	if (a == nil) != (b == nil) {
		return fmt.Errorf("%s: presence %v != %v", at, a != nil, b != nil)
	}
	if a == nil {
		return nil
	}
	if err := cmpIdent(at+" package", a.Pkg, b.Pkg); err != nil {
		return err
	}
	return cmpIdent(at+" name", a.Name, b.Name)
}

func cmpIdent(at string, a, b *ast.Ident) error {
	switch {
	case (a == nil) != (b == nil):
		return fmt.Errorf("%s: presence %v != %v", at, a != nil, b != nil)
	case a != nil && a.Name != b.Name:
		return fmt.Errorf("%s: %q != %q", at, a.Name, b.Name)
	}
	return nil
}

func cmpComments(at string, a, b *ast.Comments) error {
	if err := cmpGroups(at+" before", a.Before, b.Before); err != nil {
		return err
	}
	switch {
	case (a.Suffix == nil) != (b.Suffix == nil):
		return fmt.Errorf("%s suffix: presence %v != %v", at, a.Suffix != nil, b.Suffix != nil)
	case a.Suffix != nil && a.Suffix.Text != b.Suffix.Text:
		return fmt.Errorf("%s suffix: %q != %q", at, a.Suffix.Text, b.Suffix.Text)
	}
	return nil
}

func cmpGroups(at string, a, b []*ast.CommentGroup) error {
	if len(a) != len(b) {
		return fmt.Errorf("%s: %d != %d comment groups", at, len(a), len(b))
	}
	for i := range a {
		if len(a[i].List) != len(b[i].List) {
			return fmt.Errorf("%s group[%d]: %d != %d comments", at, i, len(a[i].List), len(b[i].List))
		}
		for j := range a[i].List {
			if a[i].List[j].Text != b[i].List[j].Text {
				return fmt.Errorf("%s group[%d][%d]: %q != %q", at, i, j, a[i].List[j].Text, b[i].List[j].Text)
			}
		}
	}
	return nil
}
