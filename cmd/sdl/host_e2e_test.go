// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main_test

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// One prebuilt binary serves any image compiled against its
// catalogue: examples/host — Mode-P, built once against the example
// catalogue, nothing generated per solution — enacts the committed
// pingpong image straight from disk, reaches serving, and a SIGTERM
// winds it down to exit 0. A smoke, not a matrix: the wet semantics
// are solution/host's pins, and the image bytes are TestBuildPingpong's.
func TestPrebuiltHostServesImage(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	hostPath := filepath.Join(t.TempDir(), "host")
	build := exec.Command("go", "build", "-o", hostPath, "./examples/host")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building examples/host: %v\n%s", err, out)
	}

	cmd := exec.Command(hostPath, "-image", filepath.Join("testdata", "pingpong.json"), "-extern", "natsEndpoint=localhost:0")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	watchdog := time.AfterFunc(2*time.Minute, func() { _ = cmd.Process.Kill() })
	defer watchdog.Stop()

	// Scan until the host reports serving, ask for the wind-down, and
	// drain the rest.
	var transcript strings.Builder
	signalled := false
	sc := bufio.NewScanner(stderr)
	for sc.Scan() {
		line := sc.Text()
		transcript.WriteString(line)
		transcript.WriteByte('\n')
		if !signalled && strings.Contains(line, "running 3 instance(s)") {
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatalf("signalling the host: %v", err)
			}
			signalled = true
		}
	}
	err = cmd.Wait()
	got := transcript.String()
	t.Logf("host transcript:\n%s", got)
	if !signalled {
		t.Fatal("the host never reported its instances running")
	}
	if err != nil {
		t.Errorf("host exited %v after SIGTERM, want 0", err)
	}
	if !strings.Contains(got, "shutdown complete") {
		t.Error("transcript is missing the graceful wind-down verdict")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (audit and service logs belong to stderr)", stdout.String())
	}
}
