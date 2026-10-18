package edit_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/edit"
)

// Renaming an occurrence must preserve literal text, repeated assignments,
// self-references, and metadata while updating every logical use of its identity.
func TestRenameRelationships(t *testing.T) {
	input, err := os.ReadFile("testdata/rename.sef")
	if err != nil {
		t.Fatal(err)
	}
	file, err := text.Parse("rename.sef", bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	artifact := file.Artifact()
	reference := artifact.Definition.Bindings[1].Value.(solution.Reference)
	artifact.Definition.Bindings[1].Value = &reference
	got, err := edit.Rename(artifact.Definition, "database", "storage")
	if err != nil {
		t.Fatal(err)
	}
	if reference.Node != "database" || artifact.Definition.Nodes[0].Name != "database" || artifact.Definition.Bindings[0].Node != "database" {
		t.Fatal("source definition was modified")
	}
	artifact.Definition = got
	var output bytes.Buffer
	if err := text.Encode(&output, artifact); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/renamed.sef")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output.Bytes(), want) {
		t.Errorf("renamed artifact:\n%s\nwant:\n%s", output.Bytes(), want)
	}
	got.Bindings[1].Value.(*solution.Reference).Node = "changed"
	if reference.Node != "database" {
		t.Fatal("result aliases source reference")
	}
}

func TestRenameRejectsAmbiguousIdentity(t *testing.T) {
	for _, test := range []struct {
		nodes    []solution.Node
		old, new string
		err      error
	}{
		{nil, "absent", "new", edit.ErrMissingNode},
		{[]solution.Node{{Name: "a"}, {Name: "a"}}, "a", "b", edit.ErrDuplicateNode},
		{[]solution.Node{{Name: "a"}, {Name: "b"}}, "a", "b", edit.ErrDuplicateNode},
	} {
		source := solution.Definition{Nodes: test.nodes}
		got, err := edit.Rename(source, test.old, test.new)
		if !errors.Is(err, test.err) {
			t.Errorf("Rename(%q, %q) error = %v; want %v", test.old, test.new, err, test.err)
		}
		if !reflect.DeepEqual(got, solution.Definition{}) {
			t.Errorf("failed rename returned %#v", got)
		}
	}
}

// Occurrence names can change without changing the descriptor they denote.
func ExampleRename() {
	source := solution.Definition{
		Nodes:    []solution.Node{{Name: "cache", Definition: solution.Symbol{Package: "example.com/cache", Name: "Redis"}}},
		Bindings: []solution.Binding{{Node: "cache", Input: "prefix", Value: solution.Literal("cache")}},
	}
	renamed, err := edit.Rename(source, "cache", "session-store")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(renamed.Nodes[0].Name, renamed.Nodes[0].Definition.Name)
	fmt.Println(renamed.Bindings[0].Node, renamed.Bindings[0].Value)
	fmt.Println(source.Nodes[0].Name)
	// Output:
	// session-store Redis
	// session-store cache
	// cache
}

// An unresolved name must not acquire a producer or receiver as a side effect of
// renaming another occurrence. Such repair is a separate, deliberate edit.
func TestRenameDoesNotCaptureUnresolvedEndpoints(t *testing.T) {
	for _, binding := range []solution.Binding{
		{Node: "b", Input: "text", Value: solution.Literal("x")},
		{Node: "a", Input: "text", Value: solution.Reference{Node: "b", Output: "text"}},
		{Node: "a", Input: "text", Value: &solution.Reference{Node: "b", Output: "text"}},
	} {
		source := solution.Definition{Nodes: []solution.Node{{Name: "a"}}, Bindings: []solution.Binding{binding}}
		got, err := edit.Rename(source, "a", "b")
		if !errors.Is(err, edit.ErrNameCapture) {
			t.Errorf("binding %#v: got %v, want ErrNameCapture", binding, err)
		}
		if !reflect.DeepEqual(got, solution.Definition{}) || source.Nodes[0].Name != "a" {
			t.Fatal("failed rename changed structural data")
		}
	}
}
