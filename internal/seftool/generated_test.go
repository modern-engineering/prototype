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
	"github.com/modern-engineering/prototype/solution"
)

// Authoring resolves one module context for all bundled artifacts. Packing and
// standalone export must retain it, while local source remains an explicit
// development choice and builds leave the author's files untouched.
func TestBundleHostWorkflow(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.CopyFS(project, os.DirFS("testdata/bundle")); err != nil {
		t.Fatal(err)
	}
	bundle, err := solution.OpenBundle(os.DirFS(project))
	if err != nil {
		t.Fatal(err)
	}
	bundle.Module.Replace[0].New.Path = toolkit
	var manifest, sums bytes.Buffer
	if err := bundle.Module.Write(&manifest, &sums); err != nil {
		t.Fatal(err)
	}
	authored := append([]byte("// Catalogue dependencies for the service bundle.\n"), manifest.Bytes()...)
	if err := os.WriteFile(filepath.Join(project, "go.mod"), authored, 0o600); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Context: t.Context(), Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"mod", "tidy", "-dev", project}); status != 0 {
		t.Fatalf("tidy status=%d: %s", status, &diagnostics)
	}
	original, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(original, []byte("// Catalogue dependencies for the service bundle.")) {
		t.Fatal("module editing discarded authoring comments")
	}
	originalSums, err := os.ReadFile(filepath.Join(project, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"check", "-linked", project}, {"check", "-linked", "-dev=false", project}} {
		if status := command.Run(args); status != 1 {
			t.Fatalf("production status=%d: %s", status, &diagnostics)
		}
	}
	diagnostics.Reset()
	binary, outdir := filepath.Join(root, "service.host"), filepath.Join(root, "generated")
	if status := command.Run([]string{"build", "-dev", "-outdir", outdir, "-o", binary, project}); status != 0 {
		t.Fatalf("build status=%d: %s", status, &diagnostics)
	}
	if !strings.Contains(diagnostics.String(), "OUTDIR="+outdir) {
		t.Fatalf("missing output directory: %s", &diagnostics)
	}
	for _, name := range []string{"main.go", "go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(outdir, name))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("eu-west")) {
			t.Fatalf("%s contains a literal", name)
		}
	}
	output.Reset()
	standalone := filepath.Join(root, "service.gob.sef")
	if status := command.Run([]string{"convert", "-to", "gob", "-o", standalone, project}); status != 0 {
		t.Fatalf("snapshot status=%d: %s", status, &diagnostics)
	}
	data, err := os.ReadFile(standalone)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sef.DecodeGob(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Module.Path != "example.test/service-solution" || len(artifact.Module.Sum) == 0 {
		t.Fatalf("missing module context: %#v", artifact.Module)
	}
	jobs := artifact
	jobs.Definition.Nodes = jobs.Definition.Nodes[:1]
	jobs.Definition.Bindings = jobs.Definition.Bindings[:1]
	var jobData bytes.Buffer
	if err := sef.EncodeGob(&jobData, jobs); err != nil {
		t.Fatal(err)
	}
	jobPath := filepath.Join(root, "job.gob.sef")
	if err := os.WriteFile(jobPath, jobData.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if status := command.Run([]string{"playground", "-dev", "-job-timeout", "2s", jobPath}); status != 0 {
		t.Fatalf("generated playground status=%d: %s", status, &diagnostics)
	}
	run := exec.CommandContext(t.Context(), binary, "inspect", "-inputs")
	run.Stdin = bytes.NewReader(data)
	report, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("retained host: %v: %s", err, report)
	}
	if !bytes.Contains(report, []byte(`input "region"`)) || !bytes.Contains(report, []byte(`output "address"`)) {
		t.Fatalf("missing fields: %s", report)
	}
	if bytes.Contains(report, []byte("eu-west")) {
		t.Fatal("inspection disclosed a literal")
	}
	for _, suffix := range []string{".tar", ".tar.gz"} {
		archive := filepath.Join(root, "bundle"+suffix)
		if status := command.Run([]string{"pack", "-o", archive, project}); status != 0 {
			t.Fatalf("pack status=%d: %s", status, &diagnostics)
		}
		output.Reset()
		if status := command.Run([]string{"inspect", archive}); status != 0 {
			t.Fatalf("archive inspect status=%d: %s", status, &diagnostics)
		}
		if status := command.Run([]string{"inspect", "-inputs", "-dev", archive}); status != 1 {
			t.Fatalf("archive borrowed a development directory: %d", status)
		}
		destination := filepath.Join(root, "unpacked"+suffix)
		if status := command.Run([]string{"unpack", "-outdir", destination, archive}); status != 0 {
			t.Fatalf("unpack status=%d: %s", status, &diagnostics)
		}
		got, err := os.ReadFile(filepath.Join(destination, "service.sef"))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(project, "service.sef"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatal("archive changed authored source")
		}
		output.Reset()
		if status := command.Run([]string{"inspect", "-inputs", "-dev", destination}); status != 0 {
			t.Fatalf("unpacked inspection status=%d: %s", status, &diagnostics)
		}
		if !bytes.Equal(output.Bytes(), report) {
			t.Fatalf("unpacked artifact differs from standalone:\n%s\nwant:\n%s", &output, report)
		}
		if status := command.Run([]string{"unpack", "-outdir", destination, archive}); status != 1 {
			t.Fatalf("unpack overwrote destination: %d", status)
		}
	}
	unchanged, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	unchangedSums, err := os.ReadFile(filepath.Join(project, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, unchanged) || !bytes.Equal(originalSums, unchangedSums) {
		t.Fatal("build changed authoring module files")
	}
}

// Removed overrides must fail as usage errors, rather than silently choosing a
// different dependency context. Outdir only applies to generated builds.
func TestModuleFlagUsage(t *testing.T) {
	for _, args := range [][]string{
		{"build", "-work"}, {"build", "-toolkit", "."}, {"build", "-replace", "example.test/m=."},
		{"check", "-outdir", "generated"}, {"inspect", "-dev"},
	} {
		var diagnostics bytes.Buffer
		if status := (seftool.Command{Stderr: &diagnostics}).Run(args); status != 2 {
			t.Errorf("%v status=%d: %s", args, status, &diagnostics)
		}
	}
}
