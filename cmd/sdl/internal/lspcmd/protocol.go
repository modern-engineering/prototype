// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package lspcmd

// This file transcribes the slice of the Language Server Protocol
// (3.17) this server actually speaks, and nothing more: field names
// and casing follow the specification, optional fields the server
// neither reads nor writes are simply absent, and enumerations narrow
// to the constants used. Growing the surface means growing these
// types alongside their handlers.

// syncFull is the TextDocumentSyncKind the server advertises: every
// didChange carries the document's whole text. Incremental sync is a
// recorded door; the checks re-parse from scratch anyway, so full
// sync costs one string per keystroke and no correctness.
const syncFull = 1

// severityError is the one DiagnosticSeverity published: every
// finding is a construct the build (or the parse) rejects, never a
// stylistic warning.
const severityError = 1

// The CompletionItemKinds the item list uses.
const (
	kindField   = 5  // a top-level field of a statement verb
	kindKeyword = 14 // a grammar keyword or section word
)

// An initializeResult answers initialize with the served capabilities.
type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   serverInfo         `json:"serverInfo"`
}

// serverCapabilities declares what the client may ask of the server.
type serverCapabilities struct {
	TextDocumentSync   int      `json:"textDocumentSync"`
	CompletionProvider struct{} `json:"completionProvider"`
}

// serverInfo names the server to the client's logs.
type serverInfo struct {
	Name string `json:"name"`
}

// A textDocumentItem is didOpen's document: the URI names it, the
// text is its whole content. The languageId and version ride along in
// the protocol but nothing here reads them.
type textDocumentItem struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

// A textDocumentIdentifier names a document already open.
type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

// didOpenParams carries textDocument/didOpen.
type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

// didChangeParams carries textDocument/didChange.
type didChangeParams struct {
	TextDocument   textDocumentIdentifier `json:"textDocument"`
	ContentChanges []contentChange        `json:"contentChanges"`
}

// A contentChange is one change event. Under the advertised full sync
// Range is absent and Text is the whole document; a range-scoped
// event is out of contract and its carrier skips it rather than
// misapply a fragment.
type contentChange struct {
	Range *span  `json:"range"`
	Text  string `json:"text"`
}

// didCloseParams carries textDocument/didClose.
type didCloseParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

// publishDiagnosticsParams carries textDocument/publishDiagnostics.
// Diagnostics is never nil: the empty array is how a publish clears a
// document's earlier findings, and JSON null is not an empty array.
type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

// A diagnostic is one published finding.
type diagnostic struct {
	Range    span   `json:"range"`
	Severity int    `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

// A span is the protocol's Range: half-open, positions zero-based.
type span struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

// A position is a zero-based line and character. The protocol's
// default encoding counts characters in UTF-16 code units where the
// scanner counts bytes; the two agree over ASCII, and negotiating an
// encoding (or transcoding columns) is a recorded door.
type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// A completionItem is one completion candidate; Detail carries the
// vocabulary group it came from.
type completionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail"`
}
