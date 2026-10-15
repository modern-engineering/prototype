package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	if err := run(t.Context(), server.Client(), server.URL, "example-token", &out); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "ready\n" {
		t.Errorf("output = %q; want ready followed by newline", got)
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
	if err := run(t.Context(), nil, server.URL, "example-token", &out); err == nil {
		t.Fatal("Fetch ran without a supplied client")
	}
	if out.Len() != 0 {
		t.Errorf("failed Fetch wrote %q", out.String())
	}
}
