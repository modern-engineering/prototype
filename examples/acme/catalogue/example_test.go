package catalogue_test

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"

	"example.com/acme/catalogue"
	"github.com/modern-engineering/prototype/program/command"
)

// Your executable can expose the catalogue's fields as flags and formatted
// results. It still chooses the client and owns its transport independently.
func ExampleFetch() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, "ready"); err != nil {
			fmt.Println(err)
		}
	}))
	defer server.Close()

	runner, inputs, outputs := catalogue.Fetch.Make()
	flags := flag.NewFlagSet(catalogue.Fetch.Name, flag.ContinueOnError)
	if err := command.Parse(flags, inputs, []string{"-url=" + server.URL}); err != nil {
		fmt.Println(err)
		return
	}
	runner.(interface{ UseHTTPClient(*http.Client) }).UseHTTPClient(server.Client())
	if err := runner.Run(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	if err := command.WriteOutputs(os.Stdout, outputs); err != nil {
		fmt.Println(err)
	}
	// Output:
	// {"body":"ready\n"}
}
