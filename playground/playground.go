// Package playground runs a linked solution in one process using the job and
// application catalogues.
// Descriptors construct invocation state; the caller retains ownership of any
// resources injected into their constructors.
package playground

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// Options controls execution of a solution instance.
type Options struct {
	// JobTimeout bounds each job's Run call through its context.
	// Zero uses one minute; a negative value is invalid.
	// Cancellation is cooperative: Run waits for the job to return.
	JobTimeout time.Duration
}

// Run consumes one fresh instance without concurrently accessing its handles.
// The caller must not run the instance again or access its runners and handles
// until Run returns.
// instance must be non-nil.
//
// Before calling Set, Get, or Run, Run rejects unsupported descriptor types,
// references produced by applications, and dependency cycles.
// Only job.Descriptor, *job.Descriptor, application.Descriptor, and
// *application.Descriptor identify supported catalogue kinds.
//
// Jobs execute sequentially in dependency order.
// Among jobs whose dependencies have succeeded, the earliest authored occurrence
// runs first.
// Each invocation receives its assignments in authored binding order, including
// repeated assignments; unbound inputs keep their constructor defaults.
// References transfer strings through Getter.Get and Setter.Set.
// Each referenced output is read once after its producing job succeeds and before
// any consumer is configured.
// Outputs of failed or timed-out jobs are not read.
//
// Each job receives a context bounded by Options.JobTimeout.
// A failure or deadline stops execution before later jobs run or applications start.
// After every job succeeds, all applications are configured before any is started.
// Applications then run concurrently until shutdown.
// Returning before shutdown, with or without an error, is ErrUnexpectedExit and
// cancels the remaining applications.
// With no applications, Run returns after the jobs complete.
//
// Cancellation is cooperative: Run waits for every started runner to return.
// Runners must finish their background work and release resources they acquire
// before returning.
// Cancellation of ctx returns ctx.Err().
// Invalid options return ErrInvalidTimeout.
// Other failures return an *Error identifying the affected occurrence or binding.
// Its Error method omits the underlying cause, which can contain sensitive data;
// callers may inspect that cause through errors.Is, errors.As, or Unwrap.
func Run(ctx context.Context, instance *load.Instance, options Options) error {
	if options.JobTimeout < 0 {
		return ErrInvalidTimeout
	}
	if options.JobTimeout == 0 {
		options.JobTimeout = time.Minute
	}
	plan, err := prepare(instance)
	if err != nil {
		return err
	}
	values := make(map[outputKey]string)
	for _, invocation := range plan.jobs {
		if err := configure(ctx, instance, invocation, values); err != nil {
			return err
		}
		jobCtx, cancel := context.WithTimeout(ctx, options.JobTimeout)
		err := invocation.Runner().Run(jobCtx)
		contextErr := jobCtx.Err()
		cancel()
		if err := ctx.Err(); err != nil {
			return err
		}
		if contextErr != nil {
			return &Error{Node: invocation.Node(), Operation: "run job", Err: errors.Join(contextErr, err)}
		}
		if err != nil {
			return &Error{Node: invocation.Node(), Operation: "run job", Err: err}
		}
		for _, output := range plan.outputs[invocation.Node()] {
			if err := ctx.Err(); err != nil {
				return err
			}
			values[output.key] = output.value.Get()
		}
	}
	for _, invocation := range plan.apps {
		if err := configure(ctx, instance, invocation, values); err != nil {
			return err
		}
	}
	return runApplications(ctx, plan.apps)
}

type outputKey struct {
	node *link.Node
	name string
}

type snapshot struct {
	key   outputKey
	value program.Getter
}

type execution struct {
	jobs    []*load.Invocation
	apps    []*load.Invocation
	outputs map[*link.Node][]snapshot
}

type catalogueKind uint8

const (
	jobKind catalogueKind = iota
	applicationKind
)

func prepare(instance *load.Instance) (*execution, error) {
	invocations := slices.Collect(instance.Invocations())
	kinds := make(map[*link.Node]catalogueKind, len(invocations))
	plan := &execution{outputs: make(map[*link.Node][]snapshot)}
	var jobs []*load.Invocation
	for _, invocation := range invocations {
		switch invocation.Node().Descriptor().(type) {
		case job.Descriptor, *job.Descriptor:
			kinds[invocation.Node()] = jobKind
			jobs = append(jobs, invocation)
		case application.Descriptor, *application.Descriptor:
			kinds[invocation.Node()] = applicationKind
			plan.apps = append(plan.apps, invocation)
		default:
			return nil, &Error{Node: invocation.Node(), Operation: "classify descriptor", Err: ErrUnsupportedDescriptor}
		}
	}
	seen := make(map[outputKey]bool)
	for _, invocation := range invocations {
		for source := range invocation.Node().Bindings() {
			reference, ok := source.Value().(link.Reference)
			if !ok {
				continue
			}
			if kinds[reference.Node] == applicationKind {
				return nil, &Error{Node: invocation.Node(), Binding: source, Operation: "resolve output reference", Err: ErrApplicationReference}
			}
			key := outputKey{node: reference.Node, name: reference.Output}
			if !seen[key] {
				binding, _ := instance.Binding(source)
				output, _ := binding.Output()
				plan.outputs[reference.Node] = append(plan.outputs[reference.Node], snapshot{key: key, value: output.Value})
				seen[key] = true
			}
		}
	}
	ordered := make(map[*link.Node]bool, len(jobs))
	for len(plan.jobs) < len(jobs) {
		var next *load.Invocation
		for _, invocation := range jobs {
			if ordered[invocation.Node()] {
				continue
			}
			ready := true
			for dependency := range invocation.Node().Dependencies() {
				if !ordered[dependency] {
					ready = false
					break
				}
			}
			if ready {
				next = invocation
				break
			}
		}
		if next == nil {
			for _, invocation := range jobs {
				if !ordered[invocation.Node()] {
					return nil, &Error{Node: invocation.Node(), Operation: "order dependencies", Err: ErrDependencyCycle}
				}
			}
		}
		ordered[next.Node()] = true
		plan.jobs = append(plan.jobs, next)
	}
	return plan, nil
}

func configure(ctx context.Context, instance *load.Instance, invocation *load.Invocation, values map[outputKey]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for source := range invocation.Node().Bindings() {
		if err := ctx.Err(); err != nil {
			return err
		}
		binding, _ := instance.Binding(source)
		text, literal := binding.Literal()
		if !literal {
			reference := source.Value().(link.Reference)
			text = values[outputKey{node: reference.Node, name: reference.Output}]
		}
		err := binding.Input().Value.Set(text)
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			return &Error{Node: invocation.Node(), Binding: source, Operation: "set input", Err: err}
		}
	}
	return ctx.Err()
}

func runApplications(ctx context.Context, apps []*load.Invocation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	appCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var runners sync.WaitGroup
	var mu sync.Mutex
	var stopping bool
	var failure error
	for _, invocation := range apps {
		runners.Go(func() {
			err := invocation.Runner().Run(appCtx)
			mu.Lock()
			defer mu.Unlock()
			if !stopping {
				stopping = true
				if ctx.Err() == nil {
					failure = &Error{Node: invocation.Node(), Operation: "run application", Err: errors.Join(ErrUnexpectedExit, err)}
				}
				cancel()
			}
		})
	}
	runners.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	return failure
}
