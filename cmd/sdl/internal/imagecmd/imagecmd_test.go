// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package imagecmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// testImage builds a small image covering both record verbs and both
// symbol classes.
func testImage() *image.Image {
	return &image.Image{
		Format:     image.Format,
		Solution:   "sample",
		Generation: 1,
		Catalogue: []image.Package{{
			Path:     "example.com/acme/pingpong",
			Name:     "pingpong",
			Elements: []image.ElementSchema{{Name: "Ping", Kind: image.KindComponent}},
		}},
		Symbols: []image.SymbolDef{
			{Name: "adminKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.com/acme/substrate", Name: "Secret"}},
			{Name: "pingInterval", Class: image.ClassVar, Value: image.Duration(1500 * time.Millisecond)},
			{Name: "subject", Class: image.ClassVar, Value: image.String("com.acme.Echo")},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindSlice,
				Element: image.Ref{Package: "example.com/acme/substrate", Name: "NATS"},
				Name:    "natsAccount",
			},
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: "example.com/acme/pingpong", Name: "Ping"},
				Name:    "Ping1",
				Deployment: []image.Binding{
					{Key: "location", Value: image.Token("euCentral1"), Source: image.SourceInstance},
				},
				Extensions: map[string][]image.Binding{
					"k8s.pod":      {{Key: "replicas", Value: image.Int(3), Source: image.SourceInstance}},
					"k8s.workload": {},
				},
			},
		},
	}
}

// TestInfoReport pins the info rendering over a governance-bearing
// image: header facts first, settings as key=value, units with their
// digests, and the catalogue pin summary with pluralized element
// counts — every field literally tab-separated, the go version -m
// shape.
func TestInfoReport(t *testing.T) {
	img := testImage()
	img.Generation = 4
	img.Catalogue = append(img.Catalogue, image.Package{
		Path: "example.com/acme/substrate",
		Name: "substrate",
		Elements: []image.ElementSchema{
			{Name: "NATS", Kind: image.KindProvision},
			{Name: "Secret", Kind: image.KindSymbol},
		},
	})
	img.Build = &image.Build{
		Units: []image.UnitDigest{
			{Name: "main.sdl", SHA256: strings.Repeat("ab", 32)},
			{Name: "peer.sdl", SHA256: strings.Repeat("cd", 32)},
		},
		Settings: []image.Setting{
			{Key: "prototype.version", Value: "v0.1.0"},
			{Key: "sdl.version", Value: "v0.1.2"},
		},
	}
	var out bytes.Buffer
	if err := info(img, &out); err != nil {
		t.Fatal(err)
	}
	want := "" +
		"solution\tsample\n" +
		"generation\t4\n" +
		"format\tsolution-image/1\n" +
		"setting\tprototype.version=v0.1.0\n" +
		"setting\tsdl.version=v0.1.2\n" +
		"unit\tmain.sdl\t" + strings.Repeat("ab", 32) + "\n" +
		"unit\tpeer.sdl\t" + strings.Repeat("cd", 32) + "\n" +
		"catalogue\texample.com/acme/pingpong\tpingpong\t1 element\n" +
		"catalogue\texample.com/acme/substrate\tsubstrate\t2 elements\n"
	if out.String() != want {
		t.Errorf("info report:\n%s--- want ---\n%s", out.String(), want)
	}
}

// TestInfoWithoutBuild pins the pre-governance shape: an image without
// the block reports its header and catalogue, no setting or unit lines.
func TestInfoWithoutBuild(t *testing.T) {
	var out bytes.Buffer
	if err := info(testImage(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "setting\t") || strings.Contains(got, "unit\t") {
		t.Errorf("info of a build-less image printed governance lines:\n%s", got)
	}
	if !strings.Contains(got, "solution\tsample\n") || !strings.Contains(got, "catalogue\texample.com/acme/pingpong\tpingpong\t1 element\n") {
		t.Errorf("info report is missing header or catalogue lines:\n%s", got)
	}
}

func TestRecordsTable(t *testing.T) {
	var out bytes.Buffer
	if err := records(testImage(), &out); err != nil {
		t.Fatal(err)
	}
	want := "" +
		"VERB       KIND   ELEMENT                          NAME         DEPLOYMENT            EXTENSIONS\n" +
		"provision  slice  example.com/acme/substrate.NATS  natsAccount\n" +
		"deploy            example.com/acme/pingpong.Ping   Ping1        location: euCentral1  k8s.pod, k8s.workload\n"
	if out.String() != want {
		t.Errorf("records table:\n%s--- want ---\n%s", out.String(), want)
	}
}

func TestRecordsTableFaults(t *testing.T) {
	tests := map[string]struct {
		mutate func(img *image.Image)
		want   string
	}{
		"RefInDeployment": {
			mutate: func(img *image.Image) {
				img.Records[1].Deployment = []image.Binding{{Key: "location", Ref: &image.SymbolRef{Symbol: "subject"}}}
			},
			want: "carries a symbol reference",
		},
		"FieldWithoutValue": {
			mutate: func(img *image.Image) {
				img.Records[1].Deployment = []image.Binding{{Key: "location"}}
			},
			want: "has no value",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			img := testImage()
			tt.mutate(img)
			var out bytes.Buffer
			err := records(img, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want mention of %q", err, tt.want)
			}
		})
	}
}

func TestSymbolsTable(t *testing.T) {
	var out bytes.Buffer
	if err := symbols(testImage(), &out); err != nil {
		t.Fatal(err)
	}
	want := "" +
		"NAME          CLASS   TYPE/VALUE\n" +
		"adminKey      extern  example.com/acme/substrate.Secret\n" +
		"pingInterval  var     1.5s\n" +
		"subject       var     \"com.acme.Echo\"\n"
	if out.String() != want {
		t.Errorf("symbols table:\n%s--- want ---\n%s", out.String(), want)
	}
}

func TestSymbolsTableFaults(t *testing.T) {
	tests := map[string]struct {
		mutate func(img *image.Image)
		want   string
	}{
		"ExternWithoutType": {
			mutate: func(img *image.Image) { img.Symbols[0].Type = nil },
			want:   "has no type",
		},
		"VarWithoutValue": {
			mutate: func(img *image.Image) { img.Symbols[1].Value = nil },
			want:   "has no value",
		},
		"UnknownClass": {
			mutate: func(img *image.Image) { img.Symbols[0].Class = "weak" },
			want:   `class "weak"`,
		},
		"TokenValue": {
			mutate: func(img *image.Image) { img.Symbols[1].Value = image.Token("here") },
			want:   `value kind "token"`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			img := testImage()
			tt.mutate(img)
			var out bytes.Buffer
			err := symbols(img, &out)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want mention of %q", err, tt.want)
			}
		})
	}
}

// writeImage lands one encoded image in a fresh temp directory and
// returns its path.
func writeImage(t *testing.T, img *image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	if err := img.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "image.json")
	if err := os.WriteFile(path, buf.Bytes(), 0o666); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditGeneration(t *testing.T) {
	path := writeImage(t, testImage())
	if err := edit(path, 7); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, err := image.Decode(f)
	if err != nil {
		t.Fatalf("edited image does not decode: %v", err)
	}
	if img.Generation != 7 {
		t.Errorf("Generation = %d, want 7", img.Generation)
	}
	if !image.Equal(img, testImage()) {
		t.Error("edit must change nothing besides the generation")
	}

	// The rewrite is canonical: re-encoding the decoded image yields
	// the file's exact bytes.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	if err := img.Encode(&want); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Error("edited file is not the canonical encoding")
	}
}

// TestEditAtomicity corrupts the input and expects the fault to leave
// the original bytes untouched and no temporary litter behind.
func TestEditAtomicity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.json")
	corrupt := []byte(`{"format":"solution-image/1",`)
	if err := os.WriteFile(path, corrupt, 0o666); err != nil {
		t.Fatal(err)
	}

	if err := edit(path, 7); err == nil {
		t.Fatal("edit of a corrupt image succeeded, want an error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, corrupt) {
		t.Errorf("edit failure changed the file:\n%s", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("edit failure left %d directory entries, want the original only", len(entries))
	}
}

func TestEditMissingFile(t *testing.T) {
	if err := edit(filepath.Join(t.TempDir(), "absent.json"), 7); err == nil {
		t.Fatal("edit of a missing file succeeded, want an error")
	}
}
