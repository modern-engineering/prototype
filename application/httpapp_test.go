package application_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/application"
)

func TestHttpApplications(t *testing.T) {
	//t.Skip()
	r := application.RuntimeWithContext(t.Context())
	srv := &http.Server{
		Addr:    "localhost:0",
		Handler: http.HandlerFunc(okHandler),
	}
	r.Run((*application.HTTPServer)(srv))

	// Wait for server goroutine to serve for sure.
	time.Sleep(time.Second)

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
