package text_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/sef"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution"
)

// These files exercise authored order, repeated assignments, unresolved names,
// comments, and strings that resemble syntax. Re-encoding must preserve meaning.
func TestArtifactsRoundTrip(t *testing.T) {
	for _, name := range []string{"catalogue.sef", "strings.sef"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", name)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			file, err := seftext.Parse(path, bytes.NewReader(source))
			if err != nil {
				t.Fatal(err)
			}
			want := file.Artifact()
			var encoded bytes.Buffer
			if err := seftext.Encode(&encoded, want); err != nil {
				t.Fatal(err)
			}
			roundtrip, err := seftext.Parse("roundtrip.sef", &encoded)
			if err != nil {
				t.Fatal(err)
			}
			if got := roundtrip.Artifact(); !reflect.DeepEqual(got, want) {
				t.Errorf("round trip = %#v, want %#v", got, want)
			}
			copy := file.Artifact()
			copy.Definition.Nodes[0].Name = "changed"
			copy.Definition.Bindings[0].Value = solution.Literal("changed")
			if got := file.Artifact(); !reflect.DeepEqual(got, want) {
				t.Errorf("editing snapshot changed parsed artifact: %#v", got)
			}
		})
	}
}

// One malformed statement must not hide diagnostics or valid declarations later
// in the file. Positions remain usable without parsing error-message prose.
func TestSyntaxRecoveryHasLocations(t *testing.T) {
	const path = "testdata/malformed.sef"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := seftext.Parse(path, bytes.NewReader(source))
	var diagnostics seftext.ErrorList
	if !errors.As(err, &diagnostics) {
		t.Fatalf("Parse error = %T, want ErrorList", err)
	}
	want := []struct{ line, column int }{{2, 1}, {4, 1}, {5, 1}, {6, 1}, {7, 10}, {8, 1}, {9, 1}}
	if len(diagnostics) != len(want) {
		t.Fatalf("diagnostics = %v; want %d errors", diagnostics, len(want))
	}
	for i, diagnostic := range diagnostics {
		if diagnostic.Position.Filename != path || diagnostic.Position.Line != want[i].line || diagnostic.Position.Column != want[i].column {
			t.Errorf("diagnostic %d at %+v, want %s:%d:%d", i, diagnostic.Position, path, want[i].line, want[i].column)
		}
	}
	if got := file.Artifact().Definition.Modules; !reflect.DeepEqual(got, []solution.Module{{Path: "example.com/catalogue", Version: "v1.2.0"}}) {
		t.Errorf("recovered modules = %#v", got)
	}
	_, err = seftext.Parse("missing.sef", strings.NewReader(`solution "id"`))
	var diagnostic *seftext.Error
	if !errors.As(err, &diagnostic) || diagnostic.Position.Filename != "missing.sef" {
		t.Fatalf("missing version error = %v, want located syntax error", err)
	}
}

// Invalid artifacts must not leave an apparently valid file prefix, and stream
// failures remain distinguishable from syntax failures through errors.Is.
func TestStreamFailuresAndInvalidArtifacts(t *testing.T) {
	failure := errors.New("storage unavailable")
	if file, err := seftext.Parse("input.sef", iotest.ErrReader(failure)); file != nil || !errors.Is(err, failure) {
		t.Fatalf("Parse failing reader = %v, %v", file, err)
	}
	if err := seftext.Encode(errorWriter{failure}, solution.Artifact{Version: sef.Version}); !errors.Is(err, failure) {
		t.Errorf("Encode failing writer = %v", err)
	}
	for _, artifact := range []solution.Artifact{
		{},
		{Version: sef.Version, Definition: solution.Definition{Bindings: []solution.Binding{{}}}},
	} {
		var output bytes.Buffer
		if err := seftext.Encode(&output, artifact); err == nil || output.Len() != 0 {
			t.Errorf("Encode invalid artifact wrote %d bytes, err = %v", output.Len(), err)
		}
	}
	literal := solution.Literal("")
	reference := solution.Reference{}
	artifact := solution.Artifact{Version: sef.Version, Definition: solution.Definition{Bindings: []solution.Binding{
		{Value: &literal}, {Value: &reference},
	}}}
	var output bytes.Buffer
	if err := seftext.Encode(&output, artifact); err != nil {
		t.Fatal(err)
	}
	if _, err := seftext.Parse("pointers.sef", &output); err != nil {
		t.Fatal(err)
	}
}

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }
