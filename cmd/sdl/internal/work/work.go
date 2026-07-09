// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package work drives the back half of sdl build: it lays the generated
// compiler down in a temporary work directory, synthesizes a module
// context that resolves packages exactly as the go CLI would in the
// solution directory, builds the compiler, and runs it.
//
// # Module context
//
// Detect classifies the solution directory the way the go CLI itself
// would: an active go.work selects workspace mode, otherwise an
// enclosing go.mod selects module mode.
//
// In module mode the temporary module mirrors the solution module's
// resolution state from one 'go list -m' run: every module of the graph
// is required at its resolved version, every replacement is mirrored —
// module replacements at their replacement path and version, directory
// replacements with the directory rebased to an absolute path — the
// user's main module is required at a synthetic version backed by a
// directory replacement, and the user's go.sum is copied verbatim. The
// build then runs with the default -mod=readonly: if it still wants to
// update go.mod, that is a bug in this synthesis, never something to
// paper over with -mod=mod.
//
// In workspace mode the work directory instead joins a synthesized copy
// of the user's workspace: a go.work that use's every directory of the
// user's go.work plus the work directory itself, the user's replace
// directives mirrored and go.work.sum copied verbatim. The union build
// list then resolves the generated compiler's imports — local member
// source included — with no requirements synthesized at all, because a
// workspace resolves any member's imports through the union graph.
//
// Solutions outside any module context arrive at a later rung.
package work

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// prototypePath is this framework's own module path: the generated
// compiler imports its solution package, and the version-skew handshake
// compares the solution's resolved copy against the running CLI's.
const prototypePath = "github.com/modern-engineering/prototype"

// A Config carries one driver invocation.
type Config struct {
	// Dir is the absolute solution directory; it anchors the module
	// context every go invocation resolves in.
	Dir string

	// Context is the module context Detect reported for Dir; it selects
	// the synthesis strategy. Required.
	Context *Context

	// Source is the generated compiler's Go source, written to
	// solmain.go in the work directory.
	Source []byte

	// Output is the image destination; empty means the inherited
	// stdout.
	Output string

	// Keep preserves the work directory and prints WORK=<dir> on
	// Stderr (the cmd/go -work precedent).
	Keep bool

	// Stderr receives warnings and the relayed stderr of the child
	// processes; nil means os.Stderr.
	Stderr io.Writer
}

// Run executes the driver: synthesize the work directory's module
// context, build the generated compiler, run it, and relay its verdict.
// Solution faults come back as *base.DiagnosticsError (exit code 1,
// matching the compiler's own exit 1); everything else — including a
// compiler that exits 2 — is an ordinary error (exit code 2).
func Run(ctx context.Context, cfg Config) error {
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	if cfg.Context.Mode() == ModeWorkspace {
		return runWorkspace(ctx, cfg, stderr)
	}
	return runModule(ctx, cfg, stderr)
}

// runModule drives a build inside an enclosing module: the work
// directory carries a synthesized go.mod mirroring the module's whole
// resolution state.
func runModule(ctx context.Context, cfg Config, stderr io.Writer) error {
	env := cfg.Context.Env()
	gomod := cfg.Context.gomod

	graph, err := listModuleGraph(ctx, cfg.Dir, filepath.Dir(gomod), env)
	if err != nil {
		return err
	}
	if diag := checkPrototype(graph, gomod, stderr); diag != nil {
		return diag
	}
	goVersion, toolchain, err := modDirectives(ctx, cfg.Dir, env)
	if err != nil {
		return err
	}

	workdir, cleanup, err := makeWorkdir(cfg.Keep, stderr)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := os.WriteFile(filepath.Join(workdir, "solmain.go"), cfg.Source, 0o666); err != nil {
		return fmt.Errorf("writing generated compiler: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "go.mod"), synthesizeGoMod(graph, goVersion, toolchain), 0o666); err != nil {
		return fmt.Errorf("writing synthesized go.mod: %v", err)
	}
	if err := copyGoSum(filepath.Dir(gomod), workdir); err != nil {
		return err
	}

	if err := buildCompiler(ctx, workdir, env, stderr); err != nil {
		return err
	}
	return runCompiler(ctx, workdir, cfg.Output, stderr)
}

// makeWorkdir creates the temporary build directory and hands back its
// cleanup. Keep mode announces the directory instead (the cmd/go -work
// precedent) and the cleanup keeps its hands off.
//
// The directory is handed out with its symlinks resolved: the system
// temp directory is a symlink on darwin, and a child go process
// resolves its working directory (no PWD is exported to vouch for the
// alias), so an unresolved path would fail the go.work membership
// check of workspace mode — the go CLI compares the two as strings.
func makeWorkdir(keep bool, stderr io.Writer) (workdir string, cleanup func(), err error) {
	workdir, err = os.MkdirTemp("", "sdl-build-")
	if err != nil {
		return "", nil, fmt.Errorf("creating work directory: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(workdir); err == nil {
		workdir = resolved
	}
	if keep {
		printf(stderr, "WORK=%s\n", workdir)
		return workdir, func() {}, nil
	}
	return workdir, func() {
		if err := os.RemoveAll(workdir); err != nil {
			printf(stderr, "sdl: removing work directory: %v\n", err)
		}
	}, nil
}

// A module is one row of the solution's module graph: the module path,
// its resolved version (empty for the main module), and its replacement
// when it has one — replPath and replVersion for a module replacement
// (replace old => new vX.Y.Z), dir for a directory replacement.
type module struct {
	path    string
	version string

	replPath    string // module replacement: the replacement module path
	replVersion string // module replacement: the replacement version
	dir         string // directory replacement target; the main module's root
}

// A moduleGraph is the parsed 'go list -m all' state the temporary
// module mirrors.
type moduleGraph struct {
	main module   // dir holds the main module's root directory
	deps []module // remaining graph modules, in go list order
}

// moduleRowFormat is the go list -m template rendering one module row:
// "path version" — the version empty on a main module — with each
// replacement behind an explicit discriminator, "=> mod path version"
// or "=> dir directory", so the two replacement kinds parse
// unambiguously even though a directory may contain spaces. A module
// replacement is identified by its non-empty Replace.Version; its
// Replace.Dir (filled once the module cache holds the replacement,
// empty until then) is never asked for, because the module identity is
// the faithful mirror either way.
const moduleRowFormat = "{{.Path}} {{.Version}}{{with .Replace}}{{if .Version}} => mod {{.Path}} {{.Version}}{{else}} => dir {{.Dir}}{{end}}{{end}}"

// listModuleGraph captures the module graph with one go list run — one
// consistent snapshot for requirements, replacements, and the skew
// handshake alike.
func listModuleGraph(ctx context.Context, dir, mainDir string, env []string) (*moduleGraph, error) {
	out, err := goOutput(ctx, dir, env, "list", "-m", "-f", moduleRowFormat, "all")
	if err != nil {
		return nil, err
	}
	return parseModuleGraph(out, mainDir)
}

// parseModuleGraph reads the go list template output, one moduleRowFormat
// row per line, into the main module (the only version-less,
// replacement-less row) and its dependency modules.
func parseModuleGraph(out, mainDir string) (*moduleGraph, error) {
	g := &moduleGraph{}
	for line := range strings.Lines(out) {
		line = strings.TrimRight(line, " \n")
		if line == "" {
			continue
		}
		m, err := parseModuleRow(line, mainDir)
		if err != nil {
			return nil, err
		}
		if m.version == "" {
			if g.main.path != "" {
				return nil, fmt.Errorf("module graph lists two main modules: %s and %s", g.main.path, m.path)
			}
			g.main = module{path: m.path, dir: mainDir}
			continue
		}
		g.deps = append(g.deps, m)
	}
	if g.main.path == "" {
		return nil, errors.New("module graph lists no main module")
	}
	return g, nil
}

// parseModuleRow reads one moduleRowFormat row. The replacement
// directory is the final field, so it may contain spaces; a relative
// one is rebased onto base, where the file declaring the replacement
// resolved it.
func parseModuleRow(line, base string) (module, error) {
	// Module paths and versions never contain spaces; SplitN keeps
	// the directory field intact.
	parts := strings.SplitN(line, " ", 5)
	switch {
	case len(parts) == 1:
		return module{path: parts[0]}, nil
	case len(parts) == 2:
		return module{path: parts[0], version: parts[1]}, nil
	case len(parts) == 5 && parts[2] == "=>" && parts[3] == "mod":
		rpath, rversion, ok := strings.Cut(parts[4], " ")
		if !ok {
			break
		}
		return module{path: parts[0], version: parts[1], replPath: rpath, replVersion: rversion}, nil
	case len(parts) == 5 && parts[2] == "=>" && parts[3] == "dir":
		return module{path: parts[0], version: parts[1], dir: rebase(parts[4], base)}, nil
	}
	return module{}, fmt.Errorf("malformed module graph line %q", line)
}

// checkPrototype guards the generated compiler's own dependency: the
// prototype module must be reachable in the solution's graph, and when
// it resolves to an ordinary module version that version is compared
// against the running CLI's build info — a mismatch warns (never
// fatal), because the compiled semantics come from the solution's copy,
// not from the binary the user invoked.
func checkPrototype(g *moduleGraph, gomod string, stderr io.Writer) *base.DiagnosticsError {
	return checkPrototypeVersion(g, gomod, cliVersion(), stderr)
}

// cliVersion is the running CLI's own prototype-module version, empty
// when the binary does not know it.
func cliVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Path == prototypePath {
		return info.Main.Version
	}
	return ""
}

// checkPrototypeVersion is checkPrototype behind its build-info read.
func checkPrototypeVersion(g *moduleGraph, gomod, cliVersion string, stderr io.Writer) *base.DiagnosticsError {
	if g.main.path == prototypePath {
		return nil // the examples case: the main-module replace covers everything
	}
	for _, dep := range g.deps {
		if dep.path != prototypePath {
			continue
		}
		comparePrototype(dep, cliVersion, stderr)
		return nil
	}
	return &base.DiagnosticsError{Lines: []string{
		fmt.Sprintf("module %s does not require %s, which the generated solution compiler imports; add it to %s", g.main.path, prototypePath, gomod),
	}}
}

// comparePrototype warns when the prototype version a build will
// resolve differs from the CLI's own; cliVersion is empty when the
// running binary does not know its version. A locally sourced prototype
// — a directory replacement, or a version-less row naming a main or
// workspace module — never warns: the graph then pins no meaningful
// version to compare, and that is the everyday dev-loop shape, where
// the solution deliberately tracks local source.
func comparePrototype(m module, cliVersion string, stderr io.Writer) {
	if m.dir != "" || m.version == "" {
		return // local source: no version to compare
	}
	version := m.version
	if m.replVersion != "" {
		version = m.replVersion // the copy the build will actually resolve
	}
	if cliVersion != "" && cliVersion != version {
		printf(stderr, "sdl: warning: solution resolves %s %s but this sdl binary was built from %s; align them if results surprise\n",
			prototypePath, version, cliVersion)
	}
}

// modDirectives reads the go and toolchain directives of the module
// governing dir, via the go tool's own parser (go mod edit -json), so
// the synthesized module compiles under the same language version and
// toolchain selection rules.
func modDirectives(ctx context.Context, dir string, env []string) (goVersion, toolchain string, err error) {
	out, err := goOutput(ctx, dir, env, "mod", "edit", "-json")
	if err != nil {
		return "", "", err
	}
	var mod struct {
		Go        string
		Toolchain string
	}
	if err := json.Unmarshal([]byte(out), &mod); err != nil {
		return "", "", fmt.Errorf("reading module directives: %v", err)
	}
	return mod.Go, mod.Toolchain, nil
}

// synthesizeGoMod renders the temporary module: its own throwaway
// identity, the copied language directives, one require per graph
// module at its resolved version — the main module at a synthetic
// v0.0.0-solution — and one replace per replaced module plus the main
// module's directory replacement, so the temporary module resolves
// exactly what the solution's module does.
func synthesizeGoMod(g *moduleGraph, goVersion, toolchain string) []byte {
	var b bytes.Buffer
	b.WriteString("module sdl.invalid/solmain\n")
	if goVersion != "" {
		fmt.Fprintf(&b, "\ngo %s\n", goVersion)
	}
	if toolchain != "" {
		fmt.Fprintf(&b, "\ntoolchain %s\n", toolchain)
	}
	b.WriteString("\nrequire (\n")
	fmt.Fprintf(&b, "\t%s v0.0.0-solution\n", g.main.path)
	for _, dep := range g.deps {
		fmt.Fprintf(&b, "\t%s %s\n", dep.path, dep.version)
	}
	b.WriteString(")\n")
	fmt.Fprintf(&b, "\nreplace %s => %s\n", g.main.path, quoteDir(g.main.dir))
	for _, dep := range g.deps {
		switch {
		case dep.replVersion != "":
			// A module replacement mirrors as itself, never as the
			// module cache directory backing it: a directory replace
			// would bypass the go.sum verification the user's own
			// build performs, and the cache need not hold the
			// replacement at all yet.
			fmt.Fprintf(&b, "\nreplace %s => %s %s\n", dep.path, dep.replPath, dep.replVersion)
		case dep.dir != "":
			fmt.Fprintf(&b, "\nreplace %s => %s\n", dep.path, quoteDir(dep.dir))
		}
	}
	return b.Bytes()
}

// quoteDir renders a replacement directory as a go.mod token, quoting
// it when it would not lex as one bare word.
func quoteDir(dir string) string {
	if strings.ContainsAny(dir, " \t\"'`") {
		return strconv.Quote(dir)
	}
	return dir
}

// copyGoSum copies the solution module's go.sum into the work module,
// when there is one: the synthesized requirements are exactly the
// user's resolved graph, so the user's sums verify it.
func copyGoSum(moduleDir, workdir string) error {
	sum, err := os.ReadFile(filepath.Join(moduleDir, "go.sum"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading go.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "go.sum"), sum, 0o666); err != nil {
		return fmt.Errorf("writing go.sum: %v", err)
	}
	return nil
}

// buildCompiler compiles the generated program. Failures relay the Go
// toolchain's output behind one orienting preface; a failure here means
// a catalogue package (or sdl's own synthesis — a bug) does not
// compile, and either way the Go errors are the diagnosis.
func buildCompiler(ctx context.Context, workdir string, env []string, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", "sol.bin", ".")
	cmd.Dir = workdir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		printf(stderr, "sdl: catalogue package failed to compile; Go errors follow\n")
		printf(stderr, "%s", out.Bytes())
		return fmt.Errorf("building the solution compiler: %v", err)
	}
	return nil
}

// runCompiler executes the generated compiler and propagates its
// verdict: stdout is the image (routed to the -o file when set), stderr
// relays verbatim, exit 1 stays a diagnostics failure, everything else
// nonzero stays an internal one. A failed run — a failed close of the
// output file included — removes the output file: the compiler emits
// the image only after all checks pass, so a partial file only
// misleads.
func runCompiler(ctx context.Context, workdir, output string, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, filepath.Join(workdir, "sol.bin"))
	cmd.Dir = workdir
	cmd.Stderr = stderr
	cmd.Stdout = os.Stdout
	var f *os.File
	if output != "" {
		var err error
		f, err = os.Create(output)
		if err != nil {
			return fmt.Errorf("creating output file: %v", err)
		}
		cmd.Stdout = f
	}
	err := cmd.Run()
	if f != nil {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("writing output file: %v", cerr)
		}
	}
	if err == nil {
		return nil
	}
	if output != "" {
		if rerr := os.Remove(output); rerr != nil {
			printf(stderr, "sdl: removing incomplete output: %v\n", rerr)
		}
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return &base.DiagnosticsError{} // diagnostics already on stderr
	}
	return fmt.Errorf("solution compiler: %v", err)
}

// printf writes one warning or relay line to the driver's stderr. The
// write is best effort: a stderr that fails has nowhere better to hear
// about it, and the returned error already carries the verdict, so the
// write error is deliberately discarded — here, once, rather than at
// every call site.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// goOutput runs one go command in dir and returns its stdout, folding a
// failure's stderr into the error.
func goOutput(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(errb.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("go %s: %s", strings.Join(args, " "), detail)
	}
	return out.String(), nil
}
