package playground_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/playground"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
)

// Preparation includes unrelated jobs and every application's configuration.
// Caller cancellation joins both applications, whether their shutdown returns
// nil or the context error.
func TestApplicationsStartAfterPreparationAndJoinOnCancellation(t *testing.T) {
	for _, shutdownError := range []bool{false, true} {
		name := "nil shutdown result"
		if shutdownError {
			name = "context shutdown result"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var prepared bool
				var configured, gets int
				started := make(chan string, 2)
				stopped := make(chan string, 2)
				producer := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
					return program.RunnerFunc(func(context.Context) error { return nil }), nil,
						program.OutputList{{Field: program.Field{Name: "result"}, Value: program.GetterFunc(func() string {
							gets++
							return "prepared input"
						})}}
				}}
				unrelated := job.Descriptor{MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
					return program.RunnerFunc(func(context.Context) error { prepared = true; return nil }), nil, nil
				}}
				app := func(name string) application.Descriptor {
					return application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
						var text string
						return program.RunnerFunc(func(ctx context.Context) error {
							if !prepared || configured != 2 || text != "prepared input" {
								t.Errorf("%s started with preparation=%v, configured=%d, input=%q", name, prepared, configured, text)
							}
							started <- name
							<-ctx.Done()
							time.Sleep(time.Second)
							stopped <- name
							if shutdownError {
								return ctx.Err()
							}
							return nil
						}), program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(value string) error {
							configured++
							text = value
							return nil
						})}}
					}}
				}
				first := app("first")
				second := app("second")
				instance := instantiate(t, map[string]program.Descriptor{"First": first, "Second": &second, "Producer": producer, "Unrelated": unrelated}, solution.Definition{
					Nodes: []solution.Node{node("first", "First"), node("producer", "Producer"), node("second", "Second"), node("unrelated", "Unrelated")},
					Bindings: []solution.Binding{
						{Node: "first", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
						{Node: "second", Input: "text", Value: solution.Reference{Node: "producer", Output: "result"}},
					},
				})
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				done := make(chan struct{})
				go func() {
					defer close(done)
					if err := playground.Run(ctx, instance, playground.Options{}); err != context.Canceled {
						t.Errorf("Run error = %v, want caller cancellation", err)
					}
				}()
				<-started
				<-started
				cancel()
				<-done
				if len(stopped) != 2 {
					t.Errorf("stopped applications = %d, want both joined", len(stopped))
				}
				if gets != 1 {
					t.Errorf("output reads = %d, want one snapshot shared by both applications", gets)
				}
			})
		})
	}
}

// An early return is a failure even when nil or shaped like cancellation.
// Its initiating cause survives the peer's cancellation result and cleanup delay.
func TestUnexpectedApplicationExitCancelsAndJoinsPeers(t *testing.T) {
	privateCause := errors.New("private-application-value")
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{name: "nil result"},
		{name: "runner error", cause: privateCause},
		{name: "unexpected context error", cause: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				peerStopped := false
				broken := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
					return program.RunnerFunc(func(context.Context) error { return tc.cause }), nil
				}}
				peer := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
					return program.RunnerFunc(func(ctx context.Context) error {
						<-ctx.Done()
						time.Sleep(time.Second)
						peerStopped = true
						return ctx.Err()
					}), nil
				}}
				instance := instantiate(t, map[string]program.Descriptor{"Broken": broken, "Peer": peer}, solution.Definition{
					Nodes: []solution.Node{node("broken", "Broken"), node("peer", "Peer")},
				})
				err := playground.Run(t.Context(), instance, playground.Options{})
				failure, ok := errors.AsType[*playground.Error](err)
				if !ok || !errors.Is(err, playground.ErrUnexpectedExit) || failure.Node.Name() != "broken" || failure.Operation != "run application" {
					t.Fatalf("Run error = %#v, want contextual unexpected exit from broken", err)
				}
				if tc.cause != nil && !errors.Is(err, tc.cause) {
					t.Errorf("Run error = %v, want original runner cause", err)
				}
				if !peerStopped {
					t.Error("Run returned before peer cleanup")
				}
				if strings.Contains(err.Error(), "private-") {
					t.Errorf("diagnostic disclosed the application error: %v", err)
				}
			})
		})
	}
}

// Configuring applications is a separate phase: a later setter failure must not
// leave an earlier, fully configured application running.
func TestApplicationConfigurationFailureStartsNone(t *testing.T) {
	cause := errors.New("private-configuration-value")
	configured := 0
	app := application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
		return program.RunnerFunc(func(context.Context) error { t.Error("ran an application after configuration failed"); return nil }),
			program.InputList{{Field: program.Field{Name: "text"}, Value: program.SetterFunc(func(value string) error {
				configured++
				if value == "fail" {
					return cause
				}
				return nil
			})}}
	}}
	instance := instantiate(t, map[string]program.Descriptor{"App": app}, solution.Definition{
		Nodes: []solution.Node{node("first", "App"), node("second", "App")},
		Bindings: []solution.Binding{
			{Node: "first", Input: "text", Value: solution.Literal("succeed")},
			{Node: "second", Input: "text", Value: solution.Literal("fail")},
		},
	})
	err := playground.Run(t.Context(), instance, playground.Options{})
	failure, ok := errors.AsType[*playground.Error](err)
	if !ok || !errors.Is(err, cause) || failure.Node.Name() != "second" || failure.Binding == nil || failure.Binding.Input() != "text" {
		t.Fatalf("Run error = %#v, want contextual setter failure from second", err)
	}
	if configured != 2 {
		t.Errorf("configuration calls = %d, want both applications", configured)
	}
	if strings.Contains(err.Error(), "private-") {
		t.Errorf("diagnostic disclosed the configuration error: %v", err)
	}
}

func TestEmptySolutionCompletes(t *testing.T) {
	instance := instantiate(t, nil, solution.Definition{})
	if err := playground.Run(t.Context(), instance, playground.Options{}); err != nil {
		t.Fatal(err)
	}
}

func forbiddenApplication(t *testing.T) application.Descriptor {
	t.Helper()
	return application.Descriptor{MakeFunc: func() (program.Runner, program.Inputs) {
		return program.RunnerFunc(func(context.Context) error { t.Error("ran an application after a job failed"); return nil }), nil
	}}
}
