---
status: draft
date: 2026-05-27
---

# A5. Termination Signaling

## Context

Long-running components ([A1](A1-scope.md)) participate in their own
termination. The framework needs to express both the **trigger** that
initiates shutdown and the **confirmation** that reports the outcome. This
analysis observes that these are separate concerns, identifies the carriers
each admits, and notes how they compose.

## The Two Channels

**Trigger (push, runtime-initiated).** Something outside the component decides
shutdown should begin. The trigger arrives at the component as a signal it
must observe and respond to.

**Confirmation (pull, call-stack return).** The component reports that it has
finished, including why. The confirmation flows back through the call that
started the component.

These are orthogonal. A component can receive a trigger without exiting (the
trigger initiates a drain; the component exits when drain completes). A
component can exit without receiving a trigger (work completed naturally, or
an invariant was violated). The framework needs both, separately.

## Survey of Trigger Carriers

**OS signals (SIGTERM, SIGINT, SIGHUP).** Delivered by the kernel to the
process; process-wide, not per-instance. The idiomatic Go bridge,
`signal.NotifyContext`, returns a `context.Context` that cancels when the
named signal arrives. The bridge converts a process-level event into a
Go-idiomatic per-call object.

**`context.Context` cancellation.** The de facto Go convention. Cancellation
propagates via `Done() <-chan struct{}` and a typed `Err()` (or `Cause` in
modern Go). Composable: derived contexts (`WithTimeout`, `WithCancel`,
`WithDeadline`) admit fine-grained scoping. Carries values too, but
cancellation is the load-bearing property in this section.

**Explicit shutdown methods.** `http.Server.Shutdown(ctx)`. The component
receives a direct method call to begin shutdown; the `ctx` parameter bounds
the drain duration. The pattern decouples "begin shutting down" from "complete
by deadline." A capability interface ([A4](A4-component-contract.md))
expresses the same idea in this framework.

**Message-passing.** Erlang `gen_server`, actor systems. The runtime sends a
"shutdown" message; the actor's `terminate` callback runs. Out of band for
Go, but illustrative of the trigger-as-event model.

**systemd `ExecStop`.** The unit file declares a separate command for
shutdown. systemd invokes it before SIGTERM. Effectively a runtime-level
shutdown method. Most Go services treat `ExecStop` as redundant and rely on
SIGTERM observation.

**systemd `sd_notify` notify socket.** For status updates during normal
operation (`READY=1`, `RELOADING=1`, `STATUS=`, `WATCHDOG=1`), not specifically
for shutdown trigger. Confirms that long-running services routinely have a
separate channel for state, distinct from termination.

**Observation.** `context.Context` is this framework's natural trigger
carrier. The loader translates OS-level signals into context cancellation
once, process-wide, and fans the resulting ctx out to every instance.
Components observe `ctx.Done()` and respond.

## Survey of Confirmation Carriers

**Go `error` return.** Used by `http.Server.ListenAndServe`,
`errgroup.Group.Go(func() error)`, `peterbourgon/ff`'s `Command.Exec`, and the
broader ecosystem. Composable through `fmt.Errorf("instance %s: %w", id,
err)`; inspectable at the call site through `errors.Is` and `errors.As`;
testable through `require.NoError`, `require.ErrorIs`.

**Status callbacks during run.** systemd `sd_notify`, OpenTelemetry events.
Excellent for continuous state updates. A category mismatch for termination,
which is one-shot and final.

**Pushed events to a sink.** Logs, metrics. An observability complement, not
control. A loader can derive termination state from logs, but it should not
have to.

**Never returning at all.** Go `func main` semantics, `os.Exit` inside library
code. Disqualifying for multi-instance hosting: one component's `os.Exit`
kills siblings. The trap that `singlechecker`, `multichecker`, and `ff`'s CLI
top-level mode fall into.

**Observation.** `error` return is this framework's natural confirmation
carrier. It composes with `errgroup` (the natural loader implementation for
fanning out N runners), with test harness assertions, and with exit-code
propagation up the call stack.

## Composition

The two channels together give the framework:

- **Trigger:** `ctx`, cancelled by the loader on signal or supervisor
  command, fans out to all instances.
- **Confirmation:** `error` returned from `Run(ctx)`, aggregated by the
  loader.

A component's idiomatic shape:

```go
func (a *myApp) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            // drain or cleanup
            return ctx.Err() // or nil, or a structured exit reason
        case work := <-a.queue:
            // process
        }
    }
}
```

The same shape extends to capability interfaces ([A4](A4-component-contract.md)).
A `Shutdowner.Shutdown(ctx)` call is a parallel trigger: another goroutine
signals "begin shutting down" while `Run` is still executing. `Run` sees its
own ctx cancel and unwinds; `Shutdown` returns when the drain is done. Two
methods, two ctxs, one component, no globals.

## Optional Structuring: Typed Exit Reasons

Future iterations may benefit from typed sentinel chains:

- `ErrShuttingDown`: normal response to trigger.
- `ErrInvariantViolated`: the component refused to continue.
- `ErrLeadershipLost`: a peer instance signaled takeover.
- `ErrDependencyLost`: a required external resource is unreachable.

The base contract (`error`) does not require this. Loaders that want
classification can wrap and inspect. The requirements may defer typed
sentinels concretely until a loader has a stated need to discriminate.

## Open Questions

- Should the framework canonicalize a small set of exit-reason sentinels?
  Current direction: defer.
- Should `context.Cause` be preferred over `ctx.Err()` to expose the reason
  for cancellation? Modern Go (1.20+) supports it; the requirements will
  choose.
- Should the loader treat a `nil` return from `Run` (component completed
  naturally) differently from a `nil` return after `ctx.Done()`? The two are
  indistinguishable at the call site without convention; the framework may
  recommend the latter return `ctx.Err()` to be explicit.
