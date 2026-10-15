package seftool_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/sef"
	seftext "github.com/modern-engineering/prototype/sef/text"
)

// Authoring expressions must disappear at the artifact boundary while named
// output references remain available to the linker.
func TestCUEArtifactWorkflow(t *testing.T) {
	source, err := os.ReadFile("testdata/service.cue.sef")
	if err != nil {
		t.Fatal(err)
	}
	textSource, err := os.ReadFile("testdata/service.sef")
	if err != nil {
		t.Fatal(err)
	}
	textFile, err := seftext.Parse("service.sef", bytes.NewReader(textSource))
	if err != nil {
		t.Fatal(err)
	}
	expected := textFile.Artifact()

	for _, format := range []string{"cue", "text", "json", "gob"} {
		t.Run(format, func(t *testing.T) {
			var output, diagnostics bytes.Buffer
			command := seftool.Command{Stdin: bytes.NewReader(source), Stdout: &output, Stderr: &diagnostics}
			if status := command.Run([]string{"convert", "-from", "cue", "-to", format}); status != 0 {
				t.Fatalf("conversion status=%d: %s", status, &diagnostics)
			}
			encoded := bytes.Clone(output.Bytes())
			output.Reset()
			command.Stdin = bytes.NewReader(encoded)
			if status := command.Run([]string{"dump", "-from", format}); status != 0 {
				t.Fatalf("dump status=%d: %s", status, &diagnostics)
			}
			got, err := sef.DecodeJSON(&output)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("artifact=%#v; want %#v", got, expected)
			}
		})
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"inspect", "testdata/service.cue.sef"}); status != 0 {
		t.Fatalf("inspection status=%d: %s", status, &diagnostics)
	}
	if strings.Contains(output.String(), "eu-west") {
		t.Fatal("inspection disclosed a literal")
	}
}

// Formatting operates on source syntax, including unfinished constraints, while
// inspection requires a concrete artifact.
func TestCUESourceFormatting(t *testing.T) {
	source := "// Region remains undecided.\nversion: \"0.1\"\ndefinition: bindings: [{node: \"database\", input: \"region\", value: {kind: \"literal\", text: string}}]\n"
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Stdin: strings.NewReader(source), Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"fmt", "-from", "cue"}); status != 0 {
		t.Fatalf("format status=%d: %s", status, &diagnostics)
	}
	formatted := bytes.Clone(output.Bytes())
	if !bytes.Contains(formatted, []byte("// Region remains undecided.")) {
		t.Fatal("format lost comment")
	}
	output.Reset()
	command.Stdin = bytes.NewReader(formatted)
	if status := command.Run([]string{"fmt", "-from", "cue"}); status != 0 {
		t.Fatalf("second format status=%d: %s", status, &diagnostics)
	}
	if !bytes.Equal(output.Bytes(), formatted) {
		t.Fatal("format is not idempotent")
	}
	output.Reset()
	command.Stdin = bytes.NewReader(formatted)
	if status := command.Run([]string{"check", "-from", "cue"}); status != 1 {
		t.Fatalf("incomplete check status=%d; want 1", status)
	}
	if output.Len() != 0 {
		t.Fatal("failed check produced output")
	}
}
