// Package build compiles generated descriptor hosts in isolated Go modules.
// Artifact requirements select catalogue dependencies through normal Go module
// version selection. Caller modules, workspaces, and GOFLAGS are not inherited.
// Dependency downloads and generated manifest changes stay in the private build
// directory and the Go module cache.
package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/gen"
)

// ToolkitModule is the module supplying the generated host protocol.
const ToolkitModule = "github.com/modern-engineering/prototype"

// ErrToolkitVersion means neither the artifact nor the build options select a
// toolkit version or local toolkit source.
var ErrToolkitVersion = errors.New("host requires a toolkit version or local source")

// ErrModule means requirements or replacements cannot describe the host module.
var ErrModule = errors.New("invalid host module configuration")

// Replacement explicitly substitutes a local directory for a module. Relative
// paths are resolved against the current directory before building. Replacements
// do not declare requirements and cannot supply undeclared catalogue modules.
type Replacement struct {
	Module string
	Dir    string
}

// Options supplies the host dependency and explicit local source substitutions.
// Artifact requirements take precedence over ToolkitVersion. ToolkitDir is an
// explicit replacement for toolkit source, including an artifact-selected version;
// when no version is declared, that replacement uses v0.0.0.
// KeepWork preserves generated files after a failed build as well as success.
// Successful results always belong to the caller until Result.Close.
type Options struct {
	ToolkitDir     string
	ToolkitVersion string
	Replacements   []Replacement
	KeepWork       bool
}

// Result identifies one private generated work directory and its files. Close
// removes that original directory even if these display fields are changed.
// A kept failure may have source and manifests without a compiled binary.
type Result struct {
	WorkDir string
	Source  string
	ModFile string
	Binary  string
	owned   string
}

// Close removes only this build's private work directory. It is safe to repeat.
func (r *Result) Close() error { return os.RemoveAll(r.owned) }

// Run executes the compiled host with caller-supplied arguments and streams.
// It preserves *exec.ExitError for the host's status and context cancellation.
// It does not delete the result, infer commands, or invoke any program runners.
func (r *Result) Run(ctx context.Context, args []string, input io.Reader, output, diagnostics io.Writer) error {
	command := exec.CommandContext(ctx, r.Binary, args...)
	command.Dir = r.owned
	command.Stdin, command.Stdout, command.Stderr = input, output, diagnostics
	err := command.Run()
	if err != nil && ctx.Err() != nil {
		return errors.Join(err, ctx.Err())
	}
	return err
}

// Error distinguishes generation, module preparation, and compilation failures.
// Output retains Go tool diagnostics; Err preserves the original cause.
type Error struct {
	Stage  string
	Output string
	Err    error
}

func (e *Error) Error() string {
	message := fmt.Sprintf("host %s: %v", e.Stage, e.Err)
	if e.Output != "" {
		message += "\n" + strings.TrimSpace(e.Output)
	}
	return message
}

// Unwrap returns the operation cause.
func (e *Error) Unwrap() error { return e.Err }

// Host generates and compiles a descriptor host without running it. Its private
// module directly requires definition.Modules and the host toolkit. Dependencies
// use ordinary Go minimal version selection; declared versions are requirements,
// not a lockfile. Missing catalogue packages cannot trigger module upgrades or
// discovery outside that selected graph. Each catalogue import must lie within a declared module or the
// toolkit. Local replacements are explicit, and inaccessible internal packages
// fail according to normal Go visibility rules.
//
// Go commands use GOWORK=off, empty GOFLAGS, and GOTOOLCHAIN=local. Module downloads
// use the caller's proxy and checksum configuration. The generated go.mod/go.sum
// may change during dependency resolution; source module manifests are untouched.
// Authored literal values are never embedded in generated source.
//
// Successful results belong to the caller, which must Close them unless keeping
// the work for inspection. Failures clean temporary files unless KeepWork is set;
// a retained failure returns both a Result and an Error. Compilation and host
// execution are separate operations.
func Host(ctx context.Context, definition solution.Definition, options Options) (*Result, error) {
	source, err := gen.Main(definition)
	if err != nil {
		return nil, &Error{Stage: "generation", Err: err}
	}
	requirements, replacements, err := moduleArguments(ctx, definition, options)
	if err != nil {
		return nil, &Error{Stage: "module", Err: err}
	}
	work, err := os.MkdirTemp("", "sef-host-")
	if err != nil {
		return nil, &Error{Stage: "generation", Err: err}
	}
	result := &Result{WorkDir: work, Source: filepath.Join(work, "main.go"), ModFile: filepath.Join(work, "go.mod"), Binary: filepath.Join(work, "sef-host"), owned: work}
	if runtime.GOOS == "windows" {
		result.Binary += ".exe"
	}
	fail := func(err error) (*Result, error) {
		if options.KeepWork {
			return result, err
		}
		return nil, errors.Join(err, result.Close())
	}
	if err := os.WriteFile(result.Source, source, 0o600); err != nil {
		return fail(&Error{Stage: "generation", Err: err})
	}
	if err := os.WriteFile(result.ModFile, []byte("module example.invalid/sefhost\n\ngo 1.27.0\n"), 0o600); err != nil {
		return fail(&Error{Stage: "module", Err: err})
	}
	environment := buildEnvironment()
	args := append([]string{"mod", "edit"}, requirements...)
	args = append(args, replacements...)
	var selected map[string]string
	for _, operation := range []struct {
		stage string
		args  []string
	}{
		{"module", args},
		{"dependencies", []string{"mod", "tidy"}},
		{"compilation", []string{"build", "-mod=readonly", "-o", result.Binary, "."}},
	} {
		command := exec.CommandContext(ctx, "go", operation.args...)
		command.Dir, command.Env = work, environment
		output, err := command.CombinedOutput()
		if err != nil {
			return fail(commandError(ctx, operation.stage, err, output))
		}
		if operation.stage == "module" {
			selected, err = selectedModules(ctx, work, environment)
			if err != nil {
				return fail(err)
			}
		}
		if operation.stage == "dependencies" {
			if err := checkCatalogueModules(ctx, work, environment, definition, selected); err != nil {
				return fail(err)
			}
			if err := retainRequirements(ctx, work, environment, requirements); err != nil {
				return fail(err)
			}
		}
	}
	return result, nil
}

func buildEnvironment() []string {
	environment := os.Environ()
	return append(environment, "GOWORK=off", "GOFLAGS=", "GO111MODULE=on", "GOTOOLCHAIN=local", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
}

func commandError(ctx context.Context, stage string, err error, output []byte) error {
	if ctx.Err() != nil {
		err = errors.Join(err, ctx.Err())
	}
	if len(output) == 0 {
		if exited, ok := errors.AsType[*exec.ExitError](err); ok {
			output = exited.Stderr
		}
	}
	return &Error{Stage: stage, Output: string(output), Err: err}
}
