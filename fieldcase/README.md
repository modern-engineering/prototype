# fieldcase - managed IP-network security validation

This directory is a CI-guarded field case for an asset-identity anomaly detection system in a managed IP network.
The scenario is fictional and independent of any particular organisation or product.
It exercises configuration, event wiring, stateful correlation, lifecycle handling, and deployment generation.

## Scenario

A network controller reports assets, sessions, links, gateways, and traffic observations through an HTTPS API.
The network-observation collector publishes those reports to Kafka.
The observation translator converts them into canonical network events.
Two digital-twin stages apply the events to a Neo4j graph of the target network and publish change digests.
The identity-anomaly detector uses Redis state to identify a known asset identity appearing from an unexpected endpoint.
Its alerts fan out to a PostgreSQL journal and an optional customer-owned SIEM over syslog.

```text
network controller
    -> network-observation-collector
    -> observation-translator
    -> digital-twin translator
    -> target-network digital twin
    -> identity-anomaly detector
    +-> PostgreSQL alert journal
    +-> customer SIEM
```

The digital-twin translator retains the stable runtime name `digitaltwin` so its Kafka consumer group can resume from committed offsets after redeployment.
A site may also interpose an observation filter by rebinding channels without changing the collector or translator.

## Why this field case is useful

The flow combines HTTPS polling, Kafka fan-out and fan-in, Redis state, a Neo4j-backed digital twin, PostgreSQL storage, and syslog egress.
Its configuration includes durations, lists, booleans, required credentials, optional integrations, cross-field validation, and tainted provision outputs.
That combination makes gaps in the intermediate representation and runtime handoffs observable without relying on domain-specific algorithms.

## Layout

- `catalog/` contains the controller collector, observation translator, digital-twin stages, security detectors, and substrate types.
- `solutions/anomaly/` composes the flow and contains its site files, compiled image, generated host, Kubernetes descriptors, and proof tests.
- `host/` contains configuration resolution, provision-driver integration, health aggregation, and ordered shutdown.
- `cmd/fieldcase-gen` generates a host or Kubernetes descriptors from a compiled image and site file.
- [VALIDATION.md] records the required model capabilities, executable evidence, and deliberate limits.

The component bodies are representative loops.
The validation target is the solution contract, wiring, generation, and lifecycle behavior rather than the detector's production algorithm.

## Verification

Run `go test ./...` from this directory.
The tests compile the logical solution, reproduce committed artifacts, exercise the generated host, verify sensitive-value redaction, and parse every generated Kubernetes descriptor.

[VALIDATION.md]: VALIDATION.md
