// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// An Exec redirects the driver's last act: instead of the image
// emitter contract — stdout is the image, exit 1 means printed
// diagnostics — the built program runs as a process in its own right,
// with its own argv, the invocation's streams, and an exit code
// relayed verbatim as a [base.RelayedExit]. sdl run uses it: the
// generated program there is the tailored host, whose 0, 1, and 2
// carry the host's exit contract, not the compiler's.
type Exec struct {
	// Args is the child's argv after the program name.
	Args []string

	// WaitDelay bounds how long the child may keep running once the
	// invocation context is cancelled — the budget for its graceful
	// wind-down after the forwarded signal, sized by the caller from
	// the grace it granted the child. A child still running when the
	// delay elapses is killed. Zero waits forever.
	WaitDelay time.Duration

	// Stdout receives the child's stdout; nil means the inherited
	// stdout. Stderr rides the driver's own stderr either way, and
	// stdin is never wired: a hosted solution owns its streams but
	// reads no terminal.
	Stdout io.Writer
}

// execProgram runs the built program under the Exec contract. The
// child runs in the solution directory dir — the go test precedent,
// where the test binary runs in the package directory it was built
// from — so services resolving relative paths see the solution's own
// files, not the synthesized module context in workdir.
//
// Signal care: the child runs in its own process group, so a terminal
// interrupt is counted exactly once — it reaches this process alone,
// main translates it into the invocation context's cancellation, and
// the cancellation forwards one SIGTERM to the child. Group-shared
// delivery plus a forward would hand the child two signals, which a
// hosted solution reads as the operator's hurry-up and answers by
// skipping the very grace the wind-down was meant to grant. After the
// forward the driver waits for the child's own exit, WaitDelay the
// backstop.
func execProgram(ctx context.Context, workdir, dir string, ex *Exec, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, filepath.Join(workdir, "sol.bin"), ex.Args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	if ex.Stdout != nil {
		cmd.Stdout = ex.Stdout
	}
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = ex.WaitDelay

	// Start and Wait stay split so relayVerdict can tell a child that
	// ran from a cancellation that preempted the start.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("solution host: %v", err)
	}
	return relayVerdict(cmd.Wait())
}

// relayVerdict translates the child's outcome into the driver's. A
// child that exited carries its code back verbatim; in particular a
// clean exit after the forwarded signal — where os/exec substitutes
// the context's cancellation for the success it no longer vouches for
// — is the signalled wind-down, the run's clean outcome. Only a child
// that never got to exit on its own terms (killed past WaitDelay, or
// lost to the platform) reports as an ordinary driver failure.
func relayVerdict(err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() >= 0 {
		return &base.RelayedExit{Code: exit.ExitCode()}
	}
	return fmt.Errorf("solution host: %v", err)
}
