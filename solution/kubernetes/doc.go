// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package kubernetes renders desired-state images into static
// Kubernetes manifest sets: the manifests-at-rest half of the
// Kubernetes archetype (D-16). [Render] consumes one image and one
// site's bindings, dry — image-internal validation only, no live
// catalogue, no cluster client, never an apply — and emits one file
// per deploy record: a ConfigMap, a Secret exactly where something
// tainted flows, and a Deployment. Applying the set is the
// environment's own act, and the convergence discipline compresses
// into it — converging on the next apply.
//
// The pod is the enactment boundary. [Shard] derives each record's
// sub-image — the record, the provisions its parameters reach, the
// symbols that closure references, the full catalogue pin — and the
// basic pod contract mounts it beside the instance's extern files for
// a stock prebuilt host (examples/host's shape) to enact: the plan
// gate, the extern gate, the audit, and the exit taxonomy all run per
// pod, machinery unchanged (D-13). Nothing this package checks is
// authoritative; the pod's host re-judges everything against the
// catalogue its binary links, and exit 2 turns skew into a visible
// crash loop instead of a misreading.
//
// Rendering is a site-binding act. A [Site]'s var values rebind the
// shard's symbol table; its extern values land in mounted files split
// by taint — plain ones in the ConfigMap, sensitive ones in the
// Secret — so custody is decided by which file a value renders into,
// never by vigilance. The render gate refuses a site that leaves any
// reached extern unbound, the same MUST-bind rule every host re-runs
// wet.
//
// A [Profile] is the seam platform teams own: object dressing,
// payload files, and the container contract may all be re-skinned —
// down to the one-shared-binary, selection-at-runtime shape a studied
// production estate runs — while naming, the checksum rollout
// annotation, the custody split, and the stanza reading stay the
// renderer's. The k8s.pod and k8s.workload stanzas are honored
// advisorily (D-15): silence means defaults, never an error.
//
// Doors, recorded here where they would reopen: pruning what left the
// image (a render deletes nothing; the applying pipeline owns memory
// of the previous set); Services, once the image gains a listener
// concept; probes in the default contract, once the host serves a
// health surface; a secret-reference custody mode that keeps values
// out of the rendered set; and co-location profiles grouping several
// records into one pod.
package kubernetes
