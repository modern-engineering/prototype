// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// End-to-end proof of the run verb's signal choreography: this test
// drives sdl run the way an operator would — the one-command dev loop
// on the public example, SIGTERM included — the one run contract a
// script transcript cannot express; the run verb's stream and exit
// contracts live in testdata/script. It shares e2e_test.go's built
// CLI, helpers, and -short skip.
package main_test

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The A-13 dev-loop MoE end to end: one command takes the example
// sources to a running solution — provision audited,
// output value visibly wired into the deploys, all instances serving —
// and a SIGTERM at the driver reaches the hosted child for a graceful
// wind-down, exit 0. Stdout stays untouched throughout; the audit and
// the services own stderr.
func TestRunPingpong(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	cmd := exec.Command(sdlPath, "run", "-extern", "natsEndpoint=localhost:0", "examples/pingpong")
	cmd.Dir = repoRoot
	var stdout strings.Builder
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// The watchdog unblocks the scanner if the run never delivers or
	// never exits; the test then fails on the collected transcript.
	watchdog := time.AfterFunc(4*time.Minute, func() { _ = cmd.Process.Kill() })
	defer watchdog.Stop()

	// Scan until Ping1's first delivery — the line proving the var
	// wiring reached a running service — then ask for the wind-down
	// and drain the rest.
	var transcript strings.Builder
	signalled := false
	sc := bufio.NewScanner(stderr)
	for sc.Scan() {
		line := sc.Text()
		transcript.WriteString(line)
		transcript.WriteByte('\n')
		if !signalled && strings.Contains(line, "ping ping #") {
			t.Logf("sources to first delivery: %v", time.Since(start))
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatalf("signalling the driver: %v", err)
			}
			signalled = true
		}
	}
	err = cmd.Wait()
	got := transcript.String()
	t.Logf("run transcript:\n%s", got)
	if !signalled {
		t.Fatal("the run never logged a ping delivery")
	}
	if err != nil {
		t.Errorf("sdl run exited %v after SIGTERM, want 0", err)
	}

	for _, want := range []string{
		// PROVISION: the extern feeds the stand-in, whose output is
		// stored and audited unredacted (declared non-sensitive).
		`audit: extern natsEndpoint = "localhost:0" (site, substrate.Endpoint)`,
		"provision natsStandIn (attach substrate.StandIn)",
		`audit: natsStandIn.endpoint = "localhost:0" (extern natsEndpoint, instance)`,
		`audit: output natsStandIn.config = "stand-in://local" (substrate.StandIn)`,
		// DEPLOY: the output and the var land in the services' params.
		`audit: Ping1.nats = "stand-in://local" (output natsStandIn.config, instance)`,
		`audit: Ping1.target = "ping" (var echoSubject, instance)`,
		`audit: Pong.subject = "ping" (var echoSubject, instance)`,
		"running 3 instance(s)",
		// The wind-down: one forwarded signal, grace granted, exit 0.
		"received terminated; shutting down (grace 10s)",
		"shutdown complete",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("transcript is missing %q", want)
		}
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty (audit and service logs belong to stderr)", stdout.String())
	}
}
