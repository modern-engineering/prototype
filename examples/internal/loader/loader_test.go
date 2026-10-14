package loader_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/program"
)

// An undeclared input must fail before the loader tries to acquire its file.
func TestUnknownInputIsRejectedBeforeOpeningFile(t *testing.T) {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		inputs := program.InputList{
			program.Input{
				Field: program.Field{Name: "host"},
				Value: program.SetterFunc(func(string) error {
					t.Fatal("unknown input reached a setter")
					return nil
				}),
			},
		}
		return program.RunnerFunc(func(context.Context) error {
			t.Fatal("unknown input reached the runner")
			return nil
		}), inputs, program.OutputList(nil)
	})
	i := (loader.Loader{}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	err := i.ReadFile(t.Context(), "typo", filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, loader.ErrUnknownInput) {
		t.Fatalf("file assignment error = %v, want unknown input", err)
	}
	if _, err := i.Input("typo"); !errors.Is(err, loader.ErrUnknownInput) {
		t.Fatalf("input lookup = %v, want unknown input", err)
	}
}

func TestUnboundInputRetainsDefault(t *testing.T) {
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
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
	i := (loader.Loader{}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	host, err := i.Input("host")
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Set("api.example.test"); err != nil {
		t.Fatal(err)
	}
	if err := i.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// Diagnostics require an explicitly supplied logger, and assignment failures
// retain their cause without exposing sensitive values in those diagnostics.
func TestSensitiveFileAssignmentHidesValueAndPreservesCause(t *testing.T) {
	var diagnostics bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&diagnostics, &slog.HandlerOptions{Level: slog.LevelDebug}))
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })
	const secret = "private-token-value"
	const path = "testdata/credential.txt"
	rejected := errors.New("credential rejected")
	cause := fmt.Errorf("invalid credential %s: %w", secret, rejected)
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
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
	i := (loader.Loader{}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	err := i.ReadFile(t.Context(), "credential", path)
	if !errors.Is(err, rejected) {
		t.Fatalf("file assignment error = %v, want credential rejection", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("sensitive value appeared in ordinary diagnostics")
	}
	if diagnostics.Len() != 0 {
		t.Fatal("zero-value loader emitted diagnostics through the default logger")
	}

	i = (loader.Loader{Logger: logger}).Make(descriptor)
	defer func(invocation *loader.Invocation) {
		if err := invocation.Close(); err != nil {
			t.Error(err)
		}
	}(i)
	err = i.ReadFile(t.Context(), "credential", path)
	if !errors.Is(err, rejected) {
		t.Fatalf("file assignment with logging error = %v, want credential rejection", err)
	}
	if diagnostics.Len() == 0 {
		t.Fatal("explicit logger received no diagnostics")
	}
	if strings.Contains(diagnostics.String(), secret) || strings.Contains(err.Error(), secret) {
		t.Fatal("sensitive value appeared in ordinary diagnostics")
	}
}
