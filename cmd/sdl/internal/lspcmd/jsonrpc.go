// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package lspcmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// This file is the transport: JSON-RPC 2.0 messages under the LSP base
// protocol's HTTP-style framing (a Content-Length header block, then
// exactly that many body bytes). Hand-rolled deliberately — the base
// protocol is a page of framing rules, x/tools keeps its own
// implementation internal, and a dependency would outweigh the code.

// maxMessageBytes caps one message body. A didOpen carries a whole
// unit's text, so the cap is generous next to any plausible solution
// source; its job is refusing a hostile or corrupt Content-Length
// before the allocation, not budgeting honest traffic.
const maxMessageBytes = 8 << 20

// JSON-RPC 2.0 error codes the server answers faults with.
const (
	codeParseError     = -32700 // body is not parseable JSON
	codeInvalidRequest = -32600 // request refused, e.g. after shutdown
	codeMethodNotFound = -32601 // method outside the served surface
)

// A message is one incoming JSON-RPC 2.0 message, request and
// notification alike: an id marks a request awaiting a response, its
// absence a notification. The jsonrpc version tag is not enforced —
// the server answers any client that talks to it. The raw id travels
// back verbatim in the response, so its JSON type (number or string)
// never matters here; a null id reads as absent, which demotes the
// discouraged null-id request to a notification.
type message struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
}

// A response answers one request. Result is always serialized, null
// included: a response without a result field is not an answer under
// JSON-RPC, and shutdown's answer is precisely a null result.
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result"`
}

// An errorResponse refuses one request.
type errorResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   rpcError        `json:"error"`
}

// An rpcError is the refusal's payload.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// A notification is a server-initiated message expecting no answer;
// published diagnostics ride it.
type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// readMessage reads one framed message body. Header names compare
// case-insensitively and unknown headers (Content-Type) are skipped;
// only Content-Length matters and it must be present. A stream ending
// cleanly between messages returns io.EOF; one ending inside a message
// returns io.ErrUnexpectedEOF, so the caller can tell a hangup from a
// truncation.
func readMessage(r *bufio.Reader) ([]byte, error) {
	length := -1
	for first := true; ; first = false {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if first && line == "" {
					return nil, io.EOF
				}
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // the blank line ends the header block
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("malformed header line %q", line)
		}
		if !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < 0 {
			return nil, fmt.Errorf("malformed Content-Length %q", strings.TrimSpace(value))
		}
		if n > maxMessageBytes {
			return nil, fmt.Errorf("message of %d bytes exceeds the %d-byte cap", n, maxMessageBytes)
		}
		length = n
	}
	if length < 0 {
		return nil, errors.New("message headers carry no Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return body, nil
}

// writeMessage frames and writes one outgoing message.
func writeMessage(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}
