// Package sef describes the information carried by a SEF document independently
// of its encoding. JSON documents use the suffix .json.sef; gob documents use
// .gob.sef. DecodeJSON and EncodeJSON read and write the JSON encoding.
//
// A document carries solution identity, revision, and structural intent. Its
// definition uses names to refer to Go values; runtime registrations, linked
// objects, invocation state, and resource ownership are outside the document.
//
// This information model and its wire schema are experimental. Codecs accept
// exactly Version and reject other identifiers; no compatibility across
// experimental versions is promised. Supporting attachments and placement-specific
// information remain requirements whose representation is still open.
package sef

import "github.com/modern-engineering/prototype/solution"

// Document is the information shared by the encodings of a SEF file.
// Encodings preserve content and authored order, not Go pointer or slice identity.
type Document struct {
	// Version identifies the SEF information schema. It is independent of the
	// encoding, the solution revision, and the versions of referenced modules.
	// Codecs currently support only Version.
	Version string

	// ID identifies the solution across revisions. Its syntax and scope are
	// left to the producer and consumer; no UUID or URI scheme is implied.
	ID string

	// Revision identifies a revision of ID. It is opaque: no ordering, semantic
	// version, or content-hash interpretation is imposed here.
	Revision string

	// Definition carries module requirements, named catalogue occurrences,
	// and authored logical bindings. Linking it produces a separate view.
	Definition solution.Definition
}
