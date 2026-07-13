package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
)

// An HTTPServer adapts an http.Server into a Runner: convert a
// configured server — (*HTTPServer)(srv) — and Run serves it until the
// context ends, closing abruptly on cancellation; Shutdown is the
// graceful sibling, draining under the caller's context.
type HTTPServer http.Server

func (a *HTTPServer) Run(ctx context.Context) error {
	srv := (*http.Server)(a)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		<-ctx.Done()
		err := srv.Close()
		if err != nil {
			return fmt.Errorf("close server: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	})
	return g.Wait()
}

func (a *HTTPServer) Shutdown(ctx context.Context) error {
	srv := (*http.Server)(a)
	return srv.Shutdown(ctx)
}
