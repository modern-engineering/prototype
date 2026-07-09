// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
)

// runWorkspace drives a build inside an active workspace: the work
// directory joins a synthesized copy of the user's workspace as one
// more member, and the union build list resolves the generated
// compiler's imports exactly as it resolves the user's own packages —
// local member source included.
func runWorkspace(ctx context.Context, cfg Config, stderr io.Writer) error {
	env := cfg.Context.Env()
	gowork := cfg.Context.gowork
	ws, err := readWorkspace(ctx, cfg.Dir, gowork, env)
	if err != nil {
		return err
	}
	if err := checkWorkspacePrototype(ctx, cfg.Dir, gowork, env, stderr); err != nil {
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
	if err := os.WriteFile(filepath.Join(workdir, "go.mod"), scaffoldGoMod(ws.goVersion), 0o666); err != nil {
		return fmt.Errorf("writing scaffold go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "go.work"), synthesizeGoWork(ws), 0o666); err != nil {
		return fmt.Errorf("writing synthesized go.work: %v", err)
	}
	if err := copyWorkSum(filepath.Dir(gowork), workdir); err != nil {
		return err
	}

	// The build must resolve in the synthesized workspace, wherever the
	// work directory landed and whatever GOWORK the caller exported.
	buildEnv := append(os.Environ(), "GOWORK="+filepath.Join(workdir, "go.work"), "GOFLAGS=")
	if err := buildCompiler(ctx, workdir, buildEnv, stderr); err != nil {
		return err
	}
	return runCompiler(ctx, workdir, cfg.Output, stderr)
}

// A workspace is the go.work state mirrored into the temporary build
// context: the language directives, the use'd module directories
// rebased to absolute paths, and the replace directives.
type workspace struct {
	goVersion string
	toolchain string
	dirs      []string
	replaces  []replacement
}

// A replacement is one replace directive of the user's go.work. A
// version-less new path is a directory, already rebased absolute.
type replacement struct {
	oldPath, oldVersion string
	newPath, newVersion string
}

// readWorkspace parses the user's go.work through the go tool's own
// parser (go work edit -json), rebasing relative use and replacement
// directories onto the go.work's directory, where the file resolves
// them.
func readWorkspace(ctx context.Context, dir, gowork string, env []string) (*workspace, error) {
	out, err := goOutput(ctx, dir, env, "work", "edit", "-json", gowork)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Go        string
		Toolchain string
		Use       []struct{ DiskPath string }
		Replace   []struct {
			Old, New struct{ Path, Version string }
		}
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, fmt.Errorf("reading workspace directives: %v", err)
	}
	base := filepath.Dir(gowork)
	ws := &workspace{goVersion: parsed.Go, toolchain: parsed.Toolchain}
	for _, u := range parsed.Use {
		ws.dirs = append(ws.dirs, rebase(u.DiskPath, base))
	}
	for _, r := range parsed.Replace {
		rep := replacement{
			oldPath: r.Old.Path, oldVersion: r.Old.Version,
			newPath: r.New.Path, newVersion: r.New.Version,
		}
		if rep.newVersion == "" {
			rep.newPath = rebase(rep.newPath, base)
		}
		ws.replaces = append(ws.replaces, rep)
	}
	return ws, nil
}

// synthesizeGoWork renders the temporary workspace: the user's language
// directives, every use'd directory as an absolute path plus the work
// directory itself joining as one more member, and the user's replace
// directives mirrored.
func synthesizeGoWork(ws *workspace) []byte {
	var b bytes.Buffer
	if ws.goVersion != "" {
		fmt.Fprintf(&b, "go %s\n", ws.goVersion)
	}
	if ws.toolchain != "" {
		fmt.Fprintf(&b, "\ntoolchain %s\n", ws.toolchain)
	}
	b.WriteString("\nuse (\n")
	for _, dir := range ws.dirs {
		fmt.Fprintf(&b, "\t%s\n", quoteDir(dir))
	}
	b.WriteString("\t.\n)\n")
	for _, r := range ws.replaces {
		fmt.Fprintf(&b, "\nreplace %s", r.oldPath)
		if r.oldVersion != "" {
			fmt.Fprintf(&b, " %s", r.oldVersion)
		}
		b.WriteString(" => ")
		if r.newVersion != "" {
			fmt.Fprintf(&b, "%s %s\n", r.newPath, r.newVersion)
		} else {
			fmt.Fprintf(&b, "%s\n", quoteDir(r.newPath))
		}
	}
	return b.Bytes()
}

// scaffoldGoMod renders the work module for the modes that do not
// mirror a user module's requirements: a throwaway identity plus a
// language version, nothing more. In workspace mode the version is the
// workspace's own and the union build list resolves every import; in
// module-less mode it is the toolchain's and go mod tidy fills the
// requirements in.
func scaffoldGoMod(goVersion string) []byte {
	var b bytes.Buffer
	b.WriteString("module sdl.invalid/solmain\n")
	if goVersion != "" {
		fmt.Fprintf(&b, "\ngo %s\n", goVersion)
	}
	return b.Bytes()
}

// copyWorkSum copies the workspace's go.work.sum beside the synthesized
// go.work, when there is one: the synthesized workspace resolves the
// same union graph, so the user's sums verify it.
func copyWorkSum(workspaceDir, workdir string) error {
	sum, err := os.ReadFile(filepath.Join(workspaceDir, "go.work.sum"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading go.work.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "go.work.sum"), sum, 0o666); err != nil {
		return fmt.Errorf("writing go.work.sum: %v", err)
	}
	return nil
}

// checkWorkspacePrototype guards the generated compiler's own
// dependency in workspace mode with one error-tolerant 'go list -m'
// query. A workspace member — the everyday shape, reported without a
// version — and a directory-replaced module are local source with no
// version to compare, so they never warn; an ordinary version is
// compared against the running CLI's; and a prototype the workspace
// cannot resolve at all is the solution author's to fix.
func checkWorkspacePrototype(ctx context.Context, dir, gowork string, env []string, stderr io.Writer) error {
	const format = "{{if .Error}}error {{.Error.Err}}{{else}}" + moduleRowFormat + "{{end}}"
	out, err := goOutput(ctx, dir, env, "list", "-m", "-e", "-f", format, prototypePath)
	if err != nil {
		return err
	}
	line := strings.TrimRight(out, " \n")
	if msg, ok := strings.CutPrefix(line, "error "); ok {
		return &base.DiagnosticsError{Lines: []string{
			fmt.Sprintf("workspace %s does not provide %s, which the generated solution compiler imports: %s", gowork, prototypePath, msg),
		}}
	}
	// The replacement directory, if any, is never resolved here, so the
	// row needs no rebasing base.
	m, err := parseModuleRow(line, "")
	if err != nil {
		return err
	}
	comparePrototype(m, cliVersion(), stderr)
	return nil
}

// rebase makes a directory absolute against base, leaving absolute
// directories alone.
func rebase(dir, base string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(base, dir)
}
