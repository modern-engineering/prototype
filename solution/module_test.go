package solution_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/solution"
)

// Authored module intent survives source parsing and writing without carrying a
// mutable syntax tree. Cloning lets a build apply local overlays independently.
func TestModuleContextRoundTrip(t *testing.T) {
	mod, err := os.ReadFile("testdata/module/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("testdata/module/go.sum")
	if err != nil {
		t.Fatal(err)
	}
	original, err := solution.ParseModule(bytes.NewReader(mod), bytes.NewReader(sums))
	if err != nil {
		t.Fatal(err)
	}
	clone := original.Clone()
	clone.Require[0].Version = "v1.3.0"
	clone.Exclude[0].Version = "v1.0.0"
	clone.Replace[0].New.Version = "v1.3.1"
	clone.Sum[0].Version = "v1.3.0"
	if original.Require[0].Version != "v1.2.0" || original.Exclude[0].Version != "v1.1.0" || original.Replace[0].New.Version != "v1.2.1" || original.Sum[0].Version != "v1.2.0" {
		t.Fatal("clone shares module slices")
	}
	var output, checksums bytes.Buffer
	if err := clone.Write(&output, &checksums); err != nil {
		t.Fatal(err)
	}
	got, err := solution.ParseModule(&output, &checksums)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, clone) {
		t.Fatalf("roundtrip = %#v; want %#v", got, clone)
	}
	output.Reset()
	if err := original.Write(&output, nil); err != nil {
		t.Fatal(err)
	}
	got, err = solution.ParseModule(&output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sum != nil || !got.Require[1].Indirect {
		t.Fatalf("optional sums or indirect requirement changed: %#v", got)
	}
}

// An incomplete semantic adapter must reject directives it cannot carry rather
// than silently changing the meaning of a project when it is packaged.
func TestUnsupportedModuleDirectives(t *testing.T) {
	for _, directive := range []string{"godebug panicnil=1", "retract v1.0.0", "tool example.org/tool", "ignore ./temporary"} {
		_, err := solution.ParseModule(strings.NewReader("module example.test/solution\ngo 1.27.0\n"+directive+"\n"), nil)
		if !errors.Is(err, solution.ErrModule) {
			t.Errorf("%q: %v; want ErrModule", directive, err)
		}
	}
	if _, err := solution.ParseModule(strings.NewReader("go 1.27.0\n"), nil); !errors.Is(err, solution.ErrModule) {
		t.Fatalf("missing module = %v", err)
	}
}

// Invalid intent must leave both destination files untouched. Real stream errors
// remain available to callers independently of module syntax diagnostics.
func TestModuleStreams(t *testing.T) {
	failure := errors.New("storage unavailable")
	valid := "module example.test/solution\n"
	if _, err := solution.ParseModule(iotest.ErrReader(failure), nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := solution.ParseModule(strings.NewReader(valid), iotest.ErrReader(failure)); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	context := solution.ModuleContext{Path: "example.test/solution", Sum: []solution.ModuleSum{{Path: "example.org/catalogue", Version: "v1.0.0", Hash: "h1:abc"}}}
	if err := context.Write(moduleErrorWriter{failure}, nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := context.Write(io.Discard, moduleErrorWriter{failure}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	context.Sum[0].Hash = "h1:abc\nexample.org/injected v1.0.0 h1:def"
	var mod, sums bytes.Buffer
	if err := context.Write(&mod, &sums); !errors.Is(err, solution.ErrModule) || mod.Len() != 0 || sums.Len() != 0 {
		t.Fatalf("invalid write: %v, %q, %q", err, &mod, &sums)
	}
}

type moduleErrorWriter struct{ err error }

func (w moduleErrorWriter) Write([]byte) (int, error) { return 0, w.err }
