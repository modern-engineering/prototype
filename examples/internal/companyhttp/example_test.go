package companyhttp_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/examples/internal/companyhttp"
)

// Configuration binds through the usual input surface; the HTTP client is
// supplied separately and stays owned by the caller.
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

	endpoint := companyhttp.Endpoint{URL: server.URL}
	var inputs input.List
	endpoint.BindInputs(&inputs)
	token, _ := inputs.Lookup("token")
	if err := token.Value.Set("example-token"); err != nil {
		fmt.Println(err)
		return
	}
	endpoint.UseHTTPClient(server.Client())
	body, err := endpoint.Get(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("sensitive:", token.Sensitive)
	fmt.Print(string(body))
	// Output:
	// sensitive: true
	// ready
}
