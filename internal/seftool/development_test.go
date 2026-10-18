package seftool_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"

	"github.com/modern-engineering/prototype/internal/seftool"
)

// A local catalogue must affect the generated build only after development mode
// is selected. Its own replacements and caller workspace remain independent.
func TestDevelopmentRequiresOptIn(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	caller := filepath.Join(t.TempDir(), "caller with spaces")
	if err := os.CopyFS(caller, os.DirFS("testdata/development")); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(caller, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(caller)
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "missing.work"))
	t.Setenv("GOFLAGS", "-modfile="+filepath.Join(t.TempDir(), "missing.mod"))
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Context: t.Context(), Stdout: &output, Stderr: &diagnostics}
	for _, devFlag := range []string{"", "-dev=false"} {
		args := []string{"inspect", "-inputs", "-toolkit", toolkit}
		if devFlag != "" {
			args = append(args, devFlag)
		}
		args = append(args, "service.sef")
		if status := command.Run(args); status != 1 {
			t.Fatalf("production status=%d; want 1: %s", status, &diagnostics)
		}
		if output.Len() != 0 {
			t.Fatal("production failure produced output")
		}
		diagnostics.Reset()
	}
	if status := command.Run([]string{"inspect", "-inputs", "-dev", "-toolkit", toolkit, "-work", "service.sef"}); status != 0 {
		t.Fatalf("development status=%d: %s", status, &diagnostics)
	}
	if !strings.Contains(output.String(), `input "from-local"`) {
		t.Fatalf("local descriptor absent: %s", &output)
	}
	var work string
	for line := range strings.SplitSeq(diagnostics.String(), "\n") {
		if strings.HasPrefix(line, "WORK=") {
			work = strings.TrimPrefix(line, "WORK=")
		}
	}
	if work == "" {
		t.Fatal("missing retained work")
	}
	t.Cleanup(func() { _ = os.RemoveAll(work) })
	generated, err := os.ReadFile(filepath.Join(work, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := modfile.Parse("generated/go.mod", generated, nil)
	if err != nil {
		t.Fatal(err)
	}
	var localVersion, localDirectory string
	for _, requirement := range manifest.Require {
		if requirement.Mod.Path == "example.test/local/v2" {
			localVersion = requirement.Mod.Version
		}
	}
	for _, replacement := range manifest.Replace {
		if replacement.Old.Path == "example.test/unrelated" {
			t.Fatal("caller replacement leaked")
		}
		if replacement.Old.Path == "example.test/local/v2" {
			localDirectory = replacement.New.Path
		}
	}
	if localVersion != "v2.0.0" || localDirectory != caller {
		t.Fatalf("local requirement/replacement=%q/%q; want v2.0.0/%q", localVersion, localDirectory, caller)
	}
	unchanged, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unchanged, original) {
		t.Fatal("caller go.mod changed")
	}
	if _, err := os.Stat("go.sum"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("caller go.sum: %v", err)
	}
}
