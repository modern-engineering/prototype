package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/program"
)

// A loader discovers declarations, configures a finite job, and wires its result
// into an application through the same factory contract.
func Example() {
	l := loader{trace: os.Stdout}
	address, err := l.load(Address, map[string]program.Getter{
		"host": literal("api.example.test:8443"),
	})
	if err != nil {
		fmt.Println("load address:", err)
		return
	}
	if err := address.runner.Run(context.Background()); err != nil {
		fmt.Println("run address:", err)
		return
	}
	url, err := address.output("url")
	if err != nil {
		fmt.Println("address output:", err)
		return
	}
	backend, err := l.load(Backend, map[string]program.Getter{"address": url})
	if err != nil {
		fmt.Println("load backend:", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := backend.runner.Run(ctx); err != nil {
		fmt.Println("run backend:", err)
	}
	// Output:
	// input host: service host and port
	// input scheme: service URL scheme
	// input address: upstream service URL
	// backend configured: https://api.example.test:8443
}

// Unknown bindings must fail before a getter, setter, or runner can affect state.
func TestUnknownBindingIsRejectedBeforeTransfer(t *testing.T) {
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		inputs := program.InputFields{
			program.InputField{
				Field: program.Field{Name: "host"},
				Target: program.SetterFunc(func(string) error {
					t.Fatal("unknown binding reached a setter")
					return nil
				}),
			},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("unknown binding reached the runner")
			return nil
		}), inputs, program.OutputFields(nil)
	})
	source := program.GetterFunc(func() string {
		t.Fatal("unknown binding reached a getter")
		return ""
	})
	_, err := (loader{}).load(factory, map[string]program.Getter{
		"host": source,
		"typo": source,
	})
	if !errors.Is(err, errUnknownInput) {
		t.Fatalf("load error = %v, want unknown input", err)
	}
}

func TestUnboundInputRetainsDefault(t *testing.T) {
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		flags := flag.NewFlagSet("address", flag.ContinueOnError)
		host := flags.String("host", "localhost", "service host")
		scheme := flags.String("scheme", "https", "service URL scheme")
		runner := program.RunnerFunc(func(context.Context) error {
			if *host != "api.example.test" {
				t.Errorf("host = %q, want api.example.test", *host)
			}
			if *scheme != "https" {
				t.Errorf("scheme = %q, want https", *scheme)
			}
			return nil
		})
		return runner, program.FromFlags(flags), program.OutputFields(nil)
	})
	i, err := (loader{}).load(factory, map[string]program.Getter{
		"host": literal("api.example.test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := i.runner.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// Assignment failures retain their cause without exposing sensitive values in
// ordinary diagnostics.
func TestSensitiveAssignmentFailureHidesValueAndPreservesCause(t *testing.T) {
	const secret = "private-token-value"
	rejected := errors.New("credential rejected")
	cause := fmt.Errorf("invalid credential %s: %w", secret, rejected)
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		inputs := program.InputFields{
			program.InputField{
				Field: program.Field{
					Name: "credential", Help: "access credential", Sensitive: true,
				},
				Target: program.SetterFunc(func(string) error { return cause }),
			},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("assignment failure reached the runner")
			return nil
		}), inputs, program.OutputFields(nil)
	})
	var trace bytes.Buffer
	_, err := (loader{trace: &trace}).load(factory, map[string]program.Getter{
		"credential": literal(secret),
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("load error = %v, want credential rejection", err)
	}
	if strings.Contains(trace.String(), secret) || strings.Contains(err.Error(), secret) {
		t.Fatal("sensitive value appeared in ordinary diagnostics")
	}
}
