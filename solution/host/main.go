// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modern-engineering/prototype/application"
)

// Main is the Mode-P process skin over [Run]: it adds what a library
// must not own — signal handling, the graceful-shutdown sequence, and
// the exit code — and returns that code for main to hand os.Exit. One
// prebuilt binary calling Main serves any image compiled against its
// catalogue; examples/host is the committed demonstration.
//
// The first SIGINT/SIGTERM starts the wind-down: every running service
// with a Shutdown capability is asked to stop within cfg.Grace
// ([application.Runtime.Shutdown] with the grace context passed
// verbatim), then the runtime's context is cancelled and the last
// runners drain. A second signal skips straight to the cancel.
//
// The exit contract, A-12's restartable split:
//
//	0  clean: every service completed, or a signal wound the host down
//	1  wet failure: a driver refused, a service failed, a graceful
//	   stop overran its budget
//	2  configuration fault: bad image, unresolvable plan, unbound
//	   externs, a binding no flag accepts — nothing ran (or only
//	   provisioning did); restarting without changing inputs
//	   reproduces it
func Main(cfg Config) int {
	logw := cfg.log()

	// Subscribe before anything wet runs so a signal racing startup
	// parks in the channel buffer instead of killing the process.
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	rt, err := start(context.Background(), cfg)
	if err != nil {
		fmt.Fprintln(logw, err)
		var cfgErr *configError
		if errors.As(err, &cfgErr) {
			return 2
		}
		return 1
	}

	waited := make(chan error, 1)
	go func() { waited <- rt.Wait() }()

	select {
	case err := <-waited:
		// Nobody asked anything to stop: the services ran out of work
		// on their own. Clean completions are clean — the host imposes
		// no long-runningness — and one instance's failure is the
		// solution's failure.
		if err != nil && !canceled(err) {
			fmt.Fprintf(logw, "instance failed: %v\n", err)
			return 1
		}
		return 0

	case sig := <-signals:
		fmt.Fprintf(logw, "received %s; shutting down (grace %s)\n", sig, cfg.grace())
		hurried := make(chan struct{})
		defer close(hurried)
		go func() {
			select {
			case sig := <-signals:
				fmt.Fprintf(logw, "received second %s; cancelling\n", sig)
				rt.Cancel()
			case <-hurried:
			}
		}()

		// Shutdown alone never releases runners blocked on their
		// context — it deliberately leaves the runtime's context alone
		// — so the sequence is always Shutdown, then Cancel, then Wait.
		graceCtx, cancel := context.WithTimeout(context.Background(), cfg.grace())
		defer cancel()
		shutdownErr := rt.Shutdown(graceCtx)
		rt.Cancel()
		err := <-waited
		if shutdownErr != nil {
			fmt.Fprintf(logw, "shutdown overran: %v\n", shutdownErr)
			return 1
		}
		if err != nil && !canceled(err) {
			fmt.Fprintf(logw, "shutdown finished with failure: %v\n", err)
			return 1
		}
		fmt.Fprintln(logw, "shutdown complete")
		return 0
	}
}

// canceled reports whether err is the debris of a deliberate stop: the
// runtime's own cancel cause, or the context cancellation runners
// return once that cause fires.
func canceled(err error) bool {
	return errors.Is(err, application.ErrCanceled) || errors.Is(err, context.Canceled)
}
