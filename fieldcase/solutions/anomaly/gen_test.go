// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package anomaly

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenOutputsCurrent regenerates both artifact sets from the
// committed image and dev site and byte-compares them against the
// committed gen/ trees: the committed evidence must be exactly what
// the committed generator emits (drift gate), and since the comparison
// spans two fresh runs it doubles as the determinism check.
func TestGenOutputsCurrent(t *testing.T) {
	short(t)
	for _, mode := range []string{"host", "k8s"} {
		t.Run(mode, func(t *testing.T) {
			out := t.TempDir()
			cmd := exec.Command(genBin,
				"-image", "solutions/anomaly/anomaly.json",
				"-site", "solutions/anomaly/site/dev.site",
				"-mode", mode,
				"-o", out,
			)
			cmd.Dir = moduleRoot
			if msg, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("fieldcase-gen -mode %s: %v\n%s", mode, err, msg)
			}
			compareDirs(t, out, filepath.Join(moduleRoot, "solutions", "anomaly", "gen", mode))
		})
	}
}

// TestGenGates drives the generator's own MUST-bind gates: an empty
// site fails naming every extern with its type, and a site missing a
// consumed driver output fails naming it.
func TestGenGates(t *testing.T) {
	short(t)
	t.Run("UnboundExterns", func(t *testing.T) {
		empty := filepath.Join(t.TempDir(), "empty.site")
		if err := os.WriteFile(empty, nil, 0o666); err != nil {
			t.Fatal(err)
		}
		out, err := runGen(t, empty)
		if err == nil {
			t.Fatalf("generation succeeded despite an empty site:\n%s", out)
		}
		for _, want := range []string{
			"unbound extern tenantID (substrate.Secret): the site must bind extern.tenantID",
			"unbound extern siemServer (substrate.Endpoint)",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("gate output is missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("MissingDriverOutput", func(t *testing.T) {
		dev, err := os.ReadFile(filepath.Join(moduleRoot, "solutions", "anomaly", "site", "dev.site"))
		if err != nil {
			t.Fatal(err)
		}
		gapped := strings.Replace(string(dev), "driver.twinDB.password = dev-twin-pass\n", "", 1)
		if gapped == string(dev) {
			t.Fatal("fixture rot: dev.site no longer binds driver.twinDB.password")
		}
		path := filepath.Join(t.TempDir(), "gapped.site")
		if err := os.WriteFile(path, []byte(gapped), 0o666); err != nil {
			t.Fatal(err)
		}
		out, err := runGen(t, path)
		if err == nil {
			t.Fatalf("generation succeeded despite a missing consumed output:\n%s", out)
		}
		if !strings.Contains(out, "site does not configure consumed output driver.twinDB.password (slice substrate.Neo4j)") {
			t.Errorf("gate output does not name the missing output:\n%s", out)
		}
	})
}

// TestProdShapedSiteValid keeps the redacted prod-shaped example site
// honest: it must pass the same generation gates the dev site does
// (every extern bound, every consumed driver output configured).
func TestProdShapedSiteValid(t *testing.T) {
	short(t)
	out, err := runGen(t, filepath.Join(moduleRoot, "solutions", "anomaly", "site", "prod.example.site"))
	if err != nil {
		t.Fatalf("the prod-shaped site no longer generates: %v\n%s", err, out)
	}
}

// runGen invokes the generator in host mode against the committed
// image with the given site.
func runGen(t *testing.T, sitePath string) (string, error) {
	t.Helper()
	cmd := exec.Command(genBin,
		"-image", "solutions/anomaly/anomaly.json",
		"-site", sitePath,
		"-mode", "host",
		"-o", t.TempDir(),
	)
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// compareDirs asserts got and want hold the same file names with the
// same bytes.
func compareDirs(t *testing.T, got, want string) {
	t.Helper()
	gotNames := dirNames(t, got)
	wantNames := dirNames(t, want)
	if strings.Join(gotNames, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("file sets differ:\nregenerated: %v\ncommitted:   %v", gotNames, wantNames)
	}
	for _, name := range gotNames {
		g, err := os.ReadFile(filepath.Join(got, name))
		if err != nil {
			t.Fatal(err)
		}
		w, err := os.ReadFile(filepath.Join(want, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(g, w) {
			t.Errorf("%s: committed artifact differs from regeneration; rerun fieldcase-gen (see gen/README.md)", name)
		}
	}
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}
