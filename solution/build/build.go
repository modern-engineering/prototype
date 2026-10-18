// Package build compiles generated descriptor hosts using artifact-owned Go
// module contexts. Production builds ignore caller workspaces and GOFLAGS and
// use the supplied requirements and checksums without editing authoring files.
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

// ToolkitModule supplies the generated host protocol. It must be an ordinary
// requirement in the artifact's module context.
const ToolkitModule = "github.com/modern-engineering/prototype"

// ErrModule means a module context cannot supply the generated host.
var ErrModule = errors.New("invalid host module configuration")

// ErrOutputDir means an output directory already contains files or is not a
// directory. Builds never overwrite an existing directory's contents.
var ErrOutputDir = errors.New("host output directory must be empty")

// Options controls local development and generated-file retention.
// Dir is the authoring directory, defaulting to the current directory. Dev opts
// into local replacements and modules from that directory's Go workspace and
// nearest enclosing go.mod. These substitutions never declare requirements.
// OutDir retains generated files, including after a failed build. It must be
// empty or absent. An empty OutDir uses a temporary directory instead.
// Production is the zero value: local replacements are rejected and caller
// workspaces have no effect.
type Options struct {
	Dev    bool
	Dir    string
	OutDir string
}

// Result identifies the generated source, module files, and executable.
// Close removes temporary builds; files requested through Options.OutDir belong
// to the caller and are retained. A failed retained build may lack a binary.
type Result struct {
	WorkDir string
	Source  string
	ModFile string
	SumFile string
	Binary  string
	dir     string
	owned   bool
}

// Close removes only the original temporary build directory. It is safe to
// repeat and does nothing for an explicitly retained output directory.
func (r *Result) Close() error {
	if !r.owned {
		return nil
	}
	return os.RemoveAll(r.dir)
}

// Run executes the compiled host with caller-supplied arguments and streams.
// Playground invokes program runners and forwards context cancellation as an
// interrupt, waiting for cooperative shutdown. Other commands terminate
// immediately on cancellation. Run preserves *exec.ExitError and context
// cancellation and does not remove generated files.
func (r *Result) Run(ctx context.Context, args []string, input io.Reader, output, diagnostics io.Writer) error {
	command := exec.CommandContext(ctx, r.Binary, args...)
	if len(args) > 0 && args[0] == "playground" {
		command.Cancel = func() error { return command.Process.Signal(os.Interrupt) }
	}
	command.Dir = r.dir
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

// Host generates and compiles a descriptor host without running it. The
// artifact's module context is authoritative, including its module identity,
// requirements, versioned replacements, exclusions, and checksums. Each catalogue
// import and the toolkit must be covered by a declared requirement. Package
// discovery cannot select additional modules or upgrade their versions.
//
// Go commands use GOWORK=off, empty GOFLAGS, and GOTOOLCHAIN=local. Dev translates
// local workspace modules into replacements in the generated manifest only.
// Production requires a complete module context; use Tidy while authoring to
// prepare requirements and checksums. Downloads use the caller's proxy and
// checksum configuration. Authored literals are never embedded in source.
//
// Successful temporary results must be closed. Failed temporary builds are
// removed. With OutDir, failures return both a Result and an Error, preserving
// the available files for inspection.
func Host(ctx context.Context, artifact solution.Artifact, options Options) (*Result, error) {
	result, module, err := prepare(ctx, artifact, options)
	if err != nil {
		return result, err
	}
	fail := func(err error) (*Result, error) { return failed(result, err) }
	environment := buildEnvironment()
	selected, err := selectedModules(ctx, result.dir, environment)
	if err != nil {
		return fail(err)
	}
	if err := checkCatalogueModules(ctx, result.dir, environment, artifact.Definition, module, selected); err != nil {
		return fail(err)
	}
	command := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", result.Binary, ".")
	command.Dir, command.Env = result.dir, environment
	output, err := command.CombinedOutput()
	if err != nil {
		return fail(commandError(ctx, "compilation", err, output))
	}
	return result, nil
}

func prepare(ctx context.Context, artifact solution.Artifact, options Options) (*Result, solution.ModuleContext, error) {
	source, err := gen.Main(artifact.Definition)
	if err != nil {
		return nil, solution.ModuleContext{}, &Error{Stage: "generation", Err: err}
	}
	module, err := hostModule(ctx, artifact, options)
	if err != nil {
		return nil, solution.ModuleContext{}, &Error{Stage: "module", Err: err}
	}
	result, err := newResult(options.OutDir)
	if err != nil {
		return nil, solution.ModuleContext{}, &Error{Stage: "generation", Err: err}
	}
	fail := func(err error) (*Result, solution.ModuleContext, error) {
		result, err := failed(result, err)
		return result, solution.ModuleContext{}, err
	}
	if err := os.WriteFile(result.Source, source, 0o600); err != nil {
		return fail(&Error{Stage: "generation", Err: err})
	}
	if err := writeModule(result, module); err != nil {
		return fail(&Error{Stage: "module", Err: err})
	}
	return result, module, nil
}

func newResult(outDir string) (*Result, error) {
	var dir string
	var err error
	if outDir == "" {
		dir, err = os.MkdirTemp("", "sef-host-")
	} else {
		dir, err = filepath.Abs(outDir)
		if err == nil {
			entries, readErr := os.ReadDir(dir)
			switch {
			case errors.Is(readErr, os.ErrNotExist):
				err = os.MkdirAll(dir, 0o700)
			case readErr != nil:
				err = errors.Join(ErrOutputDir, readErr)
			case len(entries) != 0:
				err = ErrOutputDir
			}
		}
	}
	if err != nil {
		return nil, err
	}
	result := &Result{WorkDir: dir, Source: filepath.Join(dir, "main.go"), ModFile: filepath.Join(dir, "go.mod"), SumFile: filepath.Join(dir, "go.sum"), Binary: filepath.Join(dir, "sef-host"), dir: dir, owned: outDir == ""}
	if runtime.GOOS == "windows" {
		result.Binary += ".exe"
	}
	return result, nil
}

func failed(result *Result, err error) (*Result, error) {
	if !result.owned {
		return result, err
	}
	return nil, errors.Join(err, result.Close())
}

func buildEnvironment() []string {
	return append(os.Environ(), "GOWORK=off", "GOFLAGS=", "GO111MODULE=on", "GOTOOLCHAIN=local", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
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
