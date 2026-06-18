package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/modern-engineering/prototype/application"
)

// CancelsOnErrors demonstrates how the first error cancels the context returned
// by [RuntimeWithContext]. Thus, signalling to other goroutines in that same
// runtime group to return.
func ExampleRuntime_cancelsOnError() {
	r, ctx := application.RuntimeWithContext(context.Background())
	r.Go(func() error { return errors.New("runtime: doomed") })
	r.Go(func() error {
		fmt.Println("Waiting for other goroutine to fail...")
		select {
		case <-ctx.Done():
			fmt.Println("Done; returning.")
			return nil
		}
	})

	gErr := r.Wait()
	fmt.Printf("Wait() = %v\n", gErr)

	cause := context.Cause(ctx)
	fmt.Printf("context.Cause(ctx) = %v\n", cause)

	// Output:
	// Waiting for other goroutine to fail...
	// Done; returning.
	// Wait() = runtime: doomed
	// context.Cause(ctx) = runtime: doomed
}

func TestZeroRuntime(t *testing.T) {
	err1 := errors.New("runtime_test: 1")
	err2 := errors.New("runtime_test: 2")

	cases := []struct {
		errs []error
	}{
		{errs: []error{}},
		{errs: []error{nil}},
		{errs: []error{err1}},
		{errs: []error{err1, nil}},
		{errs: []error{err1, nil, err2}},
	}

	synctest.Test(t, func(t *testing.T) {
		for _, tc := range cases {
			r := new(application.Runtime)

			var firstErr error
			for i, err := range tc.errs {
				r.Go(func() error { return err })

				if firstErr == nil && err != nil {
					firstErr = err
				}

				if gErr := r.Wait(); gErr != firstErr {
					t.Logf("after %T.Go(func() error { return %q }) for err #%d in %+v", r, err, i, tc.errs)
					t.Errorf("r.Wait() = %v; want %v", gErr, firstErr)
				}
			}
		}
	})
}

func TestRuntimeWithContext(t *testing.T) {
	errDoom := errors.New("runtime_test: doomed")

	cases := []struct {
		errs []error
		want error
	}{
		{want: nil},
		{errs: []error{nil}, want: nil},
		{errs: []error{errDoom}, want: errDoom},
		{errs: []error{errDoom, nil}, want: errDoom},
	}

	synctest.Test(t, func(t *testing.T) {
		for _, tc := range cases {
			r, ctx := application.RuntimeWithContext(context.Background())

			for _, err := range tc.errs {
				r.Go(func() error { return err })
			}

			if gErr := r.Wait(); gErr != tc.want {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("r.Wait() = %v; want %v", gErr, tc.want)
			}

			canceled := false
			select {
			case <-ctx.Done():
				canceled = true
			default:
			}
			if !canceled {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("ctx.Done() was not closed")
			}

			// When Wait returns nil, it cancels the context anyway, without a dedicated
			// cause.
			if tc.want == nil {
				tc.want = context.Canceled
			}
			if cause := context.Cause(ctx); cause != tc.want {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("context.Cause(ctx) = %v; want %v", cause, tc.want)
			}
		}
	})
}
func BenchmarkGo(b *testing.B) {
	fn := func() {}
	r := new(application.Runtime)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		r.Go(func() error { fn(); return nil })
	}
	_ = r.Wait()
}
