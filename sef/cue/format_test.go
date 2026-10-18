package cue_test

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

	sefcue "github.com/modern-engineering/prototype/sef/cue"
)

// Source formatting keeps authoring constructs. Evaluated encoding has a
// different purpose and must not be used to rewrite source expressions.
func TestFormatPreservesSource(t *testing.T) {
	input, err := os.ReadFile("testdata/service.input")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/service.golden")
	if err != nil {
		t.Fatal(err)
	}
	file, err := sefcue.Parse("service.cue.sef", bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	var formatted bytes.Buffer
	if err := sefcue.Format(&formatted, file); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(formatted.Bytes(), want) {
		t.Fatalf("Format:\n%s\nwant:\n%s", formatted.Bytes(), want)
	}
	reparsed, err := sefcue.Parse("service.cue.sef", bytes.NewReader(formatted.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reparsed.Artifact(), file.Artifact()) {
		t.Fatal("formatting changed the evaluated artifact")
	}
	var again bytes.Buffer
	if err := sefcue.Format(&again, reparsed); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again.Bytes(), formatted.Bytes()) {
		t.Fatal("Format is not a fixed point")
	}
	var encoded bytes.Buffer
	if err := sefcue.Encode(&encoded, file.Artifact()); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded.Bytes(), []byte("#Trim")) || bytes.Contains(encoded.Bytes(), []byte("strings.Join")) || bytes.Contains(encoded.Bytes(), []byte("//")) {
		t.Fatalf("Encode retained source constructs:\n%s", encoded.Bytes())
	}
	evaluated, err := sefcue.Parse("evaluated.cue.sef", bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(evaluated.Artifact(), file.Artifact()) {
		t.Fatal("evaluated encoding changed the artifact")
	}
}

func TestFormatSourceDoesNotRequireArtifact(t *testing.T) {
	source := "// Authoring can leave constraints incomplete.\nversion: string\nunknown:{port:int & >0}\n"
	var output ownedBuffer
	if err := sefcue.FormatSource(&output, "draft.cue.sef", strings.NewReader(source)); err != nil {
		t.Fatal(err)
	}
	want := "// Authoring can leave constraints incomplete.\nversion: string\nunknown: {port: int & >0}\n"
	if output.String() != want {
		t.Fatalf("FormatSource = %q, want %q", output.String(), want)
	}
	if _, err := sefcue.Parse("draft.cue.sef", bytes.NewReader(output.Bytes())); err == nil {
		t.Fatal("Parse accepted an incomplete artifact")
	}
	if output.closed {
		t.Fatal("FormatSource closed its writer")
	}
	failure := errors.New("storage unavailable")
	if err := sefcue.FormatSource(&output, "failed.cue.sef", iotest.ErrReader(failure)); !errors.Is(err, failure) {
		t.Errorf("FormatSource = %v, want original reader failure", err)
	}
	if err := sefcue.FormatSource(errorWriter{failure}, "draft.cue.sef", strings.NewReader(source)); !errors.Is(err, failure) {
		t.Errorf("FormatSource = %v, want original writer failure", err)
	}
	output.Reset()
	if err := sefcue.FormatSource(&output, "broken.cue.sef", strings.NewReader("version: {")); err == nil || output.Len() != 0 {
		t.Errorf("FormatSource invalid syntax wrote %d bytes, err = %v", output.Len(), err)
	}
}
