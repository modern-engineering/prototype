// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// solutionPath is the package the generated compiler itself imports;
// the resolution probe must therefore resolve it alongside the
// solution's own imports.
const solutionPath = prototypePath + "/solution"

// A ModulelessConfig carries the resolution half of a build outside any
// module context.
type ModulelessConfig struct {
	// Context is the module context Detect reported for the solution
	// directory — ModeNone, or ResolveModuleless has no business
	// running. Required.
	Context *Context

	// Imports are the solution's imported package paths; the work
	// module's go.mod comes to require the modules providing them.
	Imports []string

	// Keep and Stderr as in Config.
	Keep   bool
	Stderr io.Writer
}

// A Moduleless is a resolved module-less build: a work module whose
// go.mod carries the solution's imports as ordinary requirements,
// ready to root discovery and to compile the generated program.
type Moduleless struct {
	// Dir is the work directory: the one module context in which the
	// solution's import paths mean anything, so discovery roots here.
	Dir string

	env     []string
	stderr  io.Writer
	cleanup func()
}

// ResolveModuleless begins a build outside any module context. The
// pipeline's usual order inverts here: everywhere else discovery runs
// before the driver, in a module context that exists independently of
// the build, but with no enclosing module the work module itself is
// that context, so its requirements must be resolved — 'go mod tidy'
// over a probe program carrying the solution's imports, each landing at
// its latest version through the ambient GOPROXY and module cache
// configuration, network included — before discovery has anywhere to
// run. Imports the tidy could not resolve are left for discovery to
// report as positioned diagnostics, the same messages every other mode
// produces.
func ResolveModuleless(ctx context.Context, cfg ModulelessConfig) (*Moduleless, error) {
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	env := cfg.Context.Env()

	goVersion, err := goEnvVersion(ctx, env)
	if err != nil {
		return nil, err
	}

	workdir, cleanup, err := makeWorkdir(cfg.Keep, stderr)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Moduleless, error) {
		cleanup()
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(workdir, "go.mod"), scaffoldGoMod(scaffoldGoVersion(goVersion)), 0o666); err != nil {
		return fail(fmt.Errorf("writing scaffold go.mod: %v", err))
	}
	if err := os.WriteFile(filepath.Join(workdir, "solmain.go"), probeSource(cfg.Imports), 0o666); err != nil {
		return fail(fmt.Errorf("writing resolution probe: %v", err))
	}
	if err := tidy(ctx, workdir, env, stderr); err != nil {
		return fail(err)
	}
	if err := checkModulelessPrototype(ctx, workdir, env, stderr); err != nil {
		return fail(err)
	}
	return &Moduleless{Dir: workdir, env: env, stderr: stderr, cleanup: cleanup}, nil
}

// Env is the module-context environment for rooting discovery in Dir.
func (m *Moduleless) Env() []string { return m.env }

// Close releases the work directory — unless Keep held it back.
func (m *Moduleless) Close() { m.cleanup() }

// BuildAndRun finishes the build the way Run does for the module
// modes: the generated compiler replaces the resolution probe — its
// imports are a subset of the probe's, so the tidied go.mod already
// satisfies the default -mod=readonly build — then it is compiled and
// run, and its verdict relayed.
func (m *Moduleless) BuildAndRun(ctx context.Context, source []byte, output string) error {
	if err := os.WriteFile(filepath.Join(m.Dir, "solmain.go"), source, 0o666); err != nil {
		return fmt.Errorf("writing generated compiler: %v", err)
	}
	if err := buildCompiler(ctx, m.Dir, m.env, m.stderr); err != nil {
		return err
	}
	return runCompiler(ctx, m.Dir, output, m.stderr)
}

// BuildAndExec finishes a run the way BuildAndRun finishes a build:
// the generated tailored host replaces the probe, is compiled, and
// then runs in dir under the [Exec] contract. The host main imports
// one framework package the probe never named — solution/host — but it
// rides the framework module the probe already resolved, so the tidied
// go.mod satisfies this build too.
func (m *Moduleless) BuildAndExec(ctx context.Context, source []byte, dir string, ex *Exec) error {
	if err := os.WriteFile(filepath.Join(m.Dir, "solmain.go"), source, 0o666); err != nil {
		return fmt.Errorf("writing generated host: %v", err)
	}
	if err := buildCompiler(ctx, m.Dir, m.env, m.stderr); err != nil {
		return err
	}
	return execProgram(ctx, m.Dir, dir, ex, m.stderr)
}

// probeSource renders the resolution probe: a placeholder main whose
// blank imports carry the solution's import paths plus the solution
// package the generated compiler will need, so one tidy resolves them
// all.
func probeSource(imports []string) []byte {
	var b bytes.Buffer
	b.WriteString("// Code generated by sdl build; DO NOT EDIT.\n")
	b.WriteString("\npackage main\n")
	b.WriteString("\nimport (\n")
	fmt.Fprintf(&b, "\t_ %q\n", solutionPath)
	if len(imports) > 0 {
		b.WriteString("\n")
	}
	for _, path := range imports {
		fmt.Fprintf(&b, "\t_ %q\n", path)
	}
	b.WriteString(")\n")
	b.WriteString("\nfunc main() {}\n")
	return b.Bytes()
}

// tidy resolves the probe's imports into the work module's go.mod. The
// -e flag keeps unresolvable imports from aborting the run: tidy still
// reports each failure's cause on the relayed stderr, the go.mod gains
// no requirement for it, and discovery then positions the fault at the
// solution's import spec. A tidy that fails despite -e reports an
// environmental fault, not a solution one.
func tidy(ctx context.Context, workdir string, env []string, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy", "-e")
	cmd.Dir = workdir
	cmd.Env = env
	cmd.Stderr = stderr // download progress and resolution failures alike
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("resolving solution imports: go mod tidy: %v", err)
	}
	return nil
}

// checkModulelessPrototype runs the version-skew handshake against the
// freshly resolved work module: with no user module in sight the
// prototype arrives at whatever version tidy picked, so the comparison
// against the running CLI is the one place skew would ever surface. A
// prototype that failed to resolve at all is fatal here — the generated
// compiler cannot build without it, and no solution edit fixes a
// missing framework module.
func checkModulelessPrototype(ctx context.Context, workdir string, env []string, stderr io.Writer) error {
	const format = "{{if .Error}}error {{.Error.Err}}{{else}}" + moduleRowFormat + "{{end}}"
	out, err := goOutput(ctx, workdir, env, "list", "-m", "-e", "-f", format, prototypePath)
	if err != nil {
		return err
	}
	line := strings.TrimRight(out, " \n")
	if msg, ok := strings.CutPrefix(line, "error "); ok {
		return fmt.Errorf("cannot resolve %s, which the generated solution compiler imports: %s", prototypePath, msg)
	}
	m, err := parseModuleRow(line, "")
	if err != nil {
		return err
	}
	comparePrototype(m, cliVersion(), stderr)
	return nil
}

// goEnvVersion reports the toolchain's own version string ("go1.26.5").
func goEnvVersion(ctx context.Context, env []string) (string, error) {
	out, err := goOutput(ctx, ".", env, "env", "GOVERSION")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// scaffoldGoVersion converts a toolchain version ("go1.26.5") into a go
// directive value for the scaffold module. A development toolchain's
// version ("devel go1.27-abc123") fits no directive; the empty result
// omits the directive rather than write one the toolchain would reject.
func scaffoldGoVersion(goversion string) string {
	v := strings.TrimPrefix(goversion, "go")
	if v == "" || v[0] < '0' || v[0] > '9' {
		return ""
	}
	return v
}
