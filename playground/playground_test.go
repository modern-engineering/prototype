package playground_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/playground"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// A job graph can be authored in any occurrence order while assignments retain
// their authored meaning and every consumer sees the same output snapshot.
func TestJobsCarrySnapshotsInAuthoredAssignmentOrder(t *testing.T) {
	var calls []string
	var assignments []string
	producer := &job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		return program.RunnerFunc(func(context.Context) error {
			calls = append(calls, "run producer")
			return nil
		}), nil, program.OutputList{
			{Field: program.Field{Name: "result"}, Value: program.GetterFunc(func() string {
				calls = append(calls, "get producer")
				return "produced"
			})},
			{Field: program.Field{Name: "unused"}, Value: program.GetterFunc(func() string {
				t.Fatal("read an unreferenced output")
				return ""
			})},
		}
	}}
	sink := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		defaultValue := "default"
		return program.RunnerFunc(func(context.Context) error {
			calls = append(calls, "run sink")
			if defaultValue != "default" {
				t.Errorf("unbound input = %q, want constructor default", defaultValue)
			}
			return nil
		}), program.InputList{
			{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(text string) error {
				assignments = append(assignments, text)
				calls = append(calls, "set sink")
				return nil
			})},
			{Field: program.Field{Name: "unbound"}, Value: program.SetterFunc(func(text string) error {
				defaultValue = text
				return nil
			})},
		}, nil
	}}
	independent := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		return program.RunnerFunc(func(context.Context) error {
			calls = append(calls, "run independent")
			return nil
		}), nil, nil
	}}
	other := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		return program.RunnerFunc(func(context.Context) error {
			calls = append(calls, "run other")
			return nil
		}), program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(text string) error {
			if text != "produced" {
				t.Errorf("other consumer received %q, want produced", text)
			}
			calls = append(calls, "set other")
			return nil
		})}}, nil
	}}
	instance := instantiate(t, map[string]program.Descriptor{"Sink": sink, "Independent": independent, "Producer": producer, "Other": other}, solution.Definition{
		Nodes: []solution.Node{node("sink", "Sink"), node("independent", "Independent"), node("producer", "Producer"), node("other", "Other")},
		Bindings: []solution.Binding{
			{Node: "sink", Input: "text", Value: solution.Literal("first")},
			{Node: "other", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
			{Node: "sink", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
			{Node: "sink", Input: "text", Value: solution.Literal("last")},
			{Node: "sink", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
		},
	})
	if err := playground.Run(t.Context(), instance, playground.Options{}); err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{"run independent", "run producer", "get producer", "set sink", "set sink", "set sink", "set sink", "run sink", "set other", "run other"}
	if !slices.Equal(calls, wantCalls) {
		t.Errorf("calls = %v, want %v", calls, wantCalls)
	}
	if want := []string{"first", "produced", "last", "produced"}; !slices.Equal(assignments, want) {
		t.Errorf("assignments = %v, want %v", assignments, want)
	}
}

// Rejected graphs must not partially configure or execute an otherwise valid job.
func TestPreflightRejectsBeforeHandleAccess(t *testing.T) {
	for _, tc := range []struct {
		name       string
		descriptor program.Descriptor
		bindings   []solution.Binding
		options    playground.Options
		want       error
	}{
		{name: "negative timeout", options: playground.Options{JobTimeout: -time.Second}, want: playground.ErrInvalidTimeout},
		{name: "unknown catalogue", descriptor: program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
			return program.RunnerFunc(func(context.Context) error { return nil }), nil, nil
		}), want: playground.ErrUnsupportedDescriptor},
		{name: "application pending", descriptor: &application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
			return program.RunnerFunc(func(context.Context) error { return nil }), nil
		}}, want: playground.ErrApplicationsPending},
		{name: "self cycle", bindings: []solution.Binding{{Node: "first", Input: "text", Value: solution.Reference{Node: "first", Output: "result"}}}, want: playground.ErrDependencyCycle},
		{name: "mutual cycle", bindings: []solution.Binding{
			{Node: "first", Input: "text", Value: solution.Reference{Node: "second", Output: "result"}},
			{Node: "second", Input: "text", Value: solution.Reference{Node: "first", Output: "result"}},
		}, want: playground.ErrDependencyCycle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			untouched := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
				return program.RunnerFunc(func(context.Context) error { t.Fatal("preflight ran a job"); return nil }),
					program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(string) error { t.Fatal("preflight assigned an input"); return nil })}},
					program.OutputList{{Field: program.Field{Name: "result"}, Value: program.GetterFunc(func() string { t.Fatal("preflight read an output"); return "" })}}
			}}
			second := tc.descriptor
			if second == nil {
				second = untouched
			}
			bindings := append([]solution.Binding{{Node: "first", Input: "text", Value: solution.Literal("authored")}}, tc.bindings...)
			instance := instantiate(t, map[string]program.Descriptor{"First": untouched, "Second": second}, solution.Definition{
				Nodes: []solution.Node{node("first", "First"), node("second", "Second")}, Bindings: bindings,
			})
			if err := playground.Run(t.Context(), instance, tc.options); !errors.Is(err, tc.want) {
				t.Fatalf("Run error = %v, want %v", err, tc.want)
			}
		})
	}
}

// Arbitrary setter and runner errors can carry secrets even when no field was
// marked sensitive, so diagnostics expose location while preserving the cause.
func TestFailuresStopExecutionWithoutDisclosingCauses(t *testing.T) {
	for _, operation := range []string{"set input", "run job"} {
		t.Run(operation, func(t *testing.T) {
			cause := errors.New("private-credential-value")
			broken := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
				return program.RunnerFunc(func(context.Context) error {
					if operation == "set input" {
						t.Fatal("ran a job after its input failed")
					}
					return cause
				}), program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(string) error {
					if operation == "set input" {
						return cause
					}
					return nil
				})}}, program.OutputList{{Field: program.Field{Name: "result"}, Value: program.GetterFunc(func() string {
					t.Fatal("read an output from a failed job")
					return ""
				})}}
			}}
			later := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
				return program.RunnerFunc(func(context.Context) error { t.Fatal("ran after a failure"); return nil }),
					program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(string) error { t.Fatal("assigned after a failure"); return nil })}}, nil
			}}
			instance := instantiate(t, map[string]program.Descriptor{"Broken": broken, "Later": later}, solution.Definition{
				Nodes: []solution.Node{node("broken", "Broken"), node("later", "Later")},
				Bindings: []solution.Binding{
					{Node: "broken", Input: "text", Value: solution.Literal("private-literal-value")},
					{Node: "later", Input: "text", Value: solution.Reference{Node: "broken", Output: "result"}},
				},
			})
			err := playground.Run(t.Context(), instance, playground.Options{})
			failure, ok := errors.AsType[*playground.Error](err)
			if !ok || !errors.Is(err, cause) || failure.Node.Name() != "broken" || failure.Operation != operation {
				t.Fatalf("Run error = %#v, want contextual %s failure retaining its cause", err, operation)
			}
			if operation == "set input" && (failure.Binding == nil || failure.Binding.Input() != "text") {
				t.Fatal("setter failure lost binding identity")
			}
			if strings.Contains(err.Error(), "private-") {
				t.Errorf("diagnostic disclosed a private value: %v", err)
			}
		})
	}
}

// A runner can return nil after observing cancellation; expiration still means
// the job failed, and only its successful outputs may reach later invocations.
func TestJobDeadlinesAreCooperative(t *testing.T) {
	for _, timeout := range []time.Duration{0, 5 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				wantTimeout := timeout
				if wantTimeout == 0 {
					wantTimeout = time.Minute
				}
				expired := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
					return program.RunnerFunc(func(ctx context.Context) error {
						deadline, ok := ctx.Deadline()
						if !ok || deadline.Sub(time.Now()) != wantTimeout {
							t.Errorf("job deadline = %v, %v; want %v from now", deadline, ok, wantTimeout)
						}
						<-ctx.Done()
						time.Sleep(time.Second)
						return nil
					}), nil, program.OutputList{{Field: program.Field{Name: "result"}, Value: program.GetterFunc(func() string {
						t.Fatal("read a timed-out output")
						return ""
					})}}
				}}
				later := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
					return program.RunnerFunc(func(context.Context) error { t.Fatal("ran after a deadline"); return nil }),
						program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(string) error { t.Fatal("assigned after a deadline"); return nil })}}, nil
				}}
				instance := instantiate(t, map[string]program.Descriptor{"Expired": expired, "Later": later}, solution.Definition{
					Nodes:    []solution.Node{node("expired", "Expired"), node("later", "Later")},
					Bindings: []solution.Binding{{Node: "later", Input: "text", Value: solution.Reference{Node: "expired", Output: "result"}}},
				})
				started := time.Now()
				if err := playground.Run(t.Context(), instance, playground.Options{JobTimeout: timeout}); !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("Run error = %v, want deadline exceeded", err)
				}
				if elapsed := time.Since(started); elapsed != wantTimeout+time.Second {
					t.Errorf("Run duration = %v, want deadline plus cooperative return delay", elapsed)
				}
			})
		})
	}
}

// Caller cancellation is the final result even when the current job also fails.
func TestCallerCancellationTakesPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var calls int
	descriptor := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		return program.RunnerFunc(func(context.Context) error {
			calls++
			cancel()
			return errors.New("runner error")
		}), nil, nil
	}}
	instance := instantiate(t, map[string]program.Descriptor{"Job": descriptor}, solution.Definition{
		Nodes: []solution.Node{node("first", "Job"), node("later", "Job")},
	})
	if err := playground.Run(ctx, instance, playground.Options{}); err != context.Canceled {
		t.Fatalf("Run error = %v, want caller cancellation", err)
	}
	if calls != 1 {
		t.Errorf("Run calls = %d, want only the first job", calls)
	}
}

func instantiate(t *testing.T, descriptors map[string]program.Descriptor, definition solution.Definition) *load.Instance {
	t.Helper()
	var registry program.Registry
	for name, descriptor := range descriptors {
		if err := registry.Register("local", name, descriptor); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := link.Resolve(definition, &registry)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func node(name, descriptor string) solution.Node {
	return solution.Node{Name: name, Definition: solution.Symbol{Package: "local", Name: descriptor}}
}
