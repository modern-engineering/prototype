package loader_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// Unknown bindings must fail before a getter, setter, or runner can affect state.
func TestUnknownBindingIsRejectedBeforeTransfer(t *testing.T) {
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		inputs := program.InputList{
			program.Input{
				Field: program.Field{Name: "host"},
				Value: program.SetterFunc(func(string) error {
					t.Fatal("unknown binding reached a setter")
					return nil
				}),
			},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("unknown binding reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	source := program.GetterFunc(func() string {
		t.Fatal("unknown binding reached a getter")
		return ""
	})
	_, err := (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{
		"host": source,
		"typo": source,
	})
	if !errors.Is(err, loader.ErrUnknownInput) {
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
		return runner, program.FromFlags(flags), program.OutputList(nil)
	})
	i, err := (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{
		"host": literal("api.example.test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// Diagnostics require an explicitly supplied logger, and assignment failures
// retain their cause without exposing sensitive values in those diagnostics.
func TestSensitiveAssignmentFailureHidesValueAndPreservesCause(t *testing.T) {
	var diagnostics bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&diagnostics, &slog.HandlerOptions{Level: slog.LevelDebug}))
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })
	const secret = "private-token-value"
	rejected := errors.New("credential rejected")
	cause := fmt.Errorf("invalid credential %s: %w", secret, rejected)
	factory := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		inputs := program.InputList{
			program.Input{
				Field: program.Field{
					Name: "credential", Help: "access credential", Sensitive: true,
				},
				Value: program.SetterFunc(func(string) error { return cause }),
			},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("assignment failure reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	_, err := (loader.Loader{}).Load(t.Context(), factory, map[string]program.Getter{
		"credential": literal(secret),
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("load error = %v, want credential rejection", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("sensitive value appeared in ordinary diagnostics")
	}
	if diagnostics.Len() != 0 {
		t.Fatal("zero-value loader emitted diagnostics through the default logger")
	}

	_, err = (loader.Loader{Logger: logger}).Load(t.Context(), factory, map[string]program.Getter{
		"credential": literal(secret),
	})
	if !errors.Is(err, rejected) {
		t.Fatalf("load with logging error = %v, want credential rejection", err)
	}
	if diagnostics.Len() == 0 {
		t.Fatal("explicit logger received no diagnostics")
	}
	if strings.Contains(diagnostics.String(), secret) || strings.Contains(err.Error(), secret) {
		t.Fatal("sensitive value appeared in ordinary diagnostics")
	}
}

type literal string

func (value literal) Get() string { return string(value) }
