// Package httpclient provides Acme's HTTP conventions.
// Your shared integration packages can expose bindable configuration while
// leaving client construction and lifetime to the consuming executable.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program"
)

// Endpoint configures authenticated GET requests. BindInputs exposes its URL
// and token without opening a connection. A client must be supplied before Get.
// The client remains owned by its supplier; Endpoint never closes or changes it.
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`

	client *http.Client
}

// BindInputs registers the endpoint's configuration without changing it.
// If your program uses several endpoints, bind their fields individually under
// distinct names or choose a collection that supplies your naming convention.
func (e *Endpoint) BindInputs(inputs *input.List) {
	// Your existing JSON names work for binding too. Keep help and sensitivity
	// here so they describe the program's declarations, not its JSON encoding.
	inputs.JSON(e,
		program.Field{Name: "url", Help: "service URL"},
		program.Field{Name: "token", Help: "bearer credential", Sensitive: true},
	)
}

// UseHTTPClient retains client for subsequent requests without performing I/O.
// Passing nil clears the client; Get then returns an error.
func (e *Endpoint) UseHTTPClient(client *http.Client) {
	e.client = client
}

// Get fetches the configured URL and returns its response body. A nonempty token
// is sent as a bearer credential. Get rejects non-2xx responses and closes each
// response body before returning, including on a read or status error. A body
// closing error is joined with any request error.
func (e *Endpoint) Get(ctx context.Context) (body []byte, err error) {
	if e.client == nil {
		return nil, errors.New("HTTP client has not been supplied")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, e.URL, nil)
	if err != nil {
		return nil, err
	}
	if e.Token != "" {
		request.Header.Set("Authorization", "Bearer "+e.Token)
	}
	response, err := e.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed: %s", response.Status)
	}
	return io.ReadAll(response.Body)
}
