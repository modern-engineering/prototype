package build_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// An adopter catalogue must compile against the artifact's requirements even
// when the invoking process has an unrelated workspace and conflicting flags.
// The generated host can then inspect declarations without transferring values.
func TestIsolatedCatalogueHost(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := filepath.Abs("testdata/catalogue")
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(catalogue, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	local := filepath.Join(temp, "catalogue with spaces")
	if err := os.CopyFS(local, os.DirFS(catalogue)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(temp, "go.work"), []byte("go 1.27.0\nuse ./missing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", filepath.Join(temp, "go.work"))
	t.Setenv("GOFLAGS", "-modfile="+filepath.Join(temp, "missing.mod")+" -overlay="+filepath.Join(temp, "missing.json"))
	definition := solution.Definition{
		Modules:  []solution.Module{{Path: "example.test/catalogue", Version: "v1.2.3"}, {Path: build.ToolkitModule, Version: "v0.8.0"}},
		Nodes:    []solution.Node{{Name: "service", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Service"}}},
		Bindings: []solution.Binding{{Node: "service", Input: "name", Value: solution.Literal("private-value-not-in-source")}},
	}
	result, err := build.Host(t.Context(), definition, build.Options{ToolkitDir: toolkit, ToolkitVersion: "v0.9.0", Replacements: []build.Replacement{{Module: "example.test/catalogue", Dir: local}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := result.Close(); err != nil {
			t.Error(err)
		}
	})
	source, err := os.ReadFile(result.Source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(source, []byte("private-value-not-in-source")) {
		t.Fatal("literal embedded in generated source")
	}
	query := exec.CommandContext(t.Context(), "go", "mod", "edit", "-json", result.ModFile)
	query.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	manifest, err := query.Output()
	if err != nil {
		t.Fatal(err)
	}
	var module struct {
		Module  struct{ Path string }
		Require []struct{ Path, Version string }
	}
	if err := json.Unmarshal(manifest, &module); err != nil {
		t.Fatal(err)
	}
	if module.Module.Path != "example.invalid/sefhost" {
		t.Fatalf("generated module = %q", module.Module.Path)
	}
	versions := make(map[string]string)
	for _, requirement := range module.Require {
		versions[requirement.Path] = requirement.Version
	}
	if versions["example.test/catalogue"] != "v1.2.3" || versions[build.ToolkitModule] != "v0.8.0" {
		t.Fatalf("requirements = %v", versions)
	}
	var data bytes.Buffer
	if err := sef.EncodeGob(&data, solution.Artifact{Version: "0.1", Definition: definition}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"check"}, {"inspect"}, {"inspect", "-inputs"}} {
		var output, diagnostics bytes.Buffer
		if err := result.Run(t.Context(), args, bytes.NewReader(data.Bytes()), &output, &diagnostics); err != nil {
			t.Fatalf("%v: %v: %s", args, err, &diagnostics)
		}
		if bytes.Contains(output.Bytes(), []byte("private-value-not-in-source")) {
			t.Fatal("inspection disclosed literal")
		}
		if len(args) == 2 && !strings.Contains(output.String(), `input "name" sensitive=true`) {
			t.Fatalf("declarations absent: %s", &output)
		}
	}
	var diagnostics bytes.Buffer
	err = result.Run(t.Context(), []string{"run"}, nil, nil, &diagnostics)
	if exit, ok := errors.AsType[*exec.ExitError](err); !ok || exit.ExitCode() != 2 {
		t.Fatalf("unsupported command = %v, want exit 2", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := result.Run(canceled, []string{"check"}, nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Run = %v", err)
	}
	unchanged, err := os.ReadFile(filepath.Join(local, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, unchanged) {
		t.Fatal("catalogue module was rewritten")
	}
	if _, err := os.Stat(filepath.Join(local, "go.sum")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source go.sum: %v", err)
	}
	work := result.WorkDir
	result.WorkDir = temp // Display paths do not grant ownership of caller files.
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("work still present: %v", err)
	}
	if _, err := os.Stat(local); err != nil {
		t.Fatalf("caller directory removed: %v", err)
	}
}

// Internal visibility is determined by the generated module's import path,
// not by the directory from which a host build was requested.
func TestFailedBuildOwnership(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := filepath.Abs("testdata/catalogue")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", t.TempDir())
	definition := solution.Definition{Modules: []solution.Module{{Path: "example.test/catalogue", Version: "v1.2.3"}}, Nodes: []solution.Node{{Name: "private", Definition: solution.Symbol{Package: "example.test/catalogue/internal/private", Name: "Service"}}}}
	options := build.Options{ToolkitDir: toolkit, Replacements: []build.Replacement{{Module: "example.test/catalogue", Dir: catalogue}}}
	result, err := build.Host(t.Context(), definition, options)
	if result != nil {
		t.Fatal("failed build retained files without KeepWork")
	}
	failure, ok := errors.AsType[*build.Error](err)
	if !ok || failure.Stage != "compilation" || failure.Output == "" {
		t.Fatalf("failure = %v, want compilation diagnostics", err)
	}
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed build left %d work directories", len(entries))
	}
	options.KeepWork = true
	result, err = build.Host(t.Context(), definition, options)
	if err == nil || result == nil {
		t.Fatalf("kept failure = %v, %v", result, err)
	}
	defer func() {
		if err := result.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := os.Stat(result.Source); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.ModFile); err != nil {
		t.Fatal(err)
	}
}

func TestModuleSelectionRequired(t *testing.T) {
	if _, err := build.Host(t.Context(), solution.Definition{}, build.Options{}); !errors.Is(err, build.ErrToolkitVersion) {
		t.Fatalf("unselected toolkit = %v", err)
	}
	definition := solution.Definition{Nodes: []solution.Node{{Definition: solution.Symbol{Package: "example.test/undeclared", Name: "Service"}}}}
	if _, err := build.Host(t.Context(), definition, build.Options{ToolkitVersion: "v0.1.0"}); !errors.Is(err, build.ErrModule) {
		t.Fatalf("undeclared catalogue = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := build.Host(ctx, solution.Definition{}, build.Options{ToolkitVersion: "v0.1.0"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled build = %v", err)
	}
}

// Package discovery must report a missing package at the selected version. It
// cannot silently change catalogue code to a newer release that has that package.
func TestMissingPackageDoesNotUpgrade(t *testing.T) {
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	proxy := t.TempDir()
	versions := filepath.Join(proxy, "example.test/versioned/@v")
	if err := os.MkdirAll(versions, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"v1.0.0", "v1.1.0"} {
		files := os.DirFS(filepath.Join("testdata/versions", version))
		mod, err := fs.ReadFile(files, "go.mod")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(versions, version+".mod"), mod, 0o600); err != nil {
			t.Fatal(err)
		}
		info := fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, version)
		if err := os.WriteFile(filepath.Join(versions, version+".info"), []byte(info), 0o600); err != nil {
			t.Fatal(err)
		}
		var archive bytes.Buffer
		writer := zip.NewWriter(&archive)
		err = fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := fs.ReadFile(files, path)
			if err != nil {
				return err
			}
			member, err := writer.Create("example.test/versioned@" + version + "/" + path)
			if err != nil {
				return err
			}
			_, err = member.Write(data)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(versions, version+".zip"), archive.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(versions, "list"), []byte("v1.0.0\nv1.1.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configured, err := exec.CommandContext(t.Context(), "go", "env", "GOPROXY").Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOPROXY", (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}).String()+","+strings.TrimSpace(string(configured)))
	t.Setenv("GONOSUMDB", "example.test/*")
	definition := solution.Definition{Modules: []solution.Module{{Path: "example.test/versioned", Version: "v1.0.0"}}, Nodes: []solution.Node{{Name: "service", Definition: solution.Symbol{Package: "example.test/versioned/sub", Name: "Service"}}}}
	result, err := build.Host(t.Context(), definition, build.Options{ToolkitDir: toolkit})
	if result != nil {
		t.Cleanup(func() {
			if err := result.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	if !errors.Is(err, build.ErrModule) {
		t.Fatalf("missing package build = %v; want rejected module discovery", err)
	}
}
