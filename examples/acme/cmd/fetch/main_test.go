package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/acme/catalogue"
)

// Only the supplied client trusts this service. Descriptor configuration and
// outputs stay separate from that client's transport policy and lifetime.
func TestConfiguredFetchUsesSuppliedClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer example-token" {
			t.Errorf("Authorization = %q; want configured credential", got)
		}
		if _, err := fmt.Fprintln(w, "ready"); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	_, untouchedInputs, untouchedOutputs := catalogue.Fetch.Make()
	token, _ := untouchedInputs.Lookup("token")
	if !token.Sensitive {
		t.Fatal("credential declaration is not sensitive")
	}
	var out bytes.Buffer
	args := []string{"-url=" + server.URL, "-token=example-token"}
	if err := run(t.Context(), server.Client(), args, &out, io.Discard); err != nil {
		t.Fatal(err)
	}
	var results map[string]string
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if results["body"] != "ready\n" {
		t.Errorf("results = %v; want response body", results)
	}
	body, _ := untouchedOutputs.Lookup("body")
	if got := body.Value.Get(); got != "" {
		t.Errorf("another runner's output = %q; want empty", got)
	}

	runner, _, _ := catalogue.Fetch.Make()
	if _, ok := runner.(httpClientUser); !ok {
		t.Fatal("runner lost the company client capability")
	}
	out.Reset()
	if err := run(t.Context(), nil, args, &out, io.Discard); err == nil {
		t.Fatal("Fetch ran without a supplied client")
	}
	if out.Len() != 0 {
		t.Errorf("failed Fetch wrote %q", out.String())
	}
}

// Help exposes the catalogue's names and defaults without executing a request;
// malformed command arguments likewise stop before the runtime client is needed.
func TestCommandConfigurationPrecedesExecution(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if err := run(t.Context(), nil, []string{"-help"}, &out, &diagnostics); err != flag.ErrHelp {
		t.Fatalf("help error = %v; want flag.ErrHelp", err)
	}
	for _, text := range []string{"-url", "http://localhost:8080", "-token"} {
		if !strings.Contains(diagnostics.String(), text) {
			t.Errorf("help lacks %q: %s", text, &diagnostics)
		}
	}
	if out.Len() != 0 {
		t.Errorf("help wrote program results: %q", &out)
	}
	diagnostics.Reset()
	if err := run(t.Context(), nil, []string{"-missing"}, &out, &diagnostics); err == nil {
		t.Fatal("unknown flag accepted")
	}
	if out.Len() != 0 || diagnostics.Len() != 0 {
		t.Fatal("failed configuration emitted results or automatic diagnostics")
	}
}
