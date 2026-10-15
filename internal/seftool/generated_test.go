package seftool_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/sef"
	seftext "github.com/modern-engineering/prototype/sef/text"
)

// A standalone command must discover the named descriptors through generated
// Go code, without requiring the adopting module to maintain an executable.
func TestGeneratedHostWorkflow(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("testdata/service.sef")
	if err != nil {
		t.Fatal(err)
	}
	file, err := seftext.Parse("service.sef", bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	var materialized bytes.Buffer
	if err := sef.EncodeGob(&materialized, file.Artifact()); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Context: t.Context(), Stdout: &output, Stderr: &diagnostics}
	binary := filepath.Join(t.TempDir(), "service.host")
	if status := command.Run([]string{"build", "-toolkit", toolkit, "-work", "-o", binary, "testdata/service.sef"}); status != 0 {
		t.Fatalf("build status=%d: %s", status, &diagnostics)
	}
	var work string
	for line := range strings.SplitSeq(diagnostics.String(), "\n") {
		if strings.HasPrefix(line, "WORK=") {
			work = strings.TrimPrefix(line, "WORK=")
		}
	}
	if work == "" {
		t.Fatal("build did not report retained work")
	}
	t.Cleanup(func() { _ = os.RemoveAll(work) })
	for _, name := range []string{"main.go", "go.mod"} {
		data, err := os.ReadFile(filepath.Join(work, name))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("eu-west")) {
			t.Fatalf("%s contains a literal", name)
		}
	}
	run := exec.CommandContext(t.Context(), binary, "inspect", "-inputs")
	run.Stdin = bytes.NewReader(materialized.Bytes())
	report, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("retained host: %v: %s", err, report)
	}
	if !bytes.Contains(report, []byte(`input "region"`)) || !bytes.Contains(report, []byte(`output "address"`)) {
		t.Fatalf("missing actual fields: %s", report)
	}
	if bytes.Contains(report, []byte("eu-west")) {
		t.Fatal("inspection disclosed a literal")
	}

	output.Reset()
	diagnostics.Reset()
	if status := command.Run([]string{"inspect", "-inputs", "-toolkit", toolkit, "testdata/service.cue.sef"}); status != 0 {
		t.Fatalf("automatic host status=%d: %s", status, &diagnostics)
	}
	if !bytes.Equal(output.Bytes(), report) {
		t.Fatalf("automatic host differs from retained host:\n%s\nwant:\n%s", &output, report)
	}
}

// An explicit replacement has precedence over development-mode discovery, while
// an unreleased toolkit still needs a requirement for that local replacement.
func TestDevelopmentToolkitReplacement(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Context: t.Context(), Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"check", "-linked", "-dev", "-replace", "github.com/modern-engineering/prototype=" + toolkit, "testdata/service.sef"}); status != 0 {
		t.Fatalf("explicit development replacement status=%d: %s", status, &diagnostics)
	}
}
