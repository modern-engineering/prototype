// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The one kernel-integration pin: a real SIGTERM delivered by the
// kernel to the test process, which needs syscall.Kill and so a unix
// tag. Everything else about the wind-down sequence is pinned
// portably in shutdown_test.go through the Config.Signals seam.

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

// A nil Signals config subscribes the process's own SIGINT/SIGTERM,
// and kernel delivery drives the same wind-down the fed channel does:
// services running forever, one real SIGTERM, a clean exit 0 — the
// A-13 dev-loop ending.
func TestMainSubscribesKernelSignals(t *testing.T) {
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

	// The service is running, so the signal cannot outrun the wet
	// half; Main has subscribed, so the kernel hands the signal to
	// its channel rather than the default terminator.
	<-reports
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}

	// Real time bounds the verdict: twice the grace the test granted;
	// a hang here means the shutdown sequence lost a runner.
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("Main did not return within twice the grace; log:\n%s", log.String())
	}
	wantLine(t, log.String(), `shutdown complete`)
}
