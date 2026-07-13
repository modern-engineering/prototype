package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

type Runtime struct {
	g group

	mu sync.Mutex

	ctx       context.Context
	ctxCancel func(error)

	procs []Proc
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
	// We must cancel the context when the group is done to release its associated
	// resources, just like errgroup does.
	r.cancel(err)
	return err
}

// Go calls the given function in a new goroutine.
//
// The first call to Go must happen before a Wait.
//
// The first goroutine in the group that returns a non-nil error will cancel the
// associated Context. Wait will return that error.
func (r *Runtime) Go(f func(context.Context) error) {
	r.Run(Main(f))
}

// Run starts the given application Runner in a new goroutine.
//
// The first call to Run must happen before a Wait.
//
// The first goroutine in the group that returns a non-nil error will cancel the
// associated Context. Wait will return that error.
func (r *Runtime) Run(rr Runner) {
	complete := r.track(rr)
	r.g.Go(func() error {
		defer complete()
		// It is tempting to propagate panics from f() up to the goroutine that calls
		// Wait, but it creates more problems than it solves. See comments on group.Go
		// for more details.
		//
		// See golang/go#53757, golang/go#74275, golang/go#74304, golang/go#74306.

		err := rr.Run(r.Context())
		slog.DebugContext(r.Context(), "application.Run returned prematurely", "error", err, slog.String("application", display(rr)))
		//r.cancel(err) // TODO(@danielorbach): cancel the context if the application returned before the runtime has requested
		if err != nil {
			r.cancel(err)
		}
		// TODO(@danielorbach): a long-lived runner must never complete before being asked to. Consider retry, when opted-in.

		// A long-lived runner, by definition, never completes before being required to.
		//if
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
// TODO: consider Identity() vs Equals(Runner) for treating two program values as "the same".
func (r *Runtime) track(rr Runner) (complete func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := Proc{rr: rr, done: make(chan struct{})}
	r.procs = append(r.procs, p)
	return p.complete
}

// Running returns a copy of the currently active Runners managed by this
// runtime.
//
// Thanks to the shallow clone, it is safe for concurrent-use with this type's
// methods. Modifying the returned slice has no side effects. However, the slice
// elements are copies of the same interface values, so any interaction with them
// may have unwanted side effects (e.g. data races). Use with caution.
func (r *Runtime) Running() []Proc {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.procs)
}

type Proc struct {
	rr   Runner
	done chan struct{}
}

func (p Proc) complete() {
	close(p.done)
}

func (p Proc) Done() <-chan struct{} {
	return p.done
}

func (p Proc) String() string {
	return display(p.rr)
}

// display renders a runner's identity without touching its state: a
// Stringer answers for itself under its own concurrency rules, and
// anything else renders as its type. Reflecting over a live runner's
// fields (as %v on a non-Stringer would) races with the runner's own
// lifecycle — a Shutdown mutating state while the runtime logs, say.
func display(rr Runner) string {
	if s, ok := rr.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%T", rr)
}

// TODO: consider using generic methods with go1.27 release instead of exposing the Runner directly.
//func (p Process) As[T any]() (v T, ok bool) {
//	v, ok = p.rr.(T)
//	return
//}

func (p Proc) Runner() Runner {
	return p.rr
}

func (r *Runtime) Cancel() {
	r.cancel(ErrCanceled)
}

// ErrCanceled is the error returned by [context.Cause] when the context is
// canceled explicitly by [Runtime.Cancel].
var ErrCanceled = errors.New("context canceled by app runtime")

// Shutdown asks every running Shutdowner to stop, concurrently, and
// waits for them all; the per-runner errors come back joined.
//
// The ctx bounds each runner's graceful window and is passed through
// verbatim. It is deliberately not the runtime's own context: a
// shutdown sequence has typically cancelled that one already (or is
// about to), and handing it to the Shutdowners — as earlier versions
// did — collapsed every graceful window to zero the moment it fired,
// turning each graceful stop into an abrupt one.
//
// Shutdown does not cancel the runtime's context; runners that outlive
// their graceful stop remain the caller's to release (Cancel, then
// Wait).
func (r *Runtime) Shutdown(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs error
	)
	for _, rr := range r.Running() {
		shutdowner, ok := rr.Runner().(Shutdowner)
		if !ok {
			continue
		}
		wg.Go(func() {
			err := shutdowner.Shutdown(ctx)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = errors.Join(errs, fmt.Errorf("%v: %w", rr, err))
			}
		})
	}
	wg.Wait()
	// TODO(@danielorbach): what happens here if there are still actively running goroutines?
	//  Either because the called Shutdowner returned while goroutines are still running,
	//  or because some of the Runners don't implement Shutdowner.
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
