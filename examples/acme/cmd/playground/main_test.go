package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
)

// The custom host must finish both preparation jobs before either application
// starts and wait for both applications to stop before returning.
func TestPreparationAndShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var output bytes.Buffer
		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := run(ctx, &output); !errors.Is(err, context.Canceled) {
				t.Errorf("run: got %v, want cancellation", err)
			}
		}()
		synctest.Wait()
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		jobs := []string{"environment: development -> eu-west", "database: database.eu-west.example"}
		if len(lines) != 4 || !slices.Equal(lines[:2], jobs) {
			t.Fatalf("before shutdown: got %q, want preparation jobs followed by two application starts", lines)
		}
		slices.Sort(lines[2:])
		starts := []string{"api: started with database.eu-west.example", "reports: started with database.eu-west.example"}
		if !slices.Equal(lines[2:], starts) {
			t.Errorf("application starts: got %q, want %q", lines[2:], starts)
		}
		cancel()
		<-done
		lines = strings.Split(strings.TrimSpace(output.String()), "\n")
		if len(lines) != 6 {
			t.Fatalf("after shutdown: got %q, want both applications stopped", lines)
		}
		slices.Sort(lines[4:])
		stops := []string{"api: stopped", "reports: stopped"}
		if !slices.Equal(lines[4:], stops) {
			t.Errorf("application stops: got %q, want %q", lines[4:], stops)
		}
	})
}
