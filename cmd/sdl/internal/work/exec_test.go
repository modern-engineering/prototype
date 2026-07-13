// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// exitError runs a real shell to craft an honest *exec.ExitError; the
// translation under test must read the same states os/exec produces,
// not hand-built ones.
func exitError(t *testing.T, script string) error {
	t.Helper()
	err := exec.Command("sh", "-c", script).Run()
	if err == nil {
		t.Fatalf("sh -c %q exited clean; the fixture needs a failure", script)
	}
	return err
}

// TestRelayVerdict pins the child-outcome translation of the Exec
// contract: clean exits — the plain one and the post-cancellation one
// os/exec reports as the context's error — are the run's success; an
// exit code comes back verbatim as a RelayedExit; and a child that
// never got to exit on its own terms (a signal death, a platform
// fault) is an ordinary driver error, exit 2 territory.
func TestRelayVerdict(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		if err := relayVerdict(nil); err != nil {
			t.Errorf("relayVerdict(nil) = %v, want nil", err)
		}
	})

	t.Run("signalled wind-down", func(t *testing.T) {
		// A cancelled context with a child that then exited 0: os/exec
		// substitutes ctx.Err() for the success it cannot vouch for.
		if err := relayVerdict(context.Canceled); err != nil {
			t.Errorf("relayVerdict(context.Canceled) = %v, want nil", err)
		}
	})

	t.Run("exit code", func(t *testing.T) {
		err := relayVerdict(exitError(t, "exit 3"))
		var relay *base.RelayedExit
		if !errors.As(err, &relay) || relay.Code != 3 {
			t.Errorf("relayVerdict(exit 3) = %v, want RelayedExit{3}", err)
		}
	})

	t.Run("signal death", func(t *testing.T) {
		err := relayVerdict(exitError(t, "kill -TERM $$"))
		var relay *base.RelayedExit
		if errors.As(err, &relay) {
			t.Fatalf("relayVerdict(signal death) = RelayedExit{%d}, want an ordinary error", relay.Code)
		}
		if err == nil {
			t.Fatal("relayVerdict(signal death) = nil, want an ordinary error")
		}
	})

	t.Run("platform fault", func(t *testing.T) {
		err := relayVerdict(errors.New("fork failed"))
		if err == nil || errors.As(err, new(*base.RelayedExit)) {
			t.Errorf("relayVerdict(platform fault) = %v, want an ordinary error", err)
		}
	})
}
