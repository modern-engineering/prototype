// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Open door, recorded but not built: an in-fixture error corpus in the
// go/parser (/* ERROR "rx" */) and analysistest (// want) tradition —
// testdata fixtures carrying each expected diagnostic beside the
// offending line, so positions are checked by eye where today
// TestParseErrors hand-counts line:col into a table of backtick
// strings. SDL has line comments only, so the marker would anchor a
// line and the column needs its own convention (say, a quoted lexeme
// the harness locates on that line). Reopening trigger: a grammar
// change that forces re-counting positions across many table rows, or
// the error table outgrowing at-a-glance review (~17 rows today).

package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
)

// parse parses src and fails the test on any error.
func parse(t *testing.T, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile("test.sdl", []byte(src))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return f
}

// parseErrs parses src, requires it to fail, and returns the sorted
// error list.
func parseErrs(t *testing.T, src string) scanner.ErrorList {
	t.Helper()
	_, err := parser.ParseFile("test.sdl", []byte(src))
	if err == nil {
		t.Fatalf("ParseFile succeeded, want errors\nsource:\n%s", src)
	}
	errs, ok := err.(scanner.ErrorList)
	if !ok {
		t.Fatalf("ParseFile error is %T, want scanner.ErrorList", err)
	}
	return errs
}

func lineCol(pos token.Position) string {
	return fmt.Sprintf("%d:%d", pos.Line, pos.Column)
}

func TestParseDeploySubset(t *testing.T) {
	f := parse(t, `solution sample

import ff "example.com/ff"

deploy ff.Ping as Ping1 {
	interval: 1s
	count: -1
	target: "com.acme.Echo"
	enabled: true
	nats: natsAccount.config
	subject: echoSubject
}
`)
	if f.Solution == nil || f.Solution.Name.Name != "sample" {
		t.Fatalf("solution clause: got %+v", f.Solution)
	}
	if len(f.Imports) != 1 {
		t.Fatalf("imports: got %d, want 1", len(f.Imports))
	}
	imp := f.Imports[0]
	if imp.Lparen.IsValid() {
		t.Error("single import parsed as factored")
	}
	spec := imp.Specs[0]
	if spec.Alias == nil || spec.Alias.Name != "ff" || spec.Path.Value != "example.com/ff" {
		t.Errorf("import spec: got alias %v path %+v", spec.Alias, spec.Path)
	}
	if len(f.Decls) != 1 {
		t.Fatalf("decls: got %d, want 1", len(f.Decls))
	}
	dep, ok := f.Decls[0].(*ast.DeployDecl)
	if !ok {
		t.Fatalf("decl 0 is %T, want *ast.DeployDecl", f.Decls[0])
	}
	ds := dep.Specs[0]
	if ds.Type.Pkg.Name != "ff" || ds.Type.Name.Name != "Ping" || ds.Name.Name != "Ping1" {
		t.Errorf("deploy spec: type %v.%v as %v", ds.Type.Pkg, ds.Type.Name, ds.Name)
	}
	if len(ds.Body.Items) != 6 {
		t.Fatalf("body items: got %d, want 6", len(ds.Body.Items))
	}
	params := make(map[string]ast.Value)
	for _, item := range ds.Body.Items {
		p := item.(*ast.Param)
		params[p.Key.Name] = p.Value
	}
	if v := params["interval"].(*ast.DurationLit); v.Value != time.Second || v.Raw != "1s" {
		t.Errorf("interval: got %v (%q)", v.Value, v.Raw)
	}
	if v := params["count"].(*ast.IntLit); v.Value != -1 {
		t.Errorf("count: got %d", v.Value)
	}
	if v := params["target"].(*ast.StringLit); v.Value != "com.acme.Echo" || v.Raw != `"com.acme.Echo"` {
		t.Errorf("target: got %q (%q)", v.Value, v.Raw)
	}
	if v := params["enabled"].(*ast.BoolLit); v.Value != true {
		t.Errorf("enabled: got %v", v.Value)
	}
	if v := params["nats"].(*ast.RefExpr); v.X.Name != "natsAccount" || v.Sel == nil || v.Sel.Name != "config" {
		t.Errorf("nats: got %v.%v", v.X, v.Sel)
	}
	if v := params["subject"].(*ast.RefExpr); v.X.Name != "echoSubject" || v.Sel != nil {
		t.Errorf("subject: got %v.%v", v.X, v.Sel)
	}
}

func TestParseFactoredImportAndDeploy(t *testing.T) {
	f := parse(t, `solution sample

import (
	ff "example.com/ff"
	"example.com/util"
)

deploy (
	ff.Ping as P1 {
		count: 1
	}
	ff.Pong as P2
)
`)
	imp := f.Imports[0]
	if !imp.Lparen.IsValid() || !imp.Rparen.IsValid() {
		t.Fatal("factored import lost its parens")
	}
	if len(imp.Specs) != 2 {
		t.Fatalf("import specs: got %d, want 2", len(imp.Specs))
	}
	if imp.Specs[0].Alias == nil || imp.Specs[0].Alias.Name != "ff" {
		t.Errorf("spec 0 alias: got %v", imp.Specs[0].Alias)
	}
	if imp.Specs[1].Alias != nil || imp.Specs[1].Path.Value != "example.com/util" {
		t.Errorf("spec 1: got alias %v path %+v", imp.Specs[1].Alias, imp.Specs[1].Path)
	}
	dep := f.Decls[0].(*ast.DeployDecl)
	if !dep.Lparen.IsValid() {
		t.Fatal("factored deploy lost its parens")
	}
	if len(dep.Specs) != 2 {
		t.Fatalf("deploy specs: got %d, want 2", len(dep.Specs))
	}
	p1, p2 := dep.Specs[0], dep.Specs[1]
	if p1.Name.Name != "P1" || p1.Body == nil || len(p1.Body.Items) != 1 {
		t.Errorf("P1: name %v body %+v", p1.Name, p1.Body)
	}
	if p2.Name.Name != "P2" || p2.Body != nil {
		t.Errorf("P2: name %v body %+v", p2.Name, p2.Body)
	}
}

func TestParseFullGrammar(t *testing.T) {
	f := parse(t, `solution sample

extern (
	natsAdmin nats.Secret
)

var (
	echoSubject: "com.acme.Echo"
	pingInterval: 1s
)

default deploy {
	on {
		location: awsUsEast1
	}
}

default nats.NATS {
	x: 1
}

provision nats.NATS slice as acct {
	admin: natsAdmin
}

provision pg.Postgres as legacy
`)
	if len(f.Decls) != 6 {
		t.Fatalf("decls: got %d, want 6", len(f.Decls))
	}

	ext := f.Decls[0].(*ast.ExternDecl)
	es := ext.Specs[0]
	if es.Name.Name != "natsAdmin" || es.Type.Pkg.Name != "nats" || es.Type.Name.Name != "Secret" {
		t.Errorf("extern spec: %v %v.%v", es.Name, es.Type.Pkg, es.Type.Name)
	}

	vd := f.Decls[1].(*ast.VarDecl)
	if len(vd.Specs) != 2 {
		t.Fatalf("var specs: got %d, want 2", len(vd.Specs))
	}
	if v := vd.Specs[0].Value.(*ast.StringLit); v.Value != "com.acme.Echo" {
		t.Errorf("echoSubject: got %q", v.Value)
	}
	if v := vd.Specs[1].Value.(*ast.DurationLit); v.Value != time.Second {
		t.Errorf("pingInterval: got %v", v.Value)
	}

	dv := f.Decls[2].(*ast.DefaultDecl)
	if target, ok := dv.Target.(*ast.Ident); !ok || target.Name != "deploy" {
		t.Errorf("default verb target: got %#v", dv.Target)
	}
	on := dv.Body.Items[0].(*ast.Section)
	if on.Name.Name != "on" || len(on.Body.Items) != 1 {
		t.Fatalf("on section: %v with %d items", on.Name, len(on.Body.Items))
	}
	loc := on.Body.Items[0].(*ast.Param)
	if ref, ok := loc.Value.(*ast.RefExpr); !ok || ref.X.Name != "awsUsEast1" || ref.Sel != nil {
		t.Errorf("location value: got %#v", loc.Value)
	}

	dt := f.Decls[3].(*ast.DefaultDecl)
	if target, ok := dt.Target.(*ast.TypeRef); !ok || target.Pkg.Name != "nats" || target.Name.Name != "NATS" {
		t.Errorf("default type target: got %#v", dt.Target)
	}

	p1 := f.Decls[4].(*ast.ProvisionDecl).Specs[0]
	if p1.Kind == nil || p1.Kind.Name != "slice" || p1.Name.Name != "acct" {
		t.Errorf("provision 1: kind %v name %v", p1.Kind, p1.Name)
	}

	p2 := f.Decls[5].(*ast.ProvisionDecl).Specs[0]
	if p2.Kind != nil || p2.Name.Name != "legacy" || p2.Body != nil {
		t.Errorf("provision 2: kind %v name %v body %v", p2.Kind, p2.Name, p2.Body)
	}
}

// Dotted segments join into one parameter-key string anchored at the
// first segment, inside instance bodies and section bodies alike: the
// composite-key production.
func TestParseDottedKeys(t *testing.T) {
	f := parse(t, `solution s

deploy ff.Ping as P {
	retry.max: 3
	retry.backoff.base: 250ms
	on {
		zone.primary: "eu"
	}
}
`)
	body := f.Decls[0].(*ast.DeployDecl).Specs[0].Body
	if len(body.Items) != 3 {
		t.Fatalf("body items: got %d, want 3", len(body.Items))
	}
	max := body.Items[0].(*ast.Param)
	if max.Key.Name != "retry.max" || max.Value.(*ast.IntLit).Value != 3 {
		t.Errorf("item 0: key %q value %+v", max.Key.Name, max.Value)
	}
	if got := lineCol(max.Key.NamePos); got != "4:2" {
		t.Errorf("dotted key position: got %s, want 4:2 (the first segment)", got)
	}
	base := body.Items[1].(*ast.Param)
	if base.Key.Name != "retry.backoff.base" || base.Value.(*ast.DurationLit).Value != 250*time.Millisecond {
		t.Errorf("item 1: key %q value %+v", base.Key.Name, base.Value)
	}
	on := body.Items[2].(*ast.Section)
	zone := on.Body.Items[0].(*ast.Param)
	if zone.Key.Name != "zone.primary" {
		t.Errorf("section key: got %q", zone.Key.Name)
	}
}

// Any section head may carry an optional dotted qualifier, in instance
// and default bodies alike and at any nesting depth, joined into a
// single identifier like a dotted key. Which words admit one is the
// linker's business, so two "with" sections in one body are no parse
// error.
func TestParseQualifiedSections(t *testing.T) {
	f := parse(t, `solution s

default deploy {
	location: awsUsEast1

	with k8s.pod {
		priorityClass: standard
	}
}

deploy ff.Ping as P {
	params {
		count: 1
	}
	with k8s.pod {
		replicas: 3
		grid c.d {
			zone.primary: "eu"
		}
	}
	with k8s.workload {
		kind: batch
	}
}
`)
	def := f.Decls[0].(*ast.DefaultDecl)
	if len(def.Body.Items) != 2 {
		t.Fatalf("default body items: got %d, want 2", len(def.Body.Items))
	}
	sec := def.Body.Items[1].(*ast.Section)
	if sec.Name.Name != "with" || sec.Qualifier == nil || sec.Qualifier.Name != "k8s.pod" {
		t.Errorf("default section: name %v qualifier %v", sec.Name, sec.Qualifier)
	}
	if got := lineCol(sec.Name.NamePos); got != "6:2" {
		t.Errorf("section position: got %s, want 6:2 (the section name)", got)
	}
	if got := lineCol(sec.Qualifier.NamePos); got != "6:7" {
		t.Errorf("qualifier position: got %s, want 6:7 (the first segment)", got)
	}

	body := f.Decls[1].(*ast.DeployDecl).Specs[0].Body
	if len(body.Items) != 3 {
		t.Fatalf("deploy body items: got %d, want 3", len(body.Items))
	}
	params := body.Items[0].(*ast.Section)
	if params.Name.Name != "params" || params.Qualifier != nil {
		t.Errorf("params section: name %v qualifier %v", params.Name, params.Qualifier)
	}
	pod := body.Items[1].(*ast.Section)
	if pod.Qualifier == nil || pod.Qualifier.Name != "k8s.pod" {
		t.Errorf("first with section: qualifier %v", pod.Qualifier)
	}
	grid := pod.Body.Items[1].(*ast.Section)
	if grid.Name.Name != "grid" || grid.Qualifier == nil || grid.Qualifier.Name != "c.d" {
		t.Errorf("nested section: name %v qualifier %v", grid.Name, grid.Qualifier)
	}
	if key := grid.Body.Items[0].(*ast.Param).Key; key.Name != "zone.primary" {
		t.Errorf("nested dotted key: got %q", key.Name)
	}
	workload := body.Items[2].(*ast.Section)
	if workload.Qualifier == nil || workload.Qualifier.Name != "k8s.workload" {
		t.Errorf("second with section: qualifier %v", workload.Qualifier)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		pos  string
		msg  string
	}{
		{
			"duplicate parameter key",
			"solution s\n\ndeploy ff.Ping as P {\n\tcount: 1\n\tcount: 2\n}\n",
			"5:2",
			"duplicate parameter key count",
		},
		{
			// A non-empty unit must open with its solution clause; only
			// empty and comment-only units parse without one (they are
			// the linker's to diagnose).
			"missing solution clause",
			"deploy ff.Ping as P\n",
			"1:1",
			"expected 'solution', found deploy",
		},
		{
			"duplicate dotted parameter key",
			"solution s\n\ndeploy ff.Ping as P {\n\tretry.max: 3\n\tretry.max: 4\n}\n",
			"5:2",
			"duplicate parameter key retry.max",
		},
		{
			"dotted section name",
			"solution s\n\ndeploy ff.Ping as P {\n\ta.b {\n\t}\n}\n",
			"4:6",
			"expected ':', found '{'",
		},
		{
			"key ending in a dot",
			"solution s\n\ndeploy ff.Ping as P {\n\ta.: 1\n}\n",
			"4:4",
			"expected identifier, found ':'",
		},
		{
			// A dotted name can only be a parameter key, qualifier or
			// not; sections stay disambiguated by their plain first
			// identifier.
			"dotted section name with qualifier",
			"solution s\n\ndeploy ff.Ping as P {\n\ta.b q {\n\t}\n}\n",
			"4:6",
			"expected ':', found q",
		},
		{
			"qualifier without body",
			"solution s\n\ndeploy ff.Ping as P {\n\twith k8s.pod\n}\n",
			"4:14",
			"expected '{', found newline",
		},
		{
			"qualifier ending in a dot",
			"solution s\n\ndeploy ff.Ping as P {\n\twith k8s. {\n\t}\n}\n",
			"4:12",
			"expected identifier, found '{'",
		},
		{
			"missing as",
			"solution s\n\ndeploy ff.Ping P {\n}\n",
			"3:16",
			"expected 'as', found P",
		},
		{
			"missing colon",
			"solution s\n\ndeploy ff.Ping as P {\n\ttarget \"x\"\n}\n",
			"4:9",
			`expected ':' or '{', found "x"`,
		},
		{
			"unqualified type reference",
			"solution s\n\nimport ff \"example.com/ff\"\n\ndeploy Ping as P {\n}\n",
			"5:8",
			"unqualified type reference Ping: element references must be package-qualified through an import (e.g. pkg.Ping)",
		},
		{
			"var value must be literal",
			"solution s\n\nvar x: someRef\n",
			"3:8",
			"var value must be a literal (string, int, duration, or bool)",
		},
		{
			// The scanner holds source bytes to UTF-8; escapes must not
			// smuggle invalid bytes into the value.
			"non-UTF-8 string value",
			"solution s\n\ndeploy ff.Ping as P {\n\ttarget: \"\\xff\"\n}\n",
			"4:10",
			"string literal is not valid UTF-8",
		},
		{
			"non-UTF-8 import path",
			"solution s\n\nimport ff \"\\xff\"\n",
			"3:11",
			"string literal is not valid UTF-8",
		},
		{
			"invalid duration",
			"solution s\n\ndeploy ff.Ping as P {\n\tinterval: 10x\n}\n",
			"4:12",
			"invalid duration literal 10x",
		},
		{
			"import after declarations",
			"solution s\n\ndeploy ff.Ping as P\n\nimport ff \"x\"\n",
			"5:1",
			"import declarations must precede other declarations",
		},
		{
			"duplicate solution clause",
			"solution a\nsolution b\n",
			"2:1",
			"duplicate solution clause (one per unit)",
		},
		{
			"factored default",
			"solution s\n\ndefault (\n\tx: 1\n)\n",
			"3:9",
			"default does not take a factored block",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := parseErrs(t, tt.src)
			if len(errs) != 1 {
				t.Errorf("error count: got %d, want 1\n%v", len(errs), errs)
			}
			e := errs[0]
			if got := lineCol(e.Pos); got != tt.pos {
				t.Errorf("position: got %s, want %s", got, tt.pos)
			}
			if e.Msg != tt.msg {
				t.Errorf("message:\ngot  %q\nwant %q", e.Msg, tt.msg)
			}
		})
	}
}

// Adversarially nested bodies stop at a positioned depth error instead
// of overflowing the goroutine stack, through both body-bearing
// productions.
func TestParseDepthLimit(t *testing.T) {
	deep := strings.Repeat("a {\n", 12000) // beyond maxNestLev, never closed
	tests := map[string]string{
		"deploy body":  "solution s\n\ndeploy ff.Ping as P {\n" + deep,
		"default body": "solution s\n\ndefault deploy {\n" + deep,
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := parser.ParseFile("deep.sdl", []byte(src))
			if f == nil {
				t.Fatal("ParseFile returned a nil file")
			}
			if err == nil {
				t.Fatal("ParseFile succeeded, want a depth error")
			}
			if !strings.Contains(err.Error(), "exceeds maximum nesting depth") {
				t.Errorf("err = %v, want a nesting-depth error", err)
			}
		})
	}
}

func TestParseErrorRecovery(t *testing.T) {
	// A malformed statement must cost only itself: the parser resyncs at
	// the next terminating line break and keeps the surrounding decls.
	f, err := parser.ParseFile("test.sdl", []byte(`solution s

deploy ff.Ping P1 {
	count: 1
}

deploy ff.Pong as P2
`))
	if err == nil {
		t.Fatal("want errors")
	}
	if len(f.Decls) != 2 {
		t.Fatalf("decls after recovery: got %d, want 2", len(f.Decls))
	}
	p2 := f.Decls[1].(*ast.DeployDecl).Specs[0]
	if p2.Name.Name != "P2" {
		t.Errorf("second deploy: got %v", p2.Name)
	}
}

func TestCommentAttachment(t *testing.T) {
	f := parse(t, `// unit header
// second line

solution sample // clause suffix

import (
	// about ff
	ff "example.com/ff" // ff suffix
	// before closing paren
)

// group one

// group two
deploy ff.Ping as P1

// trailing at EOF
`)
	sol := f.Solution
	if len(sol.Before) != 1 || len(sol.Before[0].List) != 2 {
		t.Fatalf("solution Before: got %+v", sol.Before)
	}
	if sol.Before[0].List[0].Text != "// unit header" || sol.Before[0].List[1].Text != "// second line" {
		t.Errorf("solution Before texts: got %q, %q", sol.Before[0].List[0].Text, sol.Before[0].List[1].Text)
	}
	if got := lineCol(sol.Before[0].List[0].Slash); got != "1:1" {
		t.Errorf("header comment position: got %s, want 1:1", got)
	}
	if sol.Suffix == nil || sol.Suffix.Text != "// clause suffix" {
		t.Errorf("solution Suffix: got %+v", sol.Suffix)
	}

	imp := f.Imports[0]
	spec := imp.Specs[0]
	if len(spec.Before) != 1 || spec.Before[0].List[0].Text != "// about ff" {
		t.Errorf("import spec Before: got %+v", spec.Before)
	}
	if spec.Suffix == nil || spec.Suffix.Text != "// ff suffix" {
		t.Errorf("import spec Suffix: got %+v", spec.Suffix)
	}
	if len(imp.After) != 1 || imp.After[0].List[0].Text != "// before closing paren" {
		t.Errorf("import After: got %+v", imp.After)
	}

	dep := f.Decls[0].(*ast.DeployDecl).Specs[0]
	if len(dep.Before) != 2 {
		t.Fatalf("deploy Before groups: got %d, want 2", len(dep.Before))
	}
	if dep.Before[0].List[0].Text != "// group one" || dep.Before[1].List[0].Text != "// group two" {
		t.Errorf("deploy Before texts: got %q, %q", dep.Before[0].List[0].Text, dep.Before[1].List[0].Text)
	}

	if len(f.After) != 1 || f.After[0].List[0].Text != "// trailing at EOF" {
		t.Errorf("file After: got %+v", f.After)
	}
}

func TestCommentInsideBody(t *testing.T) {
	f := parse(t, `solution s

deploy ff.Ping as P {
	// explains count
	count: 1 // suffix here
	// dangling before close
}
`)
	body := f.Decls[0].(*ast.DeployDecl).Specs[0].Body
	param := body.Items[0].(*ast.Param)
	if len(param.Before) != 1 || param.Before[0].List[0].Text != "// explains count" {
		t.Errorf("param Before: got %+v", param.Before)
	}
	if param.Suffix == nil || param.Suffix.Text != "// suffix here" {
		t.Errorf("param Suffix: got %+v", param.Suffix)
	}
	if len(body.After) != 1 || body.After[0].List[0].Text != "// dangling before close" {
		t.Errorf("body After: got %+v", body.After)
	}
}

// The testdata fixtures were frozen from solution/sample.sdl's mockup 6
// corpus (the scratchpad drifts freely per docs/CLAUDE.md, so the smoke
// test owns its own copy): mockup6_main.sdl is the main unit,
// mockup6_peer.sdl the fake peer file (a unit has exactly one solution
// clause, so the test parses them separately, as the linker would).
func TestParseMockup6(t *testing.T) {
	unit1, err := os.ReadFile(filepath.Join("testdata", "mockup6_main.sdl"))
	if err != nil {
		t.Fatal(err)
	}
	unit2, err := os.ReadFile(filepath.Join("testdata", "mockup6_peer.sdl"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := parser.ParseFile("sample.sdl", unit1)
	if err != nil {
		t.Fatalf("mockup 6 main unit: %v", err)
	}
	if f.Solution == nil || f.Solution.Name.Name != "sample" {
		t.Fatalf("solution clause: got %+v", f.Solution)
	}
	if len(f.Solution.Before) == 0 {
		t.Error("mockup 6 header comments not attached to the solution clause")
	}
	wantDecls := []string{
		"*ast.ExternDecl", "*ast.VarDecl", "*ast.DefaultDecl", "*ast.DefaultDecl",
		"*ast.ProvisionDecl", "*ast.ProvisionDecl", "*ast.DeployDecl", "*ast.DeployDecl", "*ast.DeployDecl",
	}
	if len(f.Decls) != len(wantDecls) {
		t.Fatalf("decls: got %d, want %d", len(f.Decls), len(wantDecls))
	}
	for i, want := range wantDecls {
		if got := fmt.Sprintf("%T", f.Decls[i]); got != want {
			t.Errorf("decl %d: got %s, want %s", i, got, want)
		}
	}

	ext := f.Decls[0].(*ast.ExternDecl)
	if len(ext.Specs) != 3 {
		t.Fatalf("extern specs: got %d, want 3", len(ext.Specs))
	}
	natsAdmin := ext.Specs[1]
	if natsAdmin.Name.Name != "natsAdmin" || natsAdmin.Suffix == nil ||
		!strings.Contains(natsAdmin.Suffix.Text, "sensitive") {
		t.Errorf("natsAdmin spec: name %v suffix %+v", natsAdmin.Name, natsAdmin.Suffix)
	}

	if n := len(f.Decls[1].(*ast.VarDecl).Specs); n != 2 {
		t.Errorf("var specs: got %d, want 2", n)
	}

	prov := f.Decls[4].(*ast.ProvisionDecl).Specs[0]
	if prov.Kind == nil || prov.Kind.Name != "slice" || prov.Name.Name != "natsAccount" {
		t.Errorf("provision NATS: kind %v name %v", prov.Kind, prov.Name)
	}

	ping1 := f.Decls[6].(*ast.DeployDecl).Specs[0]
	nats := ping1.Body.Items[1].(*ast.Param)
	if ref := nats.Value.(*ast.RefExpr); ref.X.Name != "natsAccount" || ref.Sel.Name != "config" {
		t.Errorf("Ping1 nats param: got %v.%v", ref.X, ref.Sel)
	}

	ping2 := f.Decls[7].(*ast.DeployDecl).Specs[0]
	if len(ping2.Body.Items) != 5 {
		t.Fatalf("Ping2 body items: got %d, want 5", len(ping2.Body.Items))
	}
	on := ping2.Body.Items[3].(*ast.Section)
	if on.Name.Name != "on" {
		t.Errorf("Ping2 section 1: got %v", on.Name)
	}
	loc := on.Body.Items[0].(*ast.Param)
	if loc.Suffix == nil || !strings.Contains(loc.Suffix.Text, "overrides") {
		t.Errorf("location suffix: got %+v", loc.Suffix)
	}
	meta := ping2.Body.Items[4].(*ast.Section)
	if meta.Name.Name != "metadata" {
		t.Errorf("Ping2 section 2: got %v", meta.Name)
	}
	team := meta.Body.Items[0].(*ast.Param)
	if v := team.Value.(*ast.StringLit); v.Value != "search" {
		t.Errorf("team: got %q", v.Value)
	}

	// The peer unit repeats the solution clause and adds one deployment.
	pf, err := parser.ParseFile("sample_extra.sdl", unit2)
	if err != nil {
		t.Fatalf("mockup 6 peer unit: %v", err)
	}
	if pf.Solution == nil || pf.Solution.Name.Name != "sample" {
		t.Fatalf("peer solution clause: got %+v", pf.Solution)
	}
	if len(pf.Solution.Before) == 0 {
		t.Error("peer-file commentary not attached to the solution clause")
	}
	if len(pf.Decls) != 1 {
		t.Fatalf("peer decls: got %d, want 1", len(pf.Decls))
	}
	ping3 := pf.Decls[0].(*ast.DeployDecl).Specs[0]
	if ping3.Name.Name != "Ping3" || len(ping3.Body.Items) != 2 {
		t.Errorf("Ping3: name %v with %d items", ping3.Name, len(ping3.Body.Items))
	}
}
