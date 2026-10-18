package build_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// Version replacements and downloaded checksums are part of the portable module
// context. Production builds consume them without ambient workspace help.
func TestProductionModuleContext(t *testing.T) {
	moduleProxy(t)
	artifact := proxyArtifact()
	artifact.Module.Replace = []solution.ModuleReplacement{{
		Old: solution.Module{Path: "example.test/versioned", Version: "v1.0.0"},
		New: solution.Module{Path: "example.test/versioned", Version: "v1.1.0"},
	}}
	module, err := build.Tidy(t.Context(), artifact, build.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Sum) == 0 || len(artifact.Module.Sum) != 0 {
		t.Fatal("tidy did not return an independent checksum snapshot")
	}
	artifact.Module = module
	result, err := build.Host(t.Context(), artifact, build.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	for i := range artifact.Module.Sum {
		if artifact.Module.Sum[i].Path == "example.test/versioned" {
			artifact.Module.Sum[i].Hash = "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
		}
	}
	isolatedModuleCache(t)
	if _, err := build.Host(t.Context(), artifact, build.Options{}); err == nil {
		t.Fatal("production ignored conflicting artifact checksums")
	}
}

// Authoring may fill in transitive requirements, but cannot change a catalogue
// version merely because a missing package appeared in a newer release.
func TestTidyPreservesCatalogueSelection(t *testing.T) {
	moduleProxy(t)
	if _, err := build.Tidy(t.Context(), proxyArtifact(), build.Options{}); !errors.Is(err, build.ErrModule) {
		t.Fatalf("missing package discovery = %v", err)
	}
	artifact := proxyArtifact()
	artifact.Definition = solution.Definition{}
	module, err := build.Tidy(t.Context(), artifact, build.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var retained, authenticated bool
	for _, requirement := range module.Require {
		if requirement.Path == "example.test/versioned" && requirement.Version == "v1.0.0" {
			retained = true
		}
	}
	for _, sum := range module.Sum {
		if sum.Path == "example.test/versioned" && sum.Version == "v1.0.0/go.mod" {
			authenticated = true
		}
	}
	if !retained || !authenticated {
		t.Fatalf("unused requirement retained=%v authenticated=%v", retained, authenticated)
	}
	artifact.Module = module
	result, err := build.Host(t.Context(), artifact, build.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
}

// Temporary workspace substitutions cannot escape into a portable artifact.
func TestTidyKeepsAuthoredReplacements(t *testing.T) {
	t.Setenv("GOWORK", "off")
	artifact := localArtifact(t)
	artifact.Module.Replace[0].New.Path = "../.."
	before := artifact.Module.Replace[0]
	module, err := build.Tidy(t.Context(), artifact, build.Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	if module.Replace[0] != before {
		t.Fatalf("relative replacement changed: %v", module.Replace[0])
	}
	artifact.Module.Replace = artifact.Module.Replace[1:]
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// The toolkit is discovered from the caller's enclosing module.
	module, err = build.Tidy(t.Context(), artifact, build.Options{Dev: true, Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Replace) != 1 || module.Replace[0] != artifact.Module.Replace[0] {
		t.Fatalf("development overlay persisted: %v", module.Replace)
	}
}

func proxyArtifact() solution.Artifact {
	return solution.Artifact{Version: "0.1", Module: solution.ModuleContext{
		Path: "example.test/service", Go: "1.27.0",
		Require: []solution.Module{{Path: build.ToolkitModule, Version: "v0.0.0"}, {Path: "example.test/versioned", Version: "v1.0.0"}},
	}, Definition: solution.Definition{Nodes: []solution.Node{{Name: "service", Definition: solution.Symbol{Package: "example.test/versioned/sub", Name: "Service"}}}}}
}

func moduleProxy(t *testing.T) {
	t.Helper()
	proxy := t.TempDir()
	for _, source := range []struct{ path, version, dir string }{
		{build.ToolkitModule, "v0.0.0", "testdata/toolkit"},
		{"example.test/versioned", "v1.0.0", "testdata/versions/v1.0.0"},
		{"example.test/versioned", "v1.1.0", "testdata/versions/v1.1.0"},
	} {
		versions := filepath.Join(proxy, source.path, "@v")
		if err := os.MkdirAll(versions, 0o700); err != nil {
			t.Fatal(err)
		}
		files := os.DirFS(source.dir)
		mod, err := fs.ReadFile(files, "go.mod")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(versions, source.version+".mod"), mod, 0o600); err != nil {
			t.Fatal(err)
		}
		info := fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, source.version)
		if err := os.WriteFile(filepath.Join(versions, source.version+".info"), []byte(info), 0o600); err != nil {
			t.Fatal(err)
		}
		list, err := os.OpenFile(filepath.Join(versions, "list"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintln(list, source.version); err != nil {
			t.Fatal(err)
		}
		if err := list.Close(); err != nil {
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
			member, err := writer.Create(source.path + "@" + source.version + "/" + path)
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
		if err := os.WriteFile(filepath.Join(versions, source.version+".zip"), archive.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOPROXY", (&url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}).String())
	t.Setenv("GOSUMDB", "off")
	isolatedModuleCache(t)
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "missing.work"))
}

func isolatedModuleCache(t *testing.T) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("GOMODCACHE", cache)
	t.Cleanup(func() {
		command := exec.Command("go", "clean", "-modcache")
		command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "GOMODCACHE="+cache)
		if output, err := command.CombinedOutput(); err != nil {
			t.Errorf("clean fixture module cache: %v: %s", err, output)
		}
	})
}
