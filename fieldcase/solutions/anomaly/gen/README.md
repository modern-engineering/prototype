# Generated deployment artifacts

Everything under `host/` and `k8s/` is generated from the committed image and the development site.
Nothing in these directories is hand-edited.
`TestGenOutputsCurrent` regenerates both sets and compares them byte for byte.

From the field-case module root:

```sh
go run ./cmd/fieldcase-gen -image solutions/anomaly/anomaly.json \
    -site solutions/anomaly/site/dev.site -mode host -o solutions/anomaly/gen/host
go run ./cmd/fieldcase-gen -image solutions/anomaly/anomaly.json \
    -site solutions/anomaly/site/dev.site -mode k8s -o solutions/anomaly/gen/k8s
```

## Generated host

The generated `package main` embeds the compiled solution records and imports the catalogue packages needed to link their descriptors.
Site bindings remain external so one binary can run the same logical solution at different sites without embedding credentials.
The host loads one or more site files, resolves provision outputs, applies instance configuration, reports readiness, and coordinates shutdown.

Build and run it with:

```sh
go build -o anomaly-host ./solutions/anomaly/gen/host
./anomaly-host solutions/anomaly/site/dev.site
```

`solutions/anomaly/hostsmoke_test.go` verifies the required-extern gate, effective-value audit with redaction, selective instance hosting, health reporting, and clean termination.

## Kubernetes descriptors

The generator emits one YAML file per deployed instance through the shared Kubernetes renderer.
Each file contains a ConfigMap, an optional Secret when the instance reaches tainted values, and a Deployment.
The Deployment selects one instance from the shared host image, mounts its site fragments, exposes health probes, and applies resources from the compile-checked `k8s.pod` stanza.

Provision outputs are resolved at pod startup through the same driver interface used by the single-process host.
Adopter drivers must therefore be idempotent when several pods depend on the same provision.

The `${ANOMALY_HOST_IMAGE}` token is substituted by the deployment pipeline.
Image pull policy, replicas, autoscaling, telemetry endpoints, environment labels, placement, Services, and target reconciliation remain adopter-owned platform decisions.
The checksum annotation covers ordinary configuration; a production profile must also define its credential-rotation behavior.
