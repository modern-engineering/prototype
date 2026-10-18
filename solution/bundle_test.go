package solution_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/modern-engineering/prototype/solution"
)

// A directory shares one module context across independently encoded artifacts;
// absent checksum files must not prevent local composition.
func TestBundleDirectory(t *testing.T) {
	bundle, err := solution.OpenBundle(os.DirFS("testdata/bundle"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Module.Path != "example.test/solution" || len(bundle.Module.Require) != 1 || len(bundle.Module.Sum) != 0 {
		t.Fatalf("module = %+v", bundle.Module)
	}
	if _, err := fs.ReadFile(bundle.Files, "solutions/service.sef"); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ReadFile(bundle.Files, "assets/message.txt"); err != nil {
		t.Fatal(err)
	}
}

// Multiple manifests and filesystem links obscure which files and module
// context are actually shipped, even if the linked target is readable locally.
func TestBundleHasOneRoot(t *testing.T) {
	for _, entry := range []struct {
		name string
		mode fs.FileMode
	}{
		{"nested/go.mod", 0}, {"nested/go.sum", 0}, {"link", fs.ModeSymlink}, {"socket", fs.ModeSocket},
	} {
		files := fstest.MapFS{
			"go.mod":   {Data: []byte("module example.test/solution\n")},
			entry.name: {Data: []byte("module example.test/nested\n"), Mode: entry.mode},
		}
		if _, err := solution.OpenBundle(files); !errors.Is(err, solution.ErrBundle) {
			t.Errorf("%s: error = %v, want ErrBundle", entry.name, err)
		}
	}
	if _, err := solution.OpenBundle(fstest.MapFS{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing module: %v", err)
	}
}

// Open a directory once, then decode its artifacts with the chosen encoding.
func ExampleOpenBundle() {
	bundle, err := solution.OpenBundle(os.DirFS("testdata/bundle"))
	if err != nil {
		fmt.Println(err)
		return
	}
	content, err := fs.ReadFile(bundle.Files, "assets/message.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(bundle.Module.Path)
	fmt.Print(string(content))
	// Output:
	// example.test/solution
	// ready to serve
}
