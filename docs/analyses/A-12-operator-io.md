---
status: draft
since: 2026-07-06
---

# Operator I/O: The Process Contract over OS Primitives

## Context

"I/O" in this analysis means the channels by which human operators — and the
controllers acting for them — guide, configure, and measure applications. It
does _not_ mean dataflow between applications. An earlier framework we built
made that conflation: it formalized inter-application messaging as a first-class
framework concept, binding components to broker topics through declared roles.
The generalization failed twice over — not every modern backend communicates
through a broker, and brokers differ enough (delivery, ordering, retention
semantics) that a unifying facade erases the reasons for choosing one. The
lesson stands: the solution layer models the _operator_ contract;
application-to-application dataflow rides in ordinary parameter values that
applications interpret themselves.

The operator contract bottoms out in OS primitives. A Unix process can be
reached by its controller in only a few ways — argv, environment, files,
standard streams, signals, inherited descriptors — and anything higher-level (a
configuration service, a secrets manager, an admin RPC) must itself be
bootstrapped through those primitives. The layer models the bootstrap; what
applications build above it is their business.

One litmus governs the survey, inherited from [A-09 Solution Layer]: every
channel that operators of production deployment descriptors actually employ must
either be supported here or be explicitly named the wrong tool. A later
requirement will lock that coverage rule.

## The Channel Inventory

| Channel                     | Direction | Verdict                                        |
| --------------------------- | --------- | ---------------------------------------------- |
| argv (flags)                | in        | mainline configuration surface                 |
| environment variables       | in        | supported, declared; ambient reads advertised  |
| files read (incl. FIFOs)    | in        | supported as declared documents                |
| stdin                       | in        | edge: config-as-file for jobs; not services    |
| signals                     | in        | supported through capability interfaces        |
| inherited file descriptors  | in        | supported as handle-shaped inputs              |
| identity and envelope       | in        | controller intent, never application config    |
| tty                         | in        | wrong tool: services never see a terminal      |
| stderr                      | out       | logs and diagnostics                           |
| stdout                      | out       | measurements and data                          |
| exit code                   | out       | supported; taxonomy open                       |
| readiness signaling         | out       | supported through capability interfaces        |
| files written (state dirs)  | out       | supported as declared handles; pid files wrong |
| /proc, rusage, cgroup stats | out       | controller-side observation; app does nothing  |

Elaborations, where the table compresses too much:

**Streams.** Diagnostics go to stderr; stdout is reserved for measurements and
data. This is the Unix filter tradition (and Go's own `slog` default) rather
than the logs-to-stdout fashion; it keeps one channel machine-read. The
convention only pays if the application library enforces it, so individual
components cannot drift.

**Signals.** A signal is a mechanism, not a meaning. The meanings — stop
gracefully, reload — surface as capability interfaces on the component
([A-04 Component Contract], [A-05 Termination]); a per-process loader translates
SIGTERM to the termination capability, while a single-process host invokes the
same capability as a method call. A reload capability (SIGHUP's classic meaning)
is a natural sibling, not yet designed.

**Environment.** Readable by anyone at the same privilege via the proc
filesystem, so platform policy may veto secrets in it; the layer must not assume
env delivery for sensitive values.

**Inherited descriptors.** Socket activation and credential directories pass
already-open handles; the shape generalizes: a component receives a listener,
never "a port number to bind". Ports bind ambiently; handles are given.
(Capability-based systems and their preopened descriptors are the precedent.)

**Identity and envelope.** uid/gid, working directory, resource limits, and
scheduling class configure _how the process runs_, which is deployment intent
from [A-09 Solution Layer]'s anatomy — never something the application reads as
configuration.

**Exit codes.** Restart policies read them, so "configuration error, do not
restart" must be distinguishable from "runtime failure, do restart". The library
should map returned errors onto a small taxonomy; its contents are open.

**Readiness.** Notify sockets and probe endpoints are per-environment mechanisms
for one logical output the component already exposes as a health capability;
each environment wires its own mechanism to it.

## Modeling Inputs: Three Paths

How should the catalogue describe a component's input surface, beyond the flags
of [A-03 Parameterization]?

**Mechanism kinds.** Each input declares its transport: an env-kind input, a
file-kind input, a port-kind input. It reads naturally and fails the
single-process litmus by construction: two instances in one process cannot hold
different values of the same environment variable, two hardcoded file paths
collide in one filesystem, and a port number forces an ambient bind.

**Semantic classes.** Each input declares what it _is_ — a value, a secret, a
configuration document, a listener, an endpoint — and each environment maps
classes to its own mechanisms: a document becomes a mounted secret volume on
Kubernetes, a credential file under systemd, a direct in-memory injection in the
single-process host. No mount instruction ever appears in a definition; the
environment owns delivery. A policy dimension fits naturally: a class may admit
several deliveries, and platform policy picks (never secrets via env, say).

**One undifferentiated block.** All inputs in a single block with no
distinctions; the wiring alone (which symbol feeds which input) is the
declaration. This is the lightest to author and read, and it pushes every
distinction the other paths express — sensitivity, delivery constraints,
handle-versus-value — into the catalogue's type information or out of reach.
Whether the type system can carry all of it is exactly the open question; the
path is not dismissed.

This analysis recommends semantic classes, with the undifferentiated block kept
live as a simplification target if type information proves sufficient.

## Ambient Surfaces

Beyond the mainline flag surface, real applications touch surfaces nobody
declared: an SDK that reads its region from the environment on its own, a
library that probes a well-known path for its configuration file. Three
postures:

- **Ignore.** The status quo everywhere; the coupling exists but is written down
  nowhere, and the first missed env var at a new site is a production incident.
- **Forbid.** Unrealistic; the SDKs exist and will not stop.
- **Advertise.** The descriptor declares the ambient surfaces its application
  touches, so sites can supply, permit, or veto them:

```go
var S3Backup = application.Descriptor{
    Name: "s3-backup",
    // ...
    Ambient: []application.Ambient{
        application.AmbientEnv{Name: "AWS_REGION", Doc: "read by the AWS SDK"},
        application.AmbientFile{Path: "$HOME/.aws/credentials", Doc: "SDK fallback chain"},
    },
}
```

This analysis recommends advertising. Two consequences follow. An advertised
ambient environment variable implies a co-location constraint — two instances
needing different values cannot share one process environment — which connects
this surface to the compute-isolation question [A-09 Solution Layer] leaves
open. And the word "ambient" here names the process-boundary counterpart of
[A-08 Ambient Services]' in-process services: same instinct (make the implicit
explicit), different layer.

## Open Questions

- **Input grouping.** The undifferentiated block above: whether catalogue type
  information can absorb the distinctions, and what is lost beyond readability
  if it cannot.
- **Measurement protocol.** stdout is reserved for measurements; the format and
  framing a controller may rely on are undesigned.
- **Exit-code taxonomy.** The classes worth distinguishing and their numbering.
- **Reload.** Whether a reload capability joins termination, and what it means
  under each archetype.
- **Readiness unification.** One health capability serving notify sockets, probe
  endpoints, and in-process checks without per-environment component code.

[A-03 Parameterization]: A-03-parameterization.md
[A-04 Component Contract]: A-04-component-contract.md
[A-05 Termination]: A-05-termination.md
[A-08 Ambient Services]: A-08-ambient-services.md
[A-09 Solution Layer]: A-09-solution-layer.md
