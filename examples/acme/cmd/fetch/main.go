// Command fetch fetches an HTTP resource using the Acme catalogue.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"example.com/acme/catalogue"
)

func main() {
	if err := command(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func command() error {
	url := flag.String("url", "http://localhost:8080", "service URL")
	token := flag.String("token", "", "bearer credential")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	return run(ctx, client, *url, *token, os.Stdout)
}

// httpClientUser is the capability this command needs from its runner.
type httpClientUser interface {
	UseHTTPClient(*http.Client)
}

// run configures and runs Fetch while its caller keeps the shared client alive.
func run(ctx context.Context, client *http.Client, url, token string, out io.Writer) error {
	runner, inputs, outputs := catalogue.Fetch.Make()
	address, _ := inputs.Lookup("url")
	if err := address.Value.Set(url); err != nil {
		return err
	}
	credential, _ := inputs.Lookup("token")
	if err := credential.Value.Set(token); err != nil {
		return err
	}

	// The command supplies its client separately from declared configuration.
	runner.(httpClientUser).UseHTTPClient(client)
	if err := runner.Run(ctx); err != nil {
		return err
	}
	body, _ := outputs.Lookup("body")
	_, err := io.WriteString(out, body.Value.Get())
	return err
}
