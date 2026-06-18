package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/modern-engineering/prototype/application"
)

// CancelsOnErrors demonstrates how the first error cancels the context passed to
// sibling application programs. Thus, signalling to other goroutines in that
// same runtime group to return.
func ExampleRuntime_cancelsOnError() {
	var r application.Runtime
	r.Go(func(context.Context) error { return errors.New("runtime: doomed") })
	r.Go(func(ctx context.Context) error {
		fmt.Println("Waiting for other goroutine to fail...")
		select {
		case <-ctx.Done():
			cause := context.Cause(ctx)
			fmt.Printf("Done; context.Cause(ctx) = %v\n", cause)
			return nil
		}
	})

	gErr := r.Wait()
	fmt.Printf("Wait() = %v\n", gErr)

	// Output:
	// Waiting for other goroutine to fail...
	// Done; context.Cause(ctx) = runtime: doomed
	// Wait() = runtime: doomed
}

func ExampleRuntime_Programs() {
	var r application.Runtime
	r.Go(func(context.Context) error { return nil })
	r.Go(func(context.Context) error { return nil })
	r.Go(func(context.Context) error { return nil })
	_ = r.Wait()

	for p := range r.Programs() {
		fmt.Println(p)
	}

	// Output:
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
				r.Go(func(context.Context) error { return err })

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
			r := application.RuntimeWithContext(t.Context())

			for _, err := range tc.errs {
				r.Go(func(context.Context) error { return err })
			}

			if gErr := r.Wait(); gErr != tc.want {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("r.Wait() = %v; want %v", gErr, tc.want)
			}

			canceled := false
			select {
			case <-r.Context().Done():
				canceled = true
			default:
			}
			if !canceled {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("Runtime.Context().Done() was not closed")
			}

			// When Wait returns nil, it cancels the context anyway, without a dedicated
			// cause.
			if tc.want == nil {
				tc.want = context.Canceled
			}
			if cause := context.Cause(r.Context()); cause != tc.want {
				t.Logf("after %T.Go(func() error { return err }) for err in %+v", r, tc.errs)
				t.Errorf("context.Cause(Runtime.Context()) = %v; want %v", cause, tc.want)
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
		r.Go(func(context.Context) error { fn(); return nil })
	}
	_ = r.Wait()
}
