package catalogue_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"example.com/acme/catalogue"
)

// Configuration is available through the descriptor's inputs; the calling
// executable chooses the HTTP client and retains ownership of its transport.
func ExampleFetch() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, "ready"); err != nil {
			fmt.Println(err)
		}
	}))
	defer server.Close()

	runner, inputs, outputs := catalogue.Fetch.Make()
	url, _ := inputs.Lookup("url")
	if err := url.Value.Set(server.URL); err != nil {
		fmt.Println(err)
		return
	}
	runner.(interface{ UseHTTPClient(*http.Client) }).UseHTTPClient(server.Client())
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	body, _ := outputs.Lookup("body")
	fmt.Print(body.Value.Get())
	// Output:
	// ready
}
