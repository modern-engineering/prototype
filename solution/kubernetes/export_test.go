// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes

// The stanza readers, exported to the external test package: their
// judgement calls (defaults, the both-or-neither pair, token text)
// are contracts worth pinning directly, not only through a full
// render.
var (
	PodStanza      = podStanza
	WorkloadStanza = workloadStanza
)
