package solution

// Artifact carries solution identity, revision, schema version, and structural
// intent independently of its encoding. It contains no runtime registrations,
// linked objects, invocation state, or resource ownership.
//
// Encodings preserve content and authored order, not Go pointer or slice identity.
// Attachments and placement-specific information have no representation yet.
type Artifact struct {
	// Version identifies the information schema. It is independent of the
	// encoding, the solution revision, and referenced module versions.
	Version string

	// ID identifies the solution across revisions. Its syntax and scope
	// belong to the producer and consumer; no UUID or URI scheme is implied.
	ID string

	// Revision identifies a revision of ID. It is opaque: no ordering,
	// semantic-version, or content-hash interpretation is imposed here.
	Revision string

	// Definition carries module requirements, named catalogue occurrences,
	// and authored logical bindings. Linking produces a separate graph.
	Definition Definition
}
