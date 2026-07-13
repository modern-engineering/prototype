// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package enact plans the enactment of a desired-state image: the
// image and the live catalogue go in, a validated, ordered [Plan]
// comes out.
//
// Enactment turns an image into a running solution in two phases —
// PROVISION the backing services, then DEPLOY the applications
// (D-13). The work splits along A-14's dry/wet boundary, and this
// package is the dry half only: [Load] resolves every record against
// the catalogue the calling process links, validates that image and
// catalogue still agree, lists the extern symbols a site must bind,
// and fixes the order the wet half must follow. Load is pure — no
// I/O, no driver runs, no service starts. The wet half — gating
// externs, binding values, running drivers, hosting instances —
// belongs to the plan's consumer: the plan is data.
//
// # What the plan fixes
//
// [Plan.Provisions] carries the provision steps in dependency order:
// a step referencing another instance's outputs runs after that
// instance (Kahn's algorithm over the output-reference edges the
// linker guaranteed acyclic), and steps the edges leave free keep
// the image's record order. [Plan.Deploys] carries the deploy steps,
// all of them after every provision, in image record order:
// reconcile-time outputs feed application parameters, never the
// reverse. [Plan.Externs] lists the late-bound symbols with their
// pinned types and sensitivity; extern coverage is deliberately not
// checked here — the plan is site-agnostic, and which values a site
// supplies is the wet half's first gate.
//
// # Validation
//
// Whatever the plan can prove broken is refused at Load, before
// anything runs: a record whose element the catalogue no longer
// registers, or registers as another kind; a binding to a flag the
// element no longer declares (image/catalogue drift, held against a
// fresh instance's dry flag surface — the same surface the compiler
// validated and the wet half will parse into, A-14's drift-proofing);
// references that do not close over the image's own symbols and
// provision instances; a provision type that declares no driver; and
// slice provision records — the slice lifecycle is unbuilt, so
// attach is the one kind a plan admits. Binding values are
// deliberately not judged here: the compiler validated them dry, and
// the wet half's own flag.Value.Set validates what actually arrives.
// Faults are collected, not cut short: the returned error joins
// every fault found, one line each, so one Load reports the whole
// distance between image and catalogue.
//
// # Never assume one binary
//
// A [Step] joins a record with its resolved live element — pointers
// into the loaded catalogue — so a plan is meaningful only inside
// the process that loaded it: plans do not serialize, and nothing
// may assume the PROVISION and DEPLOY phases share a binary. A
// controller that provisions in one process and deploys in another
// loads the same image in each, against each binary's own catalogue,
// and provision outputs cross such boundaries as the rendered
// strings of [solution.OutputWriter], never as Go values. That the
// two loads see the same solution is held by the image's catalogue
// pin and by this package's validation, not by process identity.
package enact
