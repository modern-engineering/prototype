package gen_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/solution/ast"
	"github.com/modern-engineering/prototype/solution/gen"
)

// Occurrences remain distinct in the model, while linkage needs each exported
// definition only once and should not churn when occurrences are reordered.
func TestOccurrenceOrderDoesNotAffectLinkage(t *testing.T) {
	stdin := ast.Symbol{Package: "os", Name: "Stdin"}
	model := ast.Model{Nodes: []ast.Node{
		{Name: "first", Definition: stdin},
		{Name: "second", Definition: stdin},
		{Name: "eof", Definition: ast.Symbol{Package: "io", Name: "EOF"}},
	}}
	original, err := gen.Source("inspection", "Definitions", model)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(model.Nodes)
	model.Nodes = model.Nodes[:2]
	changed, err := gen.Source("inspection", "Definitions", model)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, changed) {
		t.Fatalf("reordering and removing a repeated occurrence changed linkage:\n%s\nwant:\n%s", changed, original)
	}
}

func TestIdentifierErrors(t *testing.T) {
	for _, name := range []string{"", "lowercase", "a.B", "if"} {
		model := ast.Model{Nodes: []ast.Node{{Definition: ast.Symbol{Package: "os", Name: name}}}}
		if _, err := gen.Source("inspection", "Definitions", model); !errors.Is(err, gen.ErrIdentifier) {
			t.Errorf("symbol %q: error = %v, want ErrIdentifier", name, err)
		}
	}
	for _, name := range []string{"", "_", "a.B", "if"} {
		if _, err := gen.Source(name, "Definitions", ast.Model{}); !errors.Is(err, gen.ErrIdentifier) {
			t.Errorf("package %q: error = %v, want ErrIdentifier", name, err)
		}
		if _, err := gen.Source("inspection", name, ast.Model{}); !errors.Is(err, gen.ErrIdentifier) {
			t.Errorf("function %q: error = %v, want ErrIdentifier", name, err)
		}
	}
}

func TestGeneratedFunctionNames(t *testing.T) {
	for _, test := range []struct{ pkg, function string }{
		{"inspection", "error"},
		{"inspection", "nil"},
		{"inspection", "init"},
		{"main", "main"},
	} {
		if _, err := gen.Source(test.pkg, test.function, ast.Model{}); !errors.Is(err, gen.ErrIdentifier) {
			t.Errorf("package %q, function %q: error = %v, want ErrIdentifier", test.pkg, test.function, err)
		}
	}
}
