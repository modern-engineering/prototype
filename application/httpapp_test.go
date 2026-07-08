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

	// TODO(@danielorbach) why doesn't Shutdown cause the Runner goroutine to return? Why must we call Cancel()?
	r.Cancel()
	if err := r.Wait(); err != nil {
		t.Errorf("Wait() = %v; want nil", err)
	}
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "SUCCESS", http.StatusOK)
}
