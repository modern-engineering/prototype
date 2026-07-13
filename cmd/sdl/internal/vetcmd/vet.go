// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package vetcmd implements sdl vet, the source-level lint of solution
// units and the sibling of fmt: parse the units, then report
// constructs the parser admits but the build is certain to reject,
// plus the one construct the build admits that is almost never meant —
// a var or extern symbol declared and never referenced. Vet is
// catalogue-free by design: it never resolves imports or element
// schemas, so it runs wherever the sources are, with no module context
// in reach. Semantic validation against a live catalogue is the
// build's LINK act, and checking an emitted image is plumbing under
// sdl image; neither is lint.
//
// # Drift risk
//
// Most checks shadow the linker's source-shaped half (solution's
// compile.go), and a shadow can drift from its original. The section
// and top-level-field vocabulary cannot: both sides consume
// solution.Vocabulary(), the one value the linker enforces. The
// per-word policies, the duplicate rules, and the diagnostic wordings,
// however, are duplicated by hand. Factoring one shared source checker
// that the linker, vet, and a language server all call is the recorded
// door; reopen it when a linker check changes without its shadow
// following (the corpus sweep test trips on the half that turns vet
// stricter than the build) or when the duplicated set next grows.
package vetcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/sdl/ast"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/scanner"
)

// CmdVet is the sdl vet command.
var CmdVet = &base.Command{
	UsageLine: "sdl vet [path ...]",
	Short:     "report likely mistakes in solution units",
	Long: `Vet examines solution units and reports likely mistakes: constructs
the parser admits but sdl build is certain to reject, and — the one
check the build does not share — var and extern symbols declared and
never referenced, which a build would happily pin into the image for
every deployment site to bind.

Each path is a unit file to examine or a directory to walk recursively
for .sdl files (files and directories whose names begin with a dot are
skipped); explicitly named files are always examined. With no paths,
vet examines the current directory.

The units of one directory check together as one solution unit set,
the way sdl build links a solution directory: checks that span units —
duplicate names, unused symbols — judge each directory's units against
each other and nothing else. Naming a single file checks it as a
one-unit set, so name the solution directory for whole-solution
verdicts.

Vet is deliberately catalogue-free: it never resolves imports or
element schemas, so it judges source shapes only and runs wherever the
sources are. Element resolution, parameter names and types, and value
binding belong to sdl build's LINK act; validating an emitted image is
sdl image's business; neither is lint.

The checks are:

	syntax       the parse errors, exactly as fmt reports them
	sections     unknown section words against the linker's section
	             vocabulary; params and metadata refuse a qualifier,
	             with requires one; one params and one metadata
	             section per body, one with stanza per qualifier
	root fields  top-level fields against the statement verb's closed
	             scheme (a type default's verb is catalogue knowledge,
	             so its root fields pass unjudged)
	metadata     values must be string literals
	names        duplicate instance and symbol names across the set
	symbols      var and extern symbols declared and never referenced;
	             only params-section values reference symbols — a bare
	             identifier in a top-level field or a with stanza is
	             an opaque token, so it marks nothing used

Findings print to standard error, one positioned "file:line:col:
message" line each, in position order.

Exit status 0 means every unit came back clean; 1 reports findings;
2 reports a malformed invocation or unreadable paths (each of which is
also printed to standard error).`,
}

func init() {
	CmdVet.Run = runVet
}

func runVet(ctx context.Context, cmd *base.Command, args []string) error {
	v := &vetter{stderr: os.Stderr}
	return v.run(args)
}

// A vetter carries one sdl vet invocation: the unit files gathered
// into their directory groups, the findings accumulated for one
// sorted report, and the fault tally. Faults never stop the gathering
// — every remaining path is still examined, the fmt discipline — they
// only decide the exit code.
type vetter struct {
	stderr io.Writer

	groups   map[string][]string // directory → its unit files, one set each
	visited  map[string]bool     // cleaned unit paths, so overlapping arguments check once
	findings scanner.ErrorList
	faults   int // unreadable paths and the like; exit 2
}

// run gathers the unit files of every path, checks them one directory
// group at a time, prints the findings in position order, and
// translates the verdict into the error main maps onto the exit code:
// faults (2) win over findings (1), and the findings still print
// either way.
func (v *vetter) run(paths []string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	v.groups = make(map[string][]string)
	v.visited = make(map[string]bool)
	for _, path := range paths {
		info, err := os.Stat(path)
		switch {
		case err != nil:
			v.errorf("%v", err)
		case info.IsDir():
			v.dir(path)
		default:
			v.add(path)
		}
	}
	for _, dir := range slices.Sorted(maps.Keys(v.groups)) {
		v.check(v.groups[dir])
	}

	v.findings.Sort()
	for _, e := range v.findings {
		_, _ = fmt.Fprintln(v.stderr, e)
	}
	switch {
	case v.faults == 1:
		return errors.New("1 error")
	case v.faults > 1:
		return fmt.Errorf("%d errors", v.faults)
	case len(v.findings) > 0:
		// The findings are already on stderr; the empty diagnostics
		// error only selects exit code 1.
		return &base.DiagnosticsError{}
	}
	return nil
}

// errorf reports one fault toward exit code 2. The report write is
// best effort — the exit code carries the verdict — as are all of the
// vetter's stream writes, the fmt precedent.
func (v *vetter) errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(v.stderr, format+"\n", args...)
	v.faults++
}

// dir gathers every .sdl file under root, skipping dot-files and
// dot-directories. Walk faults are reported and the walk continues.
func (v *vetter) dir(root string) {
	// WalkDir's callback only returns an error to stop the walk, which
	// a vet sweep never wants; faults are tallied on v instead.
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			v.errorf("%v", err)
			return nil
		}
		dot := strings.HasPrefix(d.Name(), ".") && path != root
		if d.IsDir() {
			if dot {
				return fs.SkipDir
			}
			return nil
		}
		if dot || !strings.HasSuffix(d.Name(), ".sdl") {
			return nil
		}
		v.add(path)
		return nil
	})
}

// add enters one unit file into its directory's group, once: each
// directory is its own unit set, so a walk over a tree of solutions
// checks every solution against itself and never one merged
// namespace.
func (v *vetter) add(path string) {
	path = filepath.Clean(path)
	if v.visited[path] {
		return
	}
	v.visited[path] = true
	dir := filepath.Dir(path)
	v.groups[dir] = append(v.groups[dir], path)
}

// check runs one directory group as a unit set. Files check in name
// order — the build's own unit order — so "first declared at" anchors
// match sdl build's. A unit with syntax errors contributes them as
// findings and drops out of the semantic checks; if any unit of the
// set is broken or unreadable, the set-scope checks are skipped
// entirely: the linker's syntax-gates-linking posture, and half a
// namespace would only yield false duplicate and unused findings.
func (v *vetter) check(files []string) {
	slices.Sort(files)
	c := newChecker()
	units := make([]*ast.File, 0, len(files))
	whole := true
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			v.errorf("%v", err)
			whole = false
			continue
		}
		f, err := parser.ParseFile(path, src)
		if err != nil {
			c.syntax(path, err)
			whole = false
			continue
		}
		c.unit(f)
		units = append(units, f)
	}
	if whole {
		c.unused(units, c.declarations(units))
	}
	v.findings = append(v.findings, c.diags...)
}
