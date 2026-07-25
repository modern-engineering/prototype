// Package leakprobe settles INTENT ruling 23: does a synctest bubble
// fail when it exits while a background goroutine is still live?
//
// Answer, from `go test ./...` on Go 1.26.5: a Close that only SIGNALS
// its goroutine passes, because the bubble waits for every goroutine in
// it to exit; a goroutine nobody signals fails the test with
// "panic: deadlock: main bubble goroutine has exited but blocked
// goroutines remain". Bubble exit is therefore a free leak detector,
// and signal-without-wait is not a defect to flag inside a bubble.
package leakprobe

import (
	"testing"
	"testing/synctest"
	"time"
)

// The cache fixture's shape: Close signals the janitor and returns
// without waiting for it. The bubble waits, the janitor exits, PASS.
func TestSignalledGoroutineDoesNotLeak(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stop := make(chan struct{})
		go func() {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for {
				select {
				case <-stop:
					return
				case <-tick.C:
				}
			}
		}()
		close(stop)
	})
}

// A goroutine nobody ever signals. Expected to FAIL with the deadlock
// panic quoted above; run it with -run TestRealLeakFailsTheBubble to
// see the diagnostic the bubble gives for free.
func TestRealLeakFailsTheBubble(t *testing.T) {
	t.Skip("expected failure; unskip to reproduce the deadlock panic")
	synctest.Test(t, func(t *testing.T) {
		go func() {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for range tick.C {
			}
		}()
		time.Sleep(time.Second)
	})
}
