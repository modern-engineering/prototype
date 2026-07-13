package application_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
)

// A converted http.Server runs as a hosted application: Run serves the
// handler, Shutdown drains the listener gracefully, and Cancel releases
// the close-watcher Run parks on. This is the package's one real-socket
// test — the kernel-integration pin for the listening boundary; a
// listener-injection seam stays unneeded because the wrapped server's
// own BaseContext hook already surfaces the bound address, which is all
// readiness takes.
func TestHTTPServerServesUntilShutdown(t *testing.T) {
	r := application.RuntimeWithContext(t.Context())

	// BaseContext fires once the kernel-chosen port is listening,
	// so receiving the address IS the readiness signal — no sleeps.
	addrs := make(chan net.Addr, 1)
	srv := &http.Server{
		Addr:    "localhost:0",
		Handler: http.HandlerFunc(okHandler),
		BaseContext: func(l net.Listener) context.Context {
			addrs <- l.Addr()
			return context.Background()
		},
	}
	r.Run((*application.HTTPServer)(srv))

	resp, err := http.Get("http://" + (<-addrs).String())
	if err != nil {
		t.Fatalf("GET while running: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "SUCCESS" {
		t.Errorf("GET = %d %q (%v), want 200 SUCCESS", resp.StatusCode, body, err)
	}

	if err := r.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown() = %v; want nil", err)
	}

	// Shutdown stops the listener gracefully but never cancels the
	// runtime's own context — that stays the caller's decision — and
	// HTTPServer.Run holds its close-watcher goroutine until the
	// context ends, so Cancel releases the runner before Wait.
	r.Cancel()
	if err := r.Wait(); err != nil {
		t.Errorf("Wait() = %v; want nil", err)
	}
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SUCCESS", http.StatusOK)
}
