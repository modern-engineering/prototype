// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package host is the wet half of enactment: it takes a desired-state
// image and the live catalogue a binary links, and turns them into a
// running solution in that one process. The dry half — resolving and
// ordering the image against the catalogue — is [enact.Load]; this
// package consumes the plan and does everything the plan deliberately
// does not: bind values, run drivers, host services.
//
// Enactment is two phases (D-13). PROVISION runs first: each provision
// step Makes a fresh provisioner, wet-binds the record's parameters
// into the provisioner's own flags, and calls the driver, which writes
// the type's declared outputs through a [solution.OutputWriter]. Only
// then does DEPLOY run: each deploy record Makes a fresh service,
// wet-binds its parameters — provision outputs now among the values —
// and every instance runs concurrently on one [application.Runtime].
// Wet binding reuses the surface the compiler validated: the same
// flag.Value.Set that judged the literal at compile judges the resolved
// string here, so a slot cannot tell dry from wet (A-14).
//
// Values arrive through the front door. Extern symbols are supplied
// in-memory by the caller ([Config].Externs), never scraped from the
// process environment (A-12); every extern the image declares must be
// bound before anything wet runs — the gate is this package's first
// act, since the plan is deliberately site-agnostic. Every resolved
// binding and output is audited to [Config].Log with sensitive values
// redacted (A-10): the audit proves a value arrived without exposing
// it.
//
// [Run] is the core: a plain context-bound call for callers that own
// their process. It is deliberately host-agnostic — the image never
// names a host, so any binary that links a catalogue can enact
// against it; the process skins for the two host modes (a prebuilt
// platform binary; a generated per-invocation host) are thin wrappers
// a library must stay separable from.
//
// Doors, recorded here where they would reopen: a site-file extern
// source beside the in-memory map (the fieldcase host's site loader is
// the precedent); running PROVISION and DEPLOY in different binaries
// (each phase re-Loads the image against its own catalogue — plans
// never cross processes); one host process serving several solutions;
// extern values supplied but not declared are ignored today, and
// would start warning the day operator typos hurt more than the
// permissiveness helps.
package host
