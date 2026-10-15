package build_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// A generated host consumes a complete module context without changing the
// authoring files. Development substitutions require explicit opt-in, and
// inspection does not transfer values or run programs.
func TestCatalogueHost(t *testing.T) {
	artifact := localArtifact(t)
	options := build.Options{Dev: true}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-modfile=/missing.mod -overlay=/missing.json")
	if _, err := build.Host(t.Context(), artifact, build.Options{}); !errors.Is(err, build.ErrModule) {
		t.Fatalf("production accepted local sources: %v", err)
	}
	result, err := build.Host(t.Context(), artifact, options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := result.Close(); err != nil {
			t.Error(err)
		}
	})
	source, err := os.ReadFile(result.Source)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(source, []byte("private-value-not-in-source")) {
		t.Fatal("literal embedded in generated source")
	}
	mod, err := os.ReadFile(result.ModFile)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(result.SumFile)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := solution.ParseModule(bytes.NewReader(mod), bytes.NewReader(sum))
	if err != nil {
		t.Fatal(err)
	}
	if generated.Path != "example.test/service" {
		t.Fatalf("module identity = %q", generated.Path)
	}
	var authoredSums, generatedSums bytes.Buffer
	if err := artifact.Module.Write(io.Discard, &authoredSums); err != nil {
		t.Fatal(err)
	}
	if err := generated.Write(io.Discard, &generatedSums); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(authoredSums.Bytes(), generatedSums.Bytes()) {
		t.Fatal("build changed authored checksums")
	}
	var data bytes.Buffer
	if err := sef.EncodeGob(&data, artifact); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"check"}, {"inspect"}, {"inspect", "-inputs"}} {
		var output, diagnostics bytes.Buffer
		if err := result.Run(t.Context(), args, bytes.NewReader(data.Bytes()), &output, &diagnostics); err != nil {
			t.Fatalf("%v: %v: %s", args, err, &diagnostics)
		}
		if bytes.Contains(output.Bytes(), []byte("private-value-not-in-source")) {
			t.Fatal("inspection disclosed literal")
		}
		if len(args) == 2 && !strings.Contains(output.String(), `input "name" sensitive=true`) {
			t.Fatalf("declarations absent: %s", &output)
		}
	}
	var diagnostics bytes.Buffer
	err = result.Run(t.Context(), []string{"run"}, nil, nil, &diagnostics)
	if exit, ok := errors.AsType[*exec.ExitError](err); !ok || exit.ExitCode() != 2 {
		t.Fatalf("unsupported command = %v, want exit 2", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := result.Run(ctx, []string{"check"}, nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Run = %v", err)
	}
	work := result.WorkDir
	unrelated := t.TempDir()
	result.WorkDir = unrelated
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary build still present: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("caller directory removed: %v", err)
	}
}

// A compiled host gives the playground cooperative signal handling while
// inspection retains ordinary process termination when its input blocks.
func TestHostShutdown(t *testing.T) {
	artifact := localArtifact(t)
	artifact.Definition.Nodes[0].Definition.Name = "Lifecycle"
	artifact.Definition.Bindings = nil
	result, err := build.Host(t.Context(), artifact, build.Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := result.Close(); err != nil {
			t.Error(err)
		}
	})
	var input bytes.Buffer
	if err := sef.EncodeGob(&input, artifact); err != nil {
		t.Fatal(err)
	}
	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		output := &lifecycleOutput{onStart: cancel}
		var diagnostics bytes.Buffer
		err := result.Run(ctx, []string{"playground"}, bytes.NewReader(input.Bytes()), output, &diagnostics)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled playground = %v: %s", err, &diagnostics)
		}
		if output.String() != "started\nstopped\n" {
			t.Fatalf("application lifecycle = %q", output.String())
		}
	})
	t.Run("SIGTERM", func(t *testing.T) {
		command := exec.CommandContext(t.Context(), result.Binary, "playground")
		output := &lifecycleOutput{onStart: func() {
			if err := command.Process.Signal(syscall.SIGTERM); err != nil {
				t.Error(err)
			}
		}}
		var diagnostics bytes.Buffer
		command.Stdin = bytes.NewReader(input.Bytes())
		command.Stdout, command.Stderr = output, &diagnostics
		err := command.Run()
		if exit, ok := errors.AsType[*exec.ExitError](err); !ok || exit.ExitCode() != 1 {
			t.Fatalf("terminated playground = %v: %s", err, &diagnostics)
		}
		if output.String() != "started\nstopped\n" {
			t.Fatalf("application lifecycle = %q", output.String())
		}
	})
	var incomplete bytes.Buffer
	artifact.Definition.Bindings = []solution.Binding{{Node: "service", Input: "name", Value: solution.Literal(strings.Repeat("x", 4<<20))}}
	if err := sef.EncodeGob(&incomplete, artifact); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"inspect", "playground"} {
		t.Run(operation+"-input", func(t *testing.T) {
			command := exec.CommandContext(t.Context(), result.Binary, operation)
			var diagnostics bytes.Buffer
			command.Stderr = &diagnostics
			stdin, err := command.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = stdin.Close() }()
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = command.Process.Kill() })
			// Exceeding the pipe's capacity makes Write wait for the host's decoder.
			// Leaving the final byte unwritten keeps the command blocked on stdin.
			if _, err := stdin.Write(incomplete.Bytes()[:incomplete.Len()-1]); err != nil {
				t.Fatal(err)
			}
			if err := command.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			err = command.Wait()
			exit, ok := errors.AsType[*exec.ExitError](err)
			if !ok {
				t.Fatalf("terminated %s = %v: %s", operation, err, &diagnostics)
			}
			if operation == "inspect" {
				status, ok := exit.Sys().(syscall.WaitStatus)
				if !ok || status.Signal() != syscall.SIGTERM {
					t.Fatalf("inspect process state = %v, want SIGTERM", exit.ProcessState)
				}
			} else if exit.ExitCode() != 1 {
				t.Fatalf("playground exit status = %v, want 1: %s", exit.ProcessState, &diagnostics)
			}
		})
	}
}

type lifecycleOutput struct {
	buffer  bytes.Buffer
	onStart func()
}

func (w *lifecycleOutput) String() string { return w.buffer.String() }

func (w *lifecycleOutput) Write(data []byte) (int, error) {
	n, err := w.buffer.Write(data)
	if w.onStart != nil && strings.Contains(w.String(), "started\n") {
		w.onStart()
		w.onStart = nil
	}
	return n, err
}

// An output directory belongs to its caller, including when compilation fails.
// Temporary failures are cleaned, and existing files are never overwritten.
func TestOutputOwnership(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("TMPDIR", t.TempDir())
	artifact := localArtifact(t)
	artifact.Definition.Nodes[0].Definition.Package = "example.test/catalogue/internal/private"
	result, err := build.Host(t.Context(), artifact, build.Options{Dev: true})
	if err == nil || result != nil {
		t.Fatalf("failed temporary build = %v, %v", result, err)
	}
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed build left %d directories", len(entries))
	}
	out := filepath.Join(t.TempDir(), "generated")
	options := build.Options{Dev: true, OutDir: out}
	result, err = build.Host(t.Context(), artifact, options)
	if err == nil || result == nil {
		t.Fatalf("retained failure = %v, %v", result, err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.Source); err != nil {
		t.Fatalf("retained source removed: %v", err)
	}
	if _, err := build.Host(t.Context(), artifact, options); !errors.Is(err, build.ErrOutputDir) {
		t.Fatalf("nonempty output directory = %v", err)
	}
}

func TestDeclaredRequirements(t *testing.T) {
	if _, err := build.Host(t.Context(), solution.Artifact{}, build.Options{}); !errors.Is(err, build.ErrModule) {
		t.Fatalf("missing context = %v", err)
	}
	artifact := localArtifact(t)
	artifact.Definition.Nodes[0].Definition.Package = "example.test/undeclared"
	if _, err := build.Host(t.Context(), artifact, build.Options{Dev: true}); !errors.Is(err, build.ErrModule) {
		t.Fatalf("undeclared catalogue = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := build.Host(ctx, artifact, build.Options{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled build = %v", err)
	}
}

// Workspace members substitute declared versions only after development mode is
// selected. Their go.mod files do not become an alternate requirements source.
func TestWorkspaceDevelopment(t *testing.T) {
	artifact := localArtifact(t)
	toolkit := artifact.Module.Replace[0].New.Path
	catalogue := artifact.Module.Replace[1].New.Path
	artifact.Module.Replace = nil
	project := t.TempDir()
	workspace := "go 1.27.0\nuse (\n" + toolkit + "\n" + catalogue + "\n)\n"
	if err := os.WriteFile(filepath.Join(project, "go.work"), []byte(workspace), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", filepath.Join(project, "go.work"))
	t.Setenv("GOPROXY", "off")
	if _, err := build.Host(t.Context(), artifact, build.Options{Dir: project}); err == nil {
		t.Fatal("production used the caller workspace")
	}
	result, err := build.Host(t.Context(), artifact, build.Options{Dev: true, Dir: project})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if len(artifact.Module.Replace) != 0 {
		t.Fatal("development modified artifact module context")
	}
}

func localArtifact(t *testing.T) solution.Artifact {
	t.Helper()
	mod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	module, err := solution.ParseModule(bytes.NewReader(mod), bytes.NewReader(sum))
	if err != nil {
		t.Fatal(err)
	}
	module.Path = "example.test/service"
	module.Require = append(module.Require, solution.Module{Path: build.ToolkitModule, Version: "v0.0.0"}, solution.Module{Path: "example.test/catalogue", Version: "v1.2.3"})
	toolkit, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := filepath.Abs("testdata/catalogue")
	if err != nil {
		t.Fatal(err)
	}
	module.Replace = []solution.ModuleReplacement{
		{Old: solution.Module{Path: build.ToolkitModule}, New: solution.Module{Path: toolkit}},
		{Old: solution.Module{Path: "example.test/catalogue"}, New: solution.Module{Path: catalogue}},
	}
	return solution.Artifact{Version: "0.1", Module: module, Definition: solution.Definition{
		Nodes:    []solution.Node{{Name: "service", Definition: solution.Symbol{Package: "example.test/catalogue", Name: "Service"}}},
		Bindings: []solution.Binding{{Node: "service", Input: "name", Value: solution.Literal("private-value-not-in-source")}},
	}}
}
