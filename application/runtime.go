package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

type Runtime struct {
	g group

	mu sync.Mutex

	ctx       context.Context
	ctxCancel func(error)

	runners []Runner
}

// RuntimeWithContext returns a new Runtime group and an associated Context
// derived from ctx.
//
// The derived Context is cancelled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func RuntimeWithContext(ctx context.Context) *Runtime {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Runtime{ctx: ctx, ctxCancel: cancel}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (r *Runtime) Wait() error {
	err := r.g.Wait()
	r.cancel(err)
	return err
}

// Go calls the given function in a new goroutine.
//
// The first call to Go must happen before a Wait.
//
// The first goroutine in the group that returns a non-nil error will cancel the
// associated Context. The error will be returned by Wait.
func (r *Runtime) Go(f func(context.Context) error) {
	r.Run(Main(f))
}

// Run starts the given application Runner in a new goroutine.
//
// The first call to Run must happen before a Wait.
//
// The first goroutine in the group that returns a non-nil error will cancel the
// associated Context. The error will be returned by Wait.
func (r *Runtime) Run(rr Runner) {
	r.track(rr)
	// TODO(@danielorbach): untrack a completed program.
	r.g.Go(func() error {
		// It is tempting to propagate panics from f() up to the goroutine that calls
		// Wait, but it creates more problems than it solves. See comments on group.Go
		// for more details.
		//
		// See golang/go#53757, golang/go#74275, golang/go#74304, golang/go#74306.

		err := rr.Run(r.Context())
		if err != nil {
			r.cancel(err)
		}
		// TODO(@danielorbach): a long-lived runner must never complete before being asked to.
		return err
	})
}

func (r *Runtime) Context() context.Context {
	r.initZeroContext()
	return r.ctx
}

func (r *Runtime) cancel(err error) {
	r.initZeroContext()
	r.ctxCancel(err)
}

func (r *Runtime) initZeroContext() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ctx == nil {
		r.ctx, r.ctxCancel = context.WithCancelCause(context.Background())
	}
}

// TODO: think about running the same program value twice.
func (r *Runtime) track(rr Runner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// A program is an interface, so its equality operator compares the value's
	// memory address. Tracking the same value (i.e. Go variables, not byte-identical
	// values) only appends it once.
	// TODO: consider Identity() vs Equals(Runner) for treating two program values as "the same".
	//if slices.Contains(r.runners, rr) {
	//	return
	//}
	r.runners = append(r.runners, rr)
}

// Running returns a copy of the currently active Runners managed by this
// runtime.
//
// Thanks to the shallow clone, it is safe for concurrent-use with this type's
// methods. Modifying the returned slice has no side effects. However, the slice
// elements are copies of the same interface values, so any interaction with them
// may have unwanted side effects (e.g. data races). Use with caution.
func (r *Runtime) Running() []Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.runners)
}

func (r *Runtime) Shutdown(context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs error
	)
	for _, rr := range r.Running() {
		shutdowner, ok := rr.(Shutdowner)
		if !ok {
			continue
		}
		wg.Go(func() {
			err := shutdowner.Shutdown(r.Context())
			if err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = errors.Join(errs, fmt.Errorf("%v: %w", rr, err))
			}
		})
	}
	wg.Wait()
	return errs
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
		// It is tempting to propagate panics from f() up to the goroutine that calls
		// Wait, but it creates more problems than it solves:
		//
		// - it delays panics arbitrarily, making bugs harder to detect;
		// - it turns f's panic stack into a mere value, hiding it from
		//   crash-monitoring tools;
		// - it risks deadlocks that hide the panic entirely, if f's panic
		//   leaves the program in a state that prevents the Wait call from
		//   being reached.
		//
		// See golang/go#53757, golang/go#74275, golang/go#74304, golang/go#74306.

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
			})
		}
	})
}
