// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The wind-down sequence runs on whatever signal feed the config
// supplies, so these tests drive it deterministically under synctest's
// virtual clock: grace budgets are measured to the tick and an overrun
// costs no real seconds. The nil-feed default — the real kernel
// subscription — is pinned once in signal_test.go.

package host_test

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

// One signal winds a running solution down cleanly: services with a
// Shutdown capability are asked first, under a context whose deadline
// sits the whole cfg.Grace away — the budget passed verbatim —
// services without one are released by the cancel that follows, and a
// cooperative solution consumes none of the budget.
func TestMainWindsDownWithinGrace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctl := newStandControl()
		reports := make(chan report, 8)
		budget := make(chan time.Duration, 1)
		signals := make(chan os.Signal, 2)
		img := millImage(nil,
			deployRec("Forever", literal("subject", image.String("on"))), // count defaults to -1: run until cancelled
			parkRec("Parker"),
			attachRec("standIn", literal("endpoint", image.String("e"))),
		)
		var log strings.Builder
		done := make(chan int, 1)
		start := time.Now()
		go func() {
			done <- host.Main(host.Config{
				Image:     img,
				Catalogue: append(millCatalogue(ctl, reports), parkCatalogue(budget)...),
				Grace:     3 * time.Second,
				Log:       &log,
				Signals:   signals,
			})
		}()

		synctest.Wait() // every service is parked in Run; the signal races nothing
		signals <- syscall.SIGTERM
		if code := <-done; code != 0 {
			t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
		}
		if got := <-budget; got != 3*time.Second {
			t.Errorf("Shutdown's deadline sits %v away, want the whole grace (3s)", got)
		}
		if elapsed := time.Since(start); elapsed != 0 {
			t.Errorf("cooperative wind-down consumed %v of the grace, want none", elapsed)
		}
		wantLine(t, log.String(), `received terminated; shutting down (grace 3s)`)
		wantLine(t, log.String(), `shutdown complete`)
	})
}

// A second signal skips what remains of the grace: the hard cancel
// releases the runners at once — even one that ignores the polite ask
// — and the wind-down ends complete without the budget ever ticking.
func TestMainSecondSignalCancels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		signals := make(chan os.Signal, 2)
		var log strings.Builder
		done := make(chan int, 1)
		start := time.Now()
		go func() {
			done <- host.Main(host.Config{
				Image:     parkImage("Stubborn"),
				Catalogue: append(millCatalogue(newStandControl(), nil), parkCatalogue(nil)...),
				Grace:     time.Minute,
				Log:       &log,
				Signals:   signals,
			})
		}()

		synctest.Wait() // the stubborn service is parked in Run
		signals <- syscall.SIGTERM
		signals <- syscall.SIGTERM
		if code := <-done; code != 0 {
			t.Errorf("Main() = %d, want 0; log:\n%s", code, log.String())
		}
		if elapsed := time.Since(start); elapsed != 0 {
			t.Errorf("the second signal still cost %v, want an immediate cancel", elapsed)
		}
		wantLine(t, log.String(), `received second terminated; cancelling`)
		wantLine(t, log.String(), `shutdown complete`)
	})
}

// A service that outlives the polite ask costs the run its clean
// verdict, and no more than the budget: the grace context expires
// after exactly cfg.Grace — ten seconds when the config leaves it
// zero — the overrun names the instance, and Main reports the wet
// failure.
func TestMainReportsShutdownOverrun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		signals := make(chan os.Signal, 2)
		var log strings.Builder
		done := make(chan int, 1)
		start := time.Now()
		go func() {
			done <- host.Main(host.Config{
				Image:     parkImage("Stubborn"),
				Catalogue: append(millCatalogue(newStandControl(), nil), parkCatalogue(nil)...),
				Log:       &log,
				Signals:   signals,
			})
		}()

		synctest.Wait() // the stubborn service is parked in Run
		signals <- syscall.SIGTERM
		if code := <-done; code != 1 {
			t.Errorf("Main() = %d, want 1; log:\n%s", code, log.String())
		}
		if elapsed := time.Since(start); elapsed != 10*time.Second {
			t.Errorf("the overrun verdict took %v, want the default grace exactly (10s)", elapsed)
		}
		wantLine(t, log.String(), `shutdown overran: one: context deadline exceeded`)
	})
}
