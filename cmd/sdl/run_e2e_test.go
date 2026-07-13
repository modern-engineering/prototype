// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// End-to-end proof of the run verb: these tests drive sdl run the way
// an operator would — the one-command dev loop on the public example,
// signals included — and pin the teaching errors of solutions that
// compile but cannot be enacted. They share e2e_test.go's built CLI,
// helpers, and -short skip.
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

// TestRunPingpong is the A-13 dev-loop MoE end to end: one command
// takes the example sources to a running solution — provision audited,
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

// TestRunUnboundExtern pins the site gate through the whole stack: a
// solution declaring an extern the invocation never bound is a
// configuration fault — exit 2 before anything wet runs — and the
// message teaches the flag that fixes it.
func TestRunUnboundExtern(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	res := runSDL(t, repoRoot, "run", "examples/pingpong")
	if res.code != 2 {
		t.Fatalf("exit %d, want 2\n%s", res.code, res.stderr)
	}
	if want := "unbound extern natsEndpoint (substrate.Endpoint): pass -extern natsEndpoint=<value>"; !strings.Contains(res.stderr, want) {
		t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
	}
	if strings.Contains(res.stderr, "running") {
		t.Errorf("the gate must hold before anything runs:\n%s", res.stderr)
	}
}

// TestRunSample pins the teaching errors of the design sample, which
// compiles fine but cannot be enacted yet: its NATS slice is refused
// at plan time — slice lifecycle is deliberately unimplemented — as a
// configuration fault, exit 2, and the plan fault preempts the extern
// gate (the sample's unbound externs never get a say).
func TestRunSample(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	res := runSDL(t, repoRoot, "run", "examples/sample")
	if res.code != 2 {
		t.Fatalf("exit %d, want 2\n%s", res.code, res.stderr)
	}
	if want := "slice provisioning is not implemented: natsAccount provisions substrate.NATS as a slice (attach only)"; !strings.Contains(res.stderr, want) {
		t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
	}
	if strings.Contains(res.stderr, "unbound extern") {
		t.Errorf("plan faults must preempt the extern gate:\n%s", res.stderr)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty", res.stdout)
	}
}

// TestRunCompileFault pins run's exit semantics against build's: the
// same link fault that exits build 1 exits run 2, because under run
// the broken solution is the invocation's input and nothing wet has
// happened — restarting reproduces it. The diagnostic itself stays
// positioned either way.
func TestRunCompileFault(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "sol.sdl", `solution broken

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Gone as G
`)
	res := runSDL(t, dir, "run")
	if res.code != 2 {
		t.Fatalf("exit %d, want 2 (run semantics; build's would be 1)\n%s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "sol.sdl:5:8: unknown element Gone") {
		t.Errorf("diagnostic not positioned at the element reference:\n%s", res.stderr)
	}
}

// TestRunNoModule is the module-less proof for run, and with it the
// clean-completion half of the exit contract: a solution of finite
// work — one ping, count 1 — resolves its imports through the ambient
// proxy configuration, runs to completion with no signal anywhere,
// and exits 0.
func TestRunNoModule(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	env := hermeticProxyEnv(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "nomod.sdl"), `solution nomod

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1 {
	params {
		count: 1
		interval: 100ms
		target: "pong"
	}
}
`)
	start := time.Now()
	res := runSDLEnv(t, dir, env, "run")
	t.Logf("module-less sdl run: %v", time.Since(start))
	if res.code != 0 {
		t.Fatalf("sdl run exited %d\n%s", res.code, res.stderr)
	}
	for _, want := range []string{"running 1 instance(s)", "ping pong #1"} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
	}
	if strings.Contains(res.stderr, "shutdown") {
		t.Errorf("no signal was sent; completion must not read as a wind-down:\n%s", res.stderr)
	}
}
