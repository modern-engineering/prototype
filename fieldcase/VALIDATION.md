# VALIDATION - managed IP-network security field case

## Purpose

This ledger records what the field case asks the solution model to express and what the executable evidence verifies.
The scenario is fictional and uses conventional managed-IP-network security concepts.

## Scenario

A network controller reports assets, sessions, links, gateways, and traffic observations from a target network.
An observation translator converts those reports into canonical events.
A digital-twin pipeline applies the events to a graph of the target network and publishes change digests.
An identity-anomaly detector correlates stable asset identities with their observed endpoints and alerts when a known identity appears from an unexpected endpoint.
The alert stream fans out to a PostgreSQL journal and an optional customer-owned SIEM endpoint.

The scenario deliberately crosses several responsibility boundaries.
The logical solution owns component selection, configuration, and channel bindings.
The adopter owns Kafka, Redis, Neo4j, PostgreSQL, the network-controller connection, the SIEM connection, credentials, and target execution.

## Required model capabilities

| Capability | Representation | Verification |
| --- | --- | --- |
| Stable component identity | Named catalogue descriptors and solution instances | Compiled-image assertions and host logs |
| Explicit component configuration | Typed or token-valued parameters on each instance | Parser, linker, and generated-host tests |
| Required credentials | Tainted site externs and provision outputs | Unbound-input failures and redaction tests |
| Explicit event topology | Shared channel variables bound to producer and consumer ports | Compiled-image assertions |
| Site-owned service access | Extern handles and slice provisions | Site loading and provision-driver tests |
| Reproducible runtime configuration | Public and sensitive site fragments generated from the compiled image | Regeneration and golden-file tests |
| Runtime readiness | Generated host health and readiness endpoints | Host smoke tests |
| Ordered shutdown | Context cancellation propagated through hosted instances | Host lifecycle tests |
| Kubernetes handoff | ConfigMap, Secret, and Deployment descriptors | YAML parse and descriptor-shape tests |
| Solution attribution | Solution and instance metadata on generated resources | Manifest assertions |

## Scenario invariants

- A channel has one declared meaning wherever it crosses a component boundary.
- Sensitive values never appear in ordinary configuration, generated logs, or public manifest fragments.
- The digital-twin translator keeps a stable runtime identity so its Kafka consumer group can resume from committed offsets.
- Controller-token state and detector-correlation state use separate Redis provisions.
- The graph writer receives Neo4j credentials through tainted provision outputs.
- The detector, journal, and SIEM exporter consume the same alert stream independently.
- An adopter may replace a bundled driver or generator only if the replacement preserves the handoff semantics exercised here.

## Evidence stages

### Stage 1 - compile the logical solution

The SDL units compose the substrate, observation, digital-twin, and detection components into one logical solution.
The compiler must reject missing required inputs, incompatible parameter values, and malformed extension blocks.
The committed compiled image is regenerated in tests and must match byte for byte.

### Stage 2 - run a generated host

The generator derives a runnable host from the compiled image and a site file.
The host resolves externs, invokes provision drivers, redacts tainted values, reports readiness, and stops its instances in dependency order.
Smoke tests select individual instances as well as the complete solution.

### Stage 3 - generate Kubernetes descriptors

The Kubernetes generator derives one descriptor set per deployed instance.
Ordinary settings are written to a ConfigMap, sensitive settings to a Secret, and the Deployment receives only file paths and non-sensitive control variables.
Tests parse every YAML document, verify probes and resource settings, and compare committed output with regeneration.

## Deliberate limits

The field case does not claim to model every deployment concern.
The following remain outside the current representation:

- message encoding and schema compatibility for each event channel;
- optional site-bound secrets without introducing a separate optional-binding construct;
- secret-file contents and volume composition beyond the path handed to the application;
- helper containers and other pod-level composition;
- broker, database, network, credential, and failure isolation supplied by the adopter's platforms;
- reconciliation policy after descriptors are handed to a target-specific deployment system;
- equivalent operational behavior across materially different target environments.

These limits are findings, not invitations to hide target-specific behavior in the portable solution declaration.

## Acceptance

The field case is internally consistent when all Go tests pass, generated artifacts reproduce exactly, sensitive values remain redacted, and each declared handoff has an observable verification point.
That evidence validates this bounded scenario only.
It does not establish customer value, cross-adopter reuse, or production readiness.
