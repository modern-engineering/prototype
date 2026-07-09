// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The stage-2 proof harness: these tests build the generated host and
// the generator once, then drive them as a deployer would — scenario site
// files, scenario process lifecycle, scenario signals. They are the slow path;
// -short skips everything that needs the Go toolchain.
package anomaly

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	moduleRoot string // the fieldcase module root
	hostBin    string // the compiled generated host (gen/host)
	genBin     string // the compiled generator (cmd/fieldcase-gen)
)

func TestMain(m *testing.M) {
	flag.Parse()
	code := func() int {
		if !testing.Short() {
			dir, err := os.MkdirTemp("", "anomaly-stage2-")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			defer os.RemoveAll(dir)
			if err := buildBinaries(dir); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		return m.Run()
	}()
	os.Exit(code)
}

// buildBinaries compiles the generated host and the generator from the
// module root, so the tests exercise exactly the committed sources.
func buildBinaries(dir string) error {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return fmt.Errorf("locating module root: %v", err)
	}
	moduleRoot = strings.TrimSpace(string(out))

	hostBin = filepath.Join(dir, "anomaly-host")
	genBin = filepath.Join(dir, "fieldcase-gen")
	for target, pkg := range map[string]string{
		hostBin: "./solutions/anomaly/gen/host",
		genBin:  "./cmd/fieldcase-gen",
	} {
		cmd := exec.Command("go", "build", "-o", target, pkg)
		cmd.Dir = moduleRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("building %s: %v\n%s", pkg, err, out)
		}
	}
	return nil
}

// short skips toolchain-driving tests under -short.
func short(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("drives built binaries; skipped in -short mode")
	}
}
