package application

import (
	"context"
	"sync"
)

type Runtime struct {
	g      group
	cancel func(error)
}

// RuntimeWithContext returns a new Runtime group and an associated Context
// derived from ctx.
//
// The derived Context is cancelled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func RuntimeWithContext(ctx context.Context) (*Runtime, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Runtime{cancel: cancel}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (r *Runtime) Wait() error {
	err := r.g.Wait()
	if r.cancel != nil {
		r.cancel(err)
	}
	return err
}

// Go calls the given function in a new goroutine.
//
// The first call to Go must happen before a Wait.
// It blocks until the new goroutine can be added without the number of
// goroutines in the group exceeding the configured limit.
//
// The first goroutine in the group that returns a non-nil error will
// cancel the associated Context, if any. The error will be returned
// by Wait.
func (r *Runtime) Go(f func() error) {
	r.g.Go(func() error {
		// It is tempting to propagate panics from f() up to the goroutine that calls
		// Wait, but it creates more problems than it solves. See comments on group.Go
		// for more details.
		//
		// See golang/go#53757, golang/go#74275, golang/go#74304, golang/go#74306.

		err := f()
		if err != nil && r.cancel != nil {
			r.cancel(err)
		}
		return err
	})
}

// A group is a collection of goroutines working on subtasks that are part of
// the same overall task. A group should not be reused for different tasks.
//
// A zero group is valid, has no limit on the number of active goroutines,
// and does not cancel on error.
type group struct {
	wg sync.WaitGroup

	errOnce sync.Once
	err     error
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *group) Wait() error {
	g.wg.Wait()
	return g.err
}

// Go calls the given function in a new goroutine.
//
// The first call to Go must happen before a Wait.
// It blocks until the new goroutine can be added without the number of
// goroutines in the group exceeding the configured limit.
//
// The first goroutine in the group that returns a non-nil error will
// cancel the associated Context, if any. The error will be returned
// by Wait.
func (g *group) Go(f func() error) {
	g.wg.Go(func() {
		// It is tempting to propagate panics from f()
		// up to the goroutine that calls Wait, but
		// it creates more problems than it solves:
		// - it delays panics arbitrarily,
		//   making bugs harder to detect;
		// - it turns f's panic stack into a mere value,
		//   hiding it from crash-monitoring tools;
		// - it risks deadlocks that hide the panic entirely,
		//   if f's panic leaves the program in a state
		//   that prevents the Wait call from being reached.
		// See #53757, #74275, #74304, #74306.

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
			})
		}
	})
}
