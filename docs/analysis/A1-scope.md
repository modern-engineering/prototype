---
status: draft
date: 2026-05-27
---

# A1. Scope: Long-Running Applications, Not Command-Line Tools

## Context

The framework prototyped here addresses one class of program. The term
"application" carries different connotations across the Go ecosystem, and the
inspirations available to this project (the `golang.org/x/tools/go/analysis`
framework, `golangci-lint`, `peterbourgon/ff`) span the spectrum. This analysis
fixes the project's scope so subsequent analyses can judge inspirations against
it.

## The Spectrum

Programs in the Go ecosystem fall, roughly, into three families.

- **Command tools.** Receive arguments, perform a task, exit. Examples: `gofmt`,
  `sed`, build helpers, `kubectl` one-shot subcommands, `gopls` in offline
  modes, `ff.Command.Exec` used as a CLI entrypoint. The lifecycle is "run to
  completion or fail." Termination is unremarkable; the OS reclaims the process
  when `main` returns.
- **Long-running applications.** Bind a process to ongoing work. Examples: HTTP
  servers, gRPC services, pub-sub clients, stream processors, reconcilers,
  schedulers, background workers. The lifecycle is "run until told to stop."
  Termination is an event the program participates in (graceful drain,
  in-flight request completion).
- **Hybrid shapes.** `golangci-lint` runs as a batch tool but maintains
  persistent caches; `kubectl proxy` is a long-running mode of an otherwise
  command-shaped tool.

This project's in-scope target is the long-running family.

## What the Scope Buys

Establishing the long-running scope makes several framework concerns
first-class:

- **Signal-driven shutdown.** SIGTERM is the normal end-of-life event, not an
  exception. The framework must offer a place for components to observe
  shutdown and respond.
- **Multi-instance hosting.** A program may run several parameterized instances
  of the same component within a single process. The parameterization analysis
  ([A3](A3-parameterization.md)) develops this.
- **Long-form parameterization.** Configuration comes from environment, files,
  secret stores, control planes, command-line flags. Programs in this scope
  routinely accept richer inputs than `argv` slots.
- **Structured observability.** Logs, metrics, traces, health probes are
  first-class. The component contract should let the framework grow these
  capabilities over time.
- **Cooperative termination.** The component must communicate why it exited so
  the host process can decide whether to restart, escalate, or accept.

## What the Scope Excludes

Equally important is what the scope is not:

- **Run-once-exit.** The framework does not optimize for "parse args, do thing,
  exit." Components that complete naturally remain valid (Run returns `nil`),
  but the contract is shaped around the assumption that running until told to
  stop is the common case.
- **Immediate-on-error termination.** The framework does not assume one
  component's failure ends the process. Multi-instance hosting depends on
  isolating failures.
- **Positional-argument semantics.** Component parameterization comes through
  declared slots (working name: flags). Bare positional args are an artifact of
  CLI shape, not application shape.

## Why Inspirations from Outside the Scope Still Matter

The single-component and multi-component drivers in
`golang.org/x/tools/go/analysis`, `peterbourgon/ff`'s `Exec` shape, and
`golangci-lint`'s registry mechanics are all command-tool-shaped or
batch-shaped. Their structural choices (component-as-value introspection,
layered flag sources, error-returning entry points, builder-style registration)
translate to the long-running family. Their lifecycle choices (read args, run,
exit) do not. Subsequent analyses cite these inspirations for shape and
explicitly note where their lifecycle assumptions diverge.

## Implications for Subsequent Analyses

- [A2: Architecture](A2-architecture.md) judges hosting patterns by whether
  they admit long-running multi-instance use.
- [A3: Parameterization](A3-parameterization.md) treats input-from-everywhere
  as the default; A2 and A3 together explain why a registry is necessary.
- [A4: Component contract](A4-component-contract.md) admits one base method
  plus capability extensions, because long-running components grow
  observability surfaces over time.
- [A5: Termination](A5-termination.md) treats trigger and confirmation as
  separate channels because graceful shutdown is the normal case, not an
  exception.
- [A8: Ambient services](A8-ambient-services.md) draws sharp distinctions
  because long-running multi-instance programs routinely confuse per-instance
  ambient state with per-process shared resources.

## Open Questions

- **Hybrid-shape support.** The framework may, over time, want to accommodate
  components that complete normally (workers that drain a queue and exit). This
  analysis records the current scope (long-running) and leaves hybrid
  accommodation as a deferred concern. Nothing in the current contract
  precludes it; nothing in the current contract optimizes for it.
