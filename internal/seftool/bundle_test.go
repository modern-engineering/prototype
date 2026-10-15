package seftool_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

// Selection belongs to a bundle, while an exported artifact must retain its
// meaning when moved beneath a different or invalid module manifest.
func TestBundleSelectionAndSnapshot(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	if err := os.CopyFS(project, os.DirFS("testdata/bundle")); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(project, "service.sef"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "other.sef"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"convert", project}); status != 1 {
		t.Fatalf("ambiguous selection status=%d", status)
	}
	if output.Len() != 0 {
		t.Fatal("ambiguous selection produced an artifact")
	}
	if status := command.Run([]string{"convert", "-artifact", "service.sef", "-to", "json", project}); status != 0 {
		t.Fatalf("selection status=%d: %s", status, &diagnostics)
	}
	data := bytes.Clone(output.Bytes())
	artifact, err := sef.DecodeJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Module.Path != "example.test/service-solution" {
		t.Fatalf("module=%q", artifact.Module.Path)
	}
	unrelated := t.TempDir()
	if err := os.WriteFile(filepath.Join(unrelated, "go.mod"), []byte("invalid manifest"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(unrelated, "snapshot.json.sef")
	if err := os.WriteFile(snapshot, data, 0o600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if status := command.Run([]string{"dump", snapshot}); status != 0 {
		t.Fatalf("standalone status=%d: %s", status, &diagnostics)
	}
	if !bytes.Equal(output.Bytes(), data) {
		t.Fatal("standalone artifact borrowed surrounding module state")
	}
}

// An encoding without module support must reject the conversion even when the
// supplied context is incomplete; semantic invalidity does not permit data loss.
func TestConversionRequiresModuleSupport(t *testing.T) {
	for _, module := range []solution.ModuleContext{
		{Path: "example.test/solution"},
		{Require: []solution.Module{{Path: "example.test/catalogue", Version: "v1.0.0"}}},
	} {
		var source, output, diagnostics bytes.Buffer
		if err := sef.EncodeJSON(&source, solution.Artifact{Version: sef.Version, Module: module}); err != nil {
			t.Fatal(err)
		}
		command := seftool.Command{Stdin: &source, Stdout: &output, Stderr: &diagnostics}
		if status := command.Run([]string{"convert", "-from", "json", "-to", "cue"}); status != 1 {
			t.Fatalf("unsupported conversion status=%d", status)
		}
		if output.Len() != 0 {
			t.Fatal("unsupported conversion produced output")
		}
	}
}
