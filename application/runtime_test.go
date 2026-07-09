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
// sibling applications. Thus, signalling to other goroutines in that same
// runtime group to return.
func ExampleRuntime_cancelsOnError() {
	var r application.Runtime
	r.Go(func(context.Context) error { return errors.New("runtime: doomed") })
	r.Go(func(ctx context.Context) error {
		fmt.Println("Waiting for other goroutine to fail...")
		<-ctx.Done()
		cause := context.Cause(ctx)
		fmt.Printf("Done; context.Cause(ctx) = %v\n", cause)
		return nil
	})

	gErr := r.Wait()
	fmt.Printf("Wait() = %v\n", gErr)

	// Output:
	// Waiting for other goroutine to fail...
	// Done; context.Cause(ctx) = runtime: doomed
	// Wait() = runtime: doomed
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
func ExampleRuntime_Running() {
	var r application.Runtime
	r.Go(func(context.Context) error { return nil })
	r.Go(func(context.Context) error { return nil })
	r.Go(func(context.Context) error { return nil })
	_ = r.Wait()

	for i, p := range r.Running() {
		fmt.Printf("App #%v: %v\n", i, p)
	}

	// Output:
	// App #0: github.com/modern-engineering/prototype/application_test.ExampleRuntime_Running.func1
	// App #1: github.com/modern-engineering/prototype/application_test.ExampleRuntime_Running.func2
	// App #2: github.com/modern-engineering/prototype/application_test.ExampleRuntime_Running.func3
}

func ExampleRuntime_Shutdown() {
	var r application.Runtime
	r.Run(&ShutdownRunner{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	})
	if err := r.Shutdown(context.Background()); err != nil {
		fmt.Printf("Did not shutdown gracefully within deadline: %v\n", err)
	}

	// Output:
	// Shutting down runner gracefully
}

type ShutdownRunner struct {
	stop, done chan struct{}
}

func (p *ShutdownRunner) Run(ctx context.Context) error {
	defer func() { p.done <- struct{}{} }()
	select {
	case <-p.stop:
		fmt.Println("Shutting down runner gracefully")
	case <-ctx.Done():
		fmt.Println("Forced to quit the runner abruptly")
	}
	return nil
}

func (p *ShutdownRunner) Shutdown(ctx context.Context) error {
	p.stop <- struct{}{}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *ShutdownRunner) String() string {
	return "shutdown-runner"
}
