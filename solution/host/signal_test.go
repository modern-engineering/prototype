// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The signal tests deliver real SIGTERMs to the test process itself,
// which needs syscall.Kill; the exit-contract and wet-half tests stay
// portable in the untagged files.

//go:build unix

package host_test

import (
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

// sigterm signals the test process itself: Main has subscribed, so
// the kernel hands the signal to its channel rather than the default
// terminator.
func sigterm(t *testing.T) {
	t.Helper()
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}
}

// waitExit collects Main's verdict, bounded by twice the grace the
// test granted: a hang here means the shutdown sequence lost a runner.
func waitExit(t *testing.T, done <-chan int, log *strings.Builder) int {
	t.Helper()
	select {
	case code := <-done:
		return code
	case <-time.After(10 * time.Second):
		t.Fatalf("Main did not return within twice the grace; log:\n%s", log.String())
		return -1
	}
}

// TestMainShutsDownOnSignal drives the A-13 dev-loop ending: services
// running forever, one SIGTERM, a clean exit 0 inside the grace. The
// echo services carry no shutdown capability, so this exercises the
// Cancel-then-Wait half of the sequence.
func TestMainShutsDownOnSignal(t *testing.T) {
	ctl := newStandControl()
	reports := make(chan report, 8)
	img := millImage(nil,
		deployRec("Forever", literal("subject", image.String("on"))), // count defaults to -1: run until cancelled
		attachRec("standIn", literal("endpoint", image.String("e"))),
	)
	var log strings.Builder
	done := make(chan int, 1)
	go func() {
		done <- host.Main(host.Config{
			Image:     img,
			Catalogue: millCatalogue(ctl, reports),
			Grace:     5 * time.Second,
			Log:       &log,
		})
	}()

	<-reports // the service is running; the signal cannot outrun it
	sigterm(t)
	if code := waitExit(t, done, &log); code != 0 {
		t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
	}
	wantLine(t, log.String(), `shutdown complete`)
}

// TestMainGrantsGraceToShutdowners proves the graceful half: a service
// with a Shutdown capability is asked first, under a context carrying
// the grace deadline verbatim, and its voluntary stop ends the run
// cleanly.
func TestMainGrantsGraceToShutdowners(t *testing.T) {
	started := make(chan struct{}, 1)
	sawDeadline := make(chan bool, 1)
	var log strings.Builder
	done := make(chan int, 1)
	go func() {
		done <- host.Main(host.Config{
			Image:     parkImage("Parker"),
			Catalogue: append(millCatalogue(newStandControl(), nil), parkCatalogue(started, sawDeadline)...),
			Grace:     5 * time.Second,
			Log:       &log,
		})
	}()

	<-started
	sigterm(t)
	if code := waitExit(t, done, &log); code != 0 {
		t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
	}
	if !<-sawDeadline {
		t.Error("Shutdown ran without the grace deadline on its context")
	}
	wantLine(t, log.String(), `shutdown complete`)
}
