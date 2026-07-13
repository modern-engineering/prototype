// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package lspcmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/vetcmd"
	"github.com/modern-engineering/prototype/sdl/scanner"
	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution"
)

// A server carries one editor session. It is deliberately
// single-threaded and stateless beyond the shutdown latch: messages
// are handled to completion in arrival order — the checks are a
// single-file parse, so concurrency would buy nothing and cost the
// publish ordering — and no document store exists because every
// handler works from the text its own message carries.
type server struct {
	in  *bufio.Reader
	out io.Writer

	// shutdown latches after the shutdown request: the session is
	// ending, later requests are refused, and only exit (or the
	// client hanging up) is expected.
	shutdown bool

	// items is the context-free completion list, generated once from
	// the grammar code; the vocabulary cannot change mid-session.
	items []completionItem
}

// serve speaks one whole session over the given streams and returns
// nil exactly when the client ended it cleanly: a shutdown request
// followed by the exit notification, or by closing the stream — many
// clients just drop the pipe once shutdown is answered.
func serve(in io.Reader, out io.Writer) error {
	s := &server{in: bufio.NewReader(in), out: out, items: completionItems()}
	for {
		body, err := readMessage(s.in)
		if errors.Is(err, io.EOF) {
			if s.shutdown {
				return nil
			}
			return errors.New("the client closed the stream without a shutdown request")
		}
		if err != nil {
			return err
		}
		var msg message
		if err := json.Unmarshal(body, &msg); err != nil {
			// The framing was sound, so the stream is still in sync;
			// refuse the body and carry on. A parse fault has no id to
			// echo, which JSON-RPC spells as the null id.
			if err := s.reject(json.RawMessage("null"), codeParseError, "message body is not valid JSON"); err != nil {
				return err
			}
			continue
		}
		switch {
		case msg.Method == "exit":
			// The session's last word, honored even from a confused
			// client that tags it with an id: nothing follows exit.
			if !s.shutdown {
				return errors.New("the client sent exit without a shutdown request")
			}
			return nil
		case msg.Method == "":
			// A response: this server sends no requests, so a stray
			// one has nothing to land on and drops.
		case msg.ID != nil:
			err = s.request(&msg)
		default:
			err = s.notify(&msg)
		}
		if err != nil {
			// Handlers only fail to write; a broken output stream ends
			// the session.
			return err
		}
	}
}

// request answers one request. After shutdown every request is
// refused — the client asked for the end and gets only exit — and an
// unserved method is named back rather than silently swallowed, so a
// client's capability probing sees an honest surface.
func (s *server) request(msg *message) error {
	if s.shutdown {
		return s.reject(*msg.ID, codeInvalidRequest, "the server is shutting down")
	}
	switch msg.Method {
	case "initialize":
		return s.reply(*msg.ID, initializeResult{
			Capabilities: serverCapabilities{TextDocumentSync: syncFull},
			ServerInfo:   serverInfo{Name: "sdl"},
		})
	case "shutdown":
		s.shutdown = true
		return s.reply(*msg.ID, nil)
	case "textDocument/completion":
		// Context-free by design: the same vocabulary at every
		// position, so the request's position never gets decoded.
		return s.reply(*msg.ID, s.items)
	default:
		return s.reject(*msg.ID, codeMethodNotFound, fmt.Sprintf("method %s is not served", msg.Method))
	}
}

// notify handles one notification. A notification has no response
// channel, so a malformed one drops silently — the protocol offers
// nowhere to report it that an editor would surface.
func (s *server) notify(msg *message) error {
	if s.shutdown {
		return nil
	}
	switch msg.Method {
	case "initialized":
		// The handshake's acknowledgement; nothing to configure.
	case "textDocument/didOpen":
		var p didOpenParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		return s.check(p.TextDocument.URI, p.TextDocument.Text)
	case "textDocument/didChange":
		var p didChangeParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		// Under full sync every event carries the whole document and
		// the last one is current; a range-scoped event would be a
		// fragment, misapplied as a whole, so it never counts.
		text, whole := "", false
		for _, change := range p.ContentChanges {
			if change.Range == nil {
				text, whole = change.Text, true
			}
		}
		if !whole {
			return nil
		}
		return s.check(p.TextDocument.URI, text)
	case "textDocument/didClose":
		var p didCloseParams
		if err := json.Unmarshal(msg.Params, &p); err != nil {
			return nil
		}
		// The document leaves the session; clear its findings rather
		// than leave stale squiggles behind.
		return s.publish(p.TextDocument.URI, []diagnostic{})
	default:
		// Unserved notifications — $/ traffic, willSave, and whatever
		// else a client volunteers — drop by the protocol's own rule.
	}
	return nil
}

// check runs the single-file checks over one document version and
// publishes the verdict. A clean unit still publishes: the empty
// array is what clears the previous version's findings.
func (s *server) check(uri, text string) error {
	findings := vetcmd.CheckSource(uriPath(uri), []byte(text))
	diags := make([]diagnostic, 0, len(findings))
	for _, f := range findings {
		diags = append(diags, toDiagnostic(f))
	}
	return s.publish(uri, diags)
}

// publish sends one textDocument/publishDiagnostics notification.
func (s *server) publish(uri string, diags []diagnostic) error {
	return writeMessage(s.out, notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  publishDiagnosticsParams{URI: uri, Diagnostics: diags},
	})
}

// reply answers a request; a nil result serializes as the null result
// shutdown's answer is.
func (s *server) reply(id json.RawMessage, result any) error {
	return writeMessage(s.out, response{JSONRPC: "2.0", ID: id, Result: result})
}

// reject refuses a request with a coded error.
func (s *server) reject(id json.RawMessage, code int, message string) error {
	return writeMessage(s.out, errorResponse{JSONRPC: "2.0", ID: id, Error: rpcError{Code: code, Message: message}})
}

// toDiagnostic renders one finding at its anchor. The scanner's
// positions are one-based where the protocol's are zero-based; the
// range is empty because a finding names a point, not an extent; and
// a finding that carries no position at all — a parse failure
// reported without one — anchors at the top of the document.
func toDiagnostic(e *scanner.Error) diagnostic {
	var p position
	if e.Pos.IsValid() {
		p = position{Line: e.Pos.Line - 1, Character: max(e.Pos.Column-1, 0)}
	}
	return diagnostic{
		Range:    span{Start: p, End: p},
		Severity: severityError,
		Source:   "sdl",
		Message:  e.Msg,
	}
}

// uriPath renders a document URI as the filename findings cite. The
// name serves message readability only — positions travel in the
// diagnostic's range, and in-message anchors ("first declared at")
// spell the name out — so anything but a parseable file: URI degrades
// to the URI text itself, never to an error.
func uriPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return uri
	}
	return u.Path
}

// completionItems generates the context-free completion list from the
// grammar code: sdl/token's keywords, then the section words and the
// statement verbs' top-level fields of the vocabulary the linker
// enforces — the exact sources the highlighter renders, so an
// editor's completions cannot drift from the compiler either. Each
// group is sorted and its items name their group in Detail.
func completionItems() []completionItem {
	var items []completionItem
	for _, kw := range slices.Sorted(token.Keywords()) {
		items = append(items, completionItem{Label: kw, Kind: kindKeyword, Detail: "keyword"})
	}
	vocab := solution.Vocabulary()
	for _, sec := range slices.Sorted(slices.Values(vocab.Sections)) {
		items = append(items, completionItem{Label: sec, Kind: kindKeyword, Detail: "section"})
	}
	verbsByField := make(map[string][]string)
	for verb, fields := range vocab.RootFields {
		for _, field := range fields {
			verbsByField[field] = append(verbsByField[field], verb)
		}
	}
	for _, field := range slices.Sorted(maps.Keys(verbsByField)) {
		verbs := slices.Sorted(slices.Values(verbsByField[field]))
		items = append(items, completionItem{Label: field, Kind: kindField, Detail: strings.Join(verbs, ", ") + " top-level field"})
	}
	return items
}
