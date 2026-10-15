package httpclient_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"example.com/acme/httpclient"
)

// Responses belong to each request even when its client is borrowed for a
// longer lifetime. Status and read failures must release those response bodies.
// A closing failure must remain discoverable alongside a preceding read failure.
func TestResponsesCloseWithoutClosingTheClient(t *testing.T) {
	readError := errors.New("response interrupted")
	closeError := errors.New("response close failed")
	for _, test := range []struct {
		name        string
		status      int
		reader      io.Reader
		closeErr    error
		wantErr     bool
		wantReadErr bool
	}{
		{"success", http.StatusOK, strings.NewReader("ready"), nil, false, false},
		{"status error", http.StatusUnauthorized, strings.NewReader("denied"), nil, true, false},
		{"read error", http.StatusOK, iotest.ErrReader(readError), nil, true, true},
		{"close error", http.StatusOK, strings.NewReader("ready"), closeError, true, false},
		{"read and close errors", http.StatusOK, iotest.ErrReader(readError), closeError, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := &responseBody{Reader: test.reader, closeErr: test.closeErr}
			transport := &responseTransport{status: test.status, body: body}
			endpoint := httpclient.Endpoint{URL: "https://service.example"}
			endpoint.UseHTTPClient(&http.Client{Transport: transport})
			got, err := endpoint.Get(t.Context())
			if (err != nil) != test.wantErr {
				t.Fatalf("Get error = %v; want error: %v", err, test.wantErr)
			}
			if !test.wantErr && string(got) != "ready" {
				t.Errorf("Get body = %q; want ready", got)
			}
			if errors.Is(err, readError) != test.wantReadErr {
				t.Errorf("Get error = %v; want read error: %v", err, test.wantReadErr)
			}
			if errors.Is(err, closeError) != (test.closeErr != nil) {
				t.Errorf("Get error = %v; want close error: %v", err, test.closeErr != nil)
			}
			if !body.closed {
				t.Error("response body remains open")
			}
			if transport.closed {
				t.Error("borrowed client idle connections were closed")
			}
		})
	}
}

type responseBody struct {
	io.Reader
	closed   bool
	closeErr error
}

func (b *responseBody) Close() error {
	b.closed = true
	return b.closeErr
}

type responseTransport struct {
	status int
	body   io.ReadCloser
	closed bool
}

func (t *responseTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.status, Status: http.StatusText(t.status), Body: t.body}, nil
}

func (t *responseTransport) CloseIdleConnections() { t.closed = true }
