// Package sef describes the information carried by a SEF document independently
// of its encoding. JSON and gob are intended encodings,
// identified by the file suffixes .json.sef and .gob.sef respectively.
//
// A document carries solution identity, revision, and structural intent. Its
// definition uses names to refer to Go values; runtime registrations, linked
// objects, invocation state, and resource ownership are outside the document.
//
// This information model is experimental. It does not establish a stable wire
// schema, supported version identifiers, or compatibility rules. Supporting
// attachments and placement-specific information remain requirements whose
// representation is still open.
package sef

import "github.com/modern-engineering/prototype/solution"

// Document is the information shared by the encodings of a SEF file.
// These fields describe content, not JSON keys, gob registrations, or framing.
type Document struct {
	// Version identifies the SEF information schema. It is independent of the
	// encoding, the solution revision, and the versions of referenced modules.
	// Supported identifiers and their interpretation are not yet specified.
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
