// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package lspcmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/modern-engineering/prototype/sdl/token"
	"github.com/modern-engineering/prototype/solution"
)

// An interrupt — the invocation context ending — unblocks a read
// already parked on the session's input and turns the verdict into
// the session-interrupted error: the lever behind ^C on an sdl lsp
// whose editor went quiet, wired through main's signal context. The
// bubble proves both halves — the read holds while the context lives,
// and only the cancellation moves it.
func TestInterruptUnblocksPendingRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		in, clientEnd := io.Pipe() // a client that never says anything
		verdict := make(chan error, 1)
		go func() { verdict <- run(ctx, in, io.Discard) }()

		synctest.Wait() // let the session park on its pending read
		select {
		case err := <-verdict:
			t.Fatalf("run returned %v before the interrupt", err)
		default:
		}

		cancel()
		if err := <-verdict; err == nil || err.Error() != "session interrupted" {
			t.Errorf("run() = %v, want the session-interrupted verdict", err)
		}
		// Release the pump goroutine the interrupt stranded on its
		// read: production exits the process here; the bubble insists
		// every goroutine gets to leave.
		clientEnd.Close()
	})
}

// req builds one client request.
func req(id int, method string, params any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
}

// note builds one client notification.
func note(method string, params any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
}

// A raw script entry frames its bytes as one message body, malformed
// JSON and all.
type raw string

// A verbatim script entry writes stream bytes unframed, for breaking
// the framing itself.
type verbatim string

// A reply is one server message, kept raw beside its decoding: some
// contracts live in the exact JSON (a null result, an empty array)
// that a decode would blur.
type reply struct {
	raw string
	msg map[string]any
}

// session drives one whole scripted session through serve in process —
// the client transcript in, the server transcript out — and returns
// the server's messages in send order together with serve's verdict.
func session(t *testing.T, script ...any) ([]reply, error) {
	t.Helper()
	var in, out bytes.Buffer
	for _, v := range script {
		switch b := v.(type) {
		case raw:
			fmt.Fprintf(&in, "Content-Length: %d\r\n\r\n%s", len(b), b)
		case verbatim:
			in.WriteString(string(b))
		default:
			if err := writeMessage(&in, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	err := serve(&in, &out)
	r := bufio.NewReader(&out)
	var replies []reply
	for {
		body, rerr := readMessage(r)
		if errors.Is(rerr, io.EOF) {
			return replies, err
		}
		if rerr != nil {
			t.Fatalf("server output framing: %v", rerr)
		}
		var m map[string]any
		if uerr := json.Unmarshal(body, &m); uerr != nil {
			t.Fatalf("server output is not JSON: %v\n%s", uerr, body)
		}
		replies = append(replies, reply{raw: string(body), msg: m})
	}
}

// dig walks nested JSON objects by key.
func dig(t *testing.T, v any, path ...string) any {
	t.Helper()
	for _, key := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("dig %v: value at %q is not an object", path, key)
		}
		v, ok = m[key]
		if !ok {
			t.Fatalf("dig %v: no key %q", path, key)
		}
	}
	return v
}

// diagnostics asserts one reply is a publishDiagnostics notification
// for the given document and returns its diagnostics array.
func diagnostics(t *testing.T, r reply, wantURI string) []any {
	t.Helper()
	if got := dig(t, r.msg, "method"); got != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %v, want textDocument/publishDiagnostics", got)
	}
	if got := dig(t, r.msg, "params", "uri"); got != wantURI {
		t.Fatalf("published uri = %v, want %s", got, wantURI)
	}
	diags, ok := dig(t, r.msg, "params", "diagnostics").([]any)
	if !ok {
		t.Fatalf("diagnostics are not an array in %s", r.raw)
	}
	return diags
}

// TestSessionLifecycle drives the whole v0 surface in one scripted
// session: the initialize handshake, a didOpen of a unit sdl build
// would reject publishing exactly vet's single-file findings at
// zero-based positions, a context-free completion exchange, a
// didChange to a clean revision clearing the findings with an empty
// array (never null), a didClose clearing again, and the clean
// shutdown/exit end returning nil.
func TestSessionLifecycle(t *testing.T) {
	const uri = "file:///play/unit.sdl"
	const broken = `solution demo

deploy p.T as a {
	bogus {
	}
	params {
	}
	params {
	}
}
`
	const clean = `solution demo

deploy p.T as a {
	params {
	}
}
`
	replies, err := session(t,
		req(1, "initialize", map[string]any{}),
		note("initialized", map[string]any{}),
		note("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "languageId": "sdl", "version": 1, "text": broken},
		}),
		req(2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		}),
		note("textDocument/didChange", map[string]any{
			"textDocument":   map[string]any{"uri": uri, "version": 2},
			"contentChanges": []any{map[string]any{"text": clean}},
		}),
		note("textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}),
		req(3, "shutdown", nil),
		note("exit", nil),
	)
	if err != nil {
		t.Fatalf("serve returned %v, want nil", err)
	}
	if len(replies) != 6 {
		t.Fatalf("got %d server messages, want 6", len(replies))
	}

	// initialize: full sync, a completion provider, the server's name.
	if got := dig(t, replies[0].msg, "id"); got != float64(1) {
		t.Errorf("initialize response id = %v, want 1", got)
	}
	if got := dig(t, replies[0].msg, "result", "capabilities", "textDocumentSync"); got != float64(syncFull) {
		t.Errorf("textDocumentSync = %v, want %d", got, syncFull)
	}
	dig(t, replies[0].msg, "result", "capabilities", "completionProvider")
	if got := dig(t, replies[0].msg, "result", "serverInfo", "name"); got != "sdl" {
		t.Errorf("serverInfo.name = %v, want sdl", got)
	}

	// didOpen: the two findings, sorted, at zero-based anchors, with
	// the in-message anchor citing the URI's decoded path.
	wants := []struct {
		line, character float64
		message         string
	}{
		{3, 1, "unknown section bogus: sections are metadata, params, and with"},
		{7, 1, "duplicate params section (first declared at /play/unit.sdl:6:2)"},
	}
	diags := diagnostics(t, replies[1], uri)
	if len(diags) != len(wants) {
		t.Fatalf("didOpen published %d diagnostics, want %d:\n%s", len(diags), len(wants), replies[1].raw)
	}
	for i, want := range wants {
		if got := dig(t, diags[i], "message"); got != want.message {
			t.Errorf("diagnostic %d message = %v, want %q", i, got, want.message)
		}
		if got := dig(t, diags[i], "range", "start", "line"); got != want.line {
			t.Errorf("diagnostic %d start line = %v, want %v", i, got, want.line)
		}
		if got := dig(t, diags[i], "range", "start", "character"); got != want.character {
			t.Errorf("diagnostic %d start character = %v, want %v", i, got, want.character)
		}
		if got := dig(t, diags[i], "severity"); got != float64(severityError) {
			t.Errorf("diagnostic %d severity = %v, want %d", i, got, severityError)
		}
		if got := dig(t, diags[i], "source"); got != "sdl" {
			t.Errorf("diagnostic %d source = %v, want sdl", i, got)
		}
	}

	// completion: every vocabulary group answers, with its kind and
	// group detail.
	items, ok := dig(t, replies[2].msg, "result").([]any)
	if !ok {
		t.Fatalf("completion result is not an array in %s", replies[2].raw)
	}
	byLabel := make(map[string]map[string]any, len(items))
	for _, it := range items {
		m := it.(map[string]any)
		byLabel[m["label"].(string)] = m
	}
	for _, want := range []struct {
		label  string
		kind   float64
		detail string
	}{
		{"deploy", kindKeyword, "keyword"},
		{"params", kindKeyword, "section"},
		{"with", kindKeyword, "section"},
		{"location", kindField, "deploy top-level field"},
	} {
		it, found := byLabel[want.label]
		if !found {
			t.Errorf("completion misses %q", want.label)
			continue
		}
		if it["kind"] != want.kind || it["detail"] != want.detail {
			t.Errorf("completion %q = (%v, %v), want (%v, %q)", want.label, it["kind"], it["detail"], want.kind, want.detail)
		}
	}

	// didChange to clean text and didClose both clear: an empty JSON
	// array, never null.
	for _, r := range []reply{replies[3], replies[4]} {
		if diags := diagnostics(t, r, uri); len(diags) != 0 {
			t.Errorf("expected a clearing publish, got %s", r.raw)
		}
		if !strings.Contains(r.raw, `"diagnostics":[]`) {
			t.Errorf("clearing publish must carry an empty array, got %s", r.raw)
		}
	}

	// shutdown: a null result is still a result.
	if got := dig(t, replies[5].msg, "id"); got != float64(3) {
		t.Errorf("shutdown response id = %v, want 3", got)
	}
	if !strings.Contains(replies[5].raw, `"result":null`) {
		t.Errorf("shutdown response must carry a null result, got %s", replies[5].raw)
	}
}

// TestSyntaxDiagnostics pins the broken-parse route: a unit that does
// not parse publishes the parser's own positioned errors. The script
// also skips initialize entirely — the server answers any client that
// talks to it, the recorded permissive default.
func TestSyntaxDiagnostics(t *testing.T) {
	const uri = "file:///play/broken.sdl"
	replies, err := session(t,
		note("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "languageId": "sdl", "version": 1, "text": "solution demo\n\ndeploy p.T as a {\n"},
		}),
		req(1, "shutdown", nil),
		note("exit", nil),
	)
	if err != nil {
		t.Fatalf("serve returned %v, want nil", err)
	}
	if len(replies) != 2 {
		t.Fatalf("got %d server messages, want 2", len(replies))
	}
	diags := diagnostics(t, replies[0], uri)
	if len(diags) == 0 {
		t.Fatalf("a unit that does not parse published no diagnostics")
	}
	for i := range diags {
		if got := dig(t, diags[i], "severity"); got != float64(severityError) {
			t.Errorf("diagnostic %d severity = %v, want %d", i, got, severityError)
		}
		if msg := dig(t, diags[i], "message"); msg == "" {
			t.Errorf("diagnostic %d carries no message", i)
		}
	}
}

// TestSessionEndVerdicts pins serve's exit contract: only a session
// the client ends through the shutdown handshake — exit or a hangup
// after shutdown — comes back clean.
func TestSessionEndVerdicts(t *testing.T) {
	tests := []struct {
		name    string
		script  []any
		wantErr bool
	}{
		{"shutdown then exit", []any{req(1, "shutdown", nil), note("exit", nil)}, false},
		{"shutdown then hangup", []any{req(1, "shutdown", nil)}, false},
		{"exit without shutdown", []any{note("exit", nil)}, true},
		{"hangup without shutdown", []any{req(1, "initialize", map[string]any{})}, true},
		{"empty stream", nil, true},
		{"truncated message", []any{verbatim("Content-Length: 5\r\n\r\n{}")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := session(t, tt.script...)
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("serve error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRequestFaults pins the coded refusals: an unserved method, any
// request after shutdown, and a body that is not JSON (refused under
// the null id, with the session carrying on to a clean end).
func TestRequestFaults(t *testing.T) {
	end := []any{req(9, "shutdown", nil), note("exit", nil)}
	tests := []struct {
		name     string
		script   []any
		wantCode float64
		wantID   string
	}{
		{"unserved method", append([]any{req(1, "textDocument/hover", map[string]any{})}, end...), codeMethodNotFound, `"id":1`},
		{"request after shutdown", []any{req(1, "shutdown", nil), req(2, "textDocument/completion", nil), note("exit", nil)}, codeInvalidRequest, `"id":2`},
		{"unparseable body", append([]any{raw(`{"jsonrpc":`)}, end...), codeParseError, `"id":null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replies, err := session(t, tt.script...)
			if err != nil {
				t.Fatalf("serve returned %v, want nil", err)
			}
			var refusal *reply
			for i := range replies {
				if _, refused := replies[i].msg["error"]; refused {
					refusal = &replies[i]
					break
				}
			}
			if refusal == nil {
				t.Fatalf("no error response among %d server messages", len(replies))
			}
			if got := dig(t, refusal.msg, "error", "code"); got != tt.wantCode {
				t.Errorf("error code = %v, want %v", got, tt.wantCode)
			}
			if !strings.Contains(refusal.raw, tt.wantID) {
				t.Errorf("error response %s does not echo %s", refusal.raw, tt.wantID)
			}
		})
	}
}

// TestMalformedNotificationDrops pins the drop rule: a notification
// whose params do not decode has no response channel, so it vanishes
// without derailing the session.
func TestMalformedNotificationDrops(t *testing.T) {
	replies, err := session(t,
		note("textDocument/didOpen", 5),
		req(1, "shutdown", nil),
		note("exit", nil),
	)
	if err != nil {
		t.Fatalf("serve returned %v, want nil", err)
	}
	if len(replies) != 1 {
		t.Fatalf("got %d server messages, want only the shutdown response", len(replies))
	}
}

// TestReadMessageFraming pins the transport corners: header case,
// skipped extra headers, tolerated bare-LF terminators, and the
// refusals — absent or malformed or hostile Content-Length, truncated
// streams — that keep a corrupt stream from wedging the read.
func TestReadMessageFraming(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // the body; empty means an error is wanted
	}{
		{"canonical", "Content-Length: 2\r\n\r\n{}", "{}"},
		{"case-insensitive header", "content-length: 2\r\n\r\n{}", "{}"},
		{"extra headers skipped", "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\nContent-Length: 2\r\n\r\n{}", "{}"},
		{"bare newlines tolerated", "Content-Length: 2\n\n{}", "{}"},
		{"missing content-length", "Content-Type: x\r\n\r\n{}", ""},
		{"malformed header line", "garbage\r\n\r\n", ""},
		{"malformed length", "Content-Length: two\r\n\r\n{}", ""},
		{"negative length", "Content-Length: -1\r\n\r\n", ""},
		{"hostile length", "Content-Length: 9000000000\r\n\r\n", ""},
		{"truncated body", "Content-Length: 5\r\n\r\n{}", ""},
		{"truncated headers", "Content-Length: 5\r\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := readMessage(bufio.NewReader(strings.NewReader(tt.input)))
			if tt.want == "" {
				if err == nil {
					t.Fatalf("readMessage = %q, want an error", body)
				}
				return
			}
			if err != nil {
				t.Fatalf("readMessage: %v", err)
			}
			if string(body) != tt.want {
				t.Errorf("readMessage = %q, want %q", body, tt.want)
			}
		})
	}
}

// TestCleanEndOfStream pins the io.EOF contract serve's hangup
// handling stands on: a stream ending between messages is a clean
// EOF, distinct from every truncation.
func TestCleanEndOfStream(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("Content-Length: 2\r\n\r\n{}"))
	if _, err := readMessage(r); err != nil {
		t.Fatalf("first message: %v", err)
	}
	if _, err := readMessage(r); !errors.Is(err, io.EOF) {
		t.Fatalf("end of stream = %v, want io.EOF", err)
	}
}

// TestCompletionCoversVocabulary is the anti-drift gate the tooling
// layer runs on: every keyword of the grammar and every word of the
// linker's body vocabulary must appear in the generated completion
// list, so the editor's completions cannot fall behind the compiler.
func TestCompletionCoversVocabulary(t *testing.T) {
	have := make(map[string]bool)
	for _, it := range completionItems() {
		have[it.Label] = true
	}
	var wants []string
	for kw := range token.Keywords() {
		wants = append(wants, kw)
	}
	vocab := solution.Vocabulary()
	wants = append(wants, vocab.Sections...)
	for _, fields := range vocab.RootFields {
		wants = append(wants, fields...)
	}
	if len(wants) < 10 { // more than a few words, or the vocabulary went missing
		t.Fatalf("collected only %d vocabulary words", len(wants))
	}
	for _, w := range wants {
		if !have[w] {
			t.Errorf("completion list misses vocabulary word %q", w)
		}
	}
}
