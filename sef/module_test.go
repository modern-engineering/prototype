package sef_test

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution"
)

// All non-CUE encodings carry the module context separately from the legacy
// definition requirements. Neither may silently become authoritative for the other.
func TestModuleContextEncodings(t *testing.T) {
	artifact := solution.Artifact{Version: sef.Version, Module: solution.ModuleContext{
		Path: "example.test/solution", Go: "1.27.0", Toolchain: "go1.27.2",
		Require: []solution.Module{{Path: "example.org/catalogue", Version: "v1.2.0", Indirect: true}},
		Exclude: []solution.Module{{Path: "example.org/catalogue", Version: "v1.1.0"}},
		Replace: []solution.ModuleReplacement{{Old: solution.Module{Path: "example.org/catalogue"}, New: solution.Module{Path: "../local catalogue"}}},
		Sum:     []solution.ModuleSum{{Path: "example.org/catalogue", Version: "v1.2.0/go.mod", Hash: "h1:expected"}},
	}, Definition: solution.Definition{Modules: []solution.Module{{Path: "example.org/legacy", Version: "v0.1.0"}}}}
	codecs := append(encodings, struct {
		name   string
		encode func(io.Writer, solution.Artifact) error
		decode func(io.Reader) (solution.Artifact, error)
	}{
		name: "Text", encode: seftext.Encode, decode: func(r io.Reader) (solution.Artifact, error) {
			file, err := seftext.Parse("module.sef", r)
			if err != nil {
				return solution.Artifact{}, err
			}
			return file.Artifact(), nil
		},
	})
	for _, codec := range codecs {
		var encoded bytes.Buffer
		if err := codec.encode(&encoded, artifact); err != nil {
			t.Fatalf("%s: %v", codec.name, err)
		}
		got, err := codec.decode(&encoded)
		if err != nil {
			t.Fatalf("%s: %v", codec.name, err)
		}
		if !reflect.DeepEqual(got, artifact) {
			t.Fatalf("%s: module context changed: %#v", codec.name, got)
		}
	}
	artifact.Module.Replace[0].New.Path = "invalid\xff"
	if err := sef.EncodeJSON(io.Discard, artifact); !errors.Is(err, sef.ErrUnicode) {
		t.Fatalf("module Unicode error = %v", err)
	}
}
