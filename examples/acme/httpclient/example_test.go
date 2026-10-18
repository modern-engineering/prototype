package httpclient_test

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"

	"example.com/acme/httpclient"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program/command"
)

// Your shared package can expose configuration to CLI consumers while accepting
// an HTTP client chosen by each executable. The caller owns the client lifetime.
func ExampleEndpoint() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer example-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if _, err := fmt.Fprintln(w, "ready"); err != nil {
			fmt.Println(err)
		}
	}))
	defer server.Close()

	endpoint := httpclient.Endpoint{URL: server.URL}
	var inputs input.List
	endpoint.BindInputs(&inputs)
	flags := flag.NewFlagSet("request", flag.ContinueOnError)
	if err := command.Parse(flags, &inputs, []string{"-token=example-token"}); err != nil {
		fmt.Println(err)
		return
	}
	endpoint.UseHTTPClient(server.Client())
	body, err := endpoint.Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(string(body))
	// Output:
	// ready
}
