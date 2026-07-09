// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package fmtcmd implements sdl fmt, the gofmt of solution units: parse
// each unit, print it through sdl/printer, and report or apply the
// difference. Formatting normalizes layout only — lexemes reprint as
// authored — so running fmt never changes what a unit means.
package fmtcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/printer"
	"github.com/modern-engineering/prototype/sdl/scanner"
)

// CmdFmt is the sdl fmt command.
var CmdFmt = &base.Command{
	UsageLine: "sdl fmt [-l] [-w] [-d] [path ...]",
	Short:     "reformat solution units in canonical form",
	Long: `Fmt reformats solution units in the canonical form the sdl toolchain
prints: statements in source order, tab indentation inside blocks, one
space after a parameter colon and around "as", runs of blank lines
collapsed to one, and comments kept in place. Lexemes are left exactly
as authored — fmt normalizes layout, never values.

Each path is a file to format or a directory to walk recursively for
.sdl files (files and directories whose names begin with a dot are
skipped); explicitly named files are always formatted. With no paths,
fmt walks the current directory.

Without flags, fmt prints the canonical form of every unit to standard
output. The flags change that:

	-l	list files whose formatting differs from the canonical form
	-w	rewrite differing files in place instead of printing
	-d	print unified diffs from each differing file to its
		canonical form

Exit status 0 means every unit was already canonical (or was printed or
rewritten); 1 reports that -l or -d found differences; 2 reports errors
— syntax faults, unreadable paths, failed rewrites — of which each is
also printed to standard error.`,
}

var (
	flagList  bool
	flagWrite bool
	flagDiff  bool
)

func init() {
	CmdFmt.Run = runFmt // break init cycle: Run references CmdFmt's flags
	CmdFmt.Flag.BoolVar(&flagList, "l", false, "list files whose formatting differs")
	CmdFmt.Flag.BoolVar(&flagWrite, "w", false, "write the canonical form back to the files")
	CmdFmt.Flag.BoolVar(&flagDiff, "d", false, "display diffs instead of rewriting files")
}

func runFmt(ctx context.Context, cmd *base.Command, args []string) error {
	f := &formatter{
		list:   flagList,
		write:  flagWrite,
		diff:   flagDiff,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
	return f.run(args)
}

// A formatter carries one sdl fmt invocation: the mode flags, the
// streams, and the running verdict. Faults never stop the walk — every
// remaining file is still processed — they only decide the exit code.
type formatter struct {
	list, write, diff bool
	stdout, stderr    io.Writer

	faults  int // errors reported to stderr; exit 2
	differs int // files whose form is not canonical; exit 1 under -l/-d
}

// run processes every path and translates the collected verdict into
// the error main maps onto the exit code: faults win over differences.
func (f *formatter) run(paths []string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		switch {
		case err != nil:
			f.errorf("%v", err)
		case info.IsDir():
			f.dir(path)
		default:
			f.file(path, info.Mode())
		}
	}
	switch {
	case f.faults == 1:
		return errors.New("1 error")
	case f.faults > 1:
		return fmt.Errorf("%d errors", f.faults)
	case f.differs > 0 && (f.list || f.diff):
		// The findings are already on stdout; the empty diagnostics
		// error only selects exit code 1.
		return &base.DiagnosticsError{}
	}
	return nil
}

// errorf reports one fault toward exit code 2. The report write is
// best effort — the exit code carries the verdict — as are all of the
// formatter's stream writes, the printUsage precedent.
func (f *formatter) errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(f.stderr, format+"\n", args...)
	f.faults++
}

// dir formats every .sdl file under root, skipping dot-files and
// dot-directories. Walk faults are reported and the walk continues.
func (f *formatter) dir(root string) {
	// WalkDir's callback only returns an error to stop the walk, which
	// a formatting sweep never wants; faults are tallied on f instead.
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			f.errorf("%v", err)
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
		info, err := d.Info()
		if err != nil {
			f.errorf("%v", err)
			return nil
		}
		f.file(path, info.Mode())
		return nil
	})
}

// file formats one unit and applies the modes to it.
func (f *formatter) file(path string, mode fs.FileMode) {
	src, err := os.ReadFile(path)
	if err != nil {
		f.errorf("%v", err)
		return
	}
	parsed, err := parser.ParseFile(path, src)
	if err != nil {
		var list scanner.ErrorList
		if errors.As(err, &list) {
			for _, e := range list {
				_, _ = fmt.Fprintln(f.stderr, e)
			}
			f.faults += len(list)
			return
		}
		f.errorf("%s: %v", path, err)
		return
	}
	res, err := printer.Source(parsed)
	if err != nil {
		f.errorf("%s: %v", path, err)
		return
	}

	if !bytes.Equal(src, res) {
		f.differs++
		if f.list {
			_, _ = fmt.Fprintln(f.stdout, path)
		}
		if f.write {
			if err := writeFile(path, res, mode.Perm()); err != nil {
				f.errorf("%v", err)
			}
		}
		if f.diff {
			_, _ = f.stdout.Write(unifiedDiff(path+".orig", path, src, res))
		}
	}
	if !f.list && !f.write && !f.diff {
		_, _ = f.stdout.Write(res)
	}
}

// writeFile atomically replaces path with data: the bytes land in a
// temporary file in the same directory, take the original permissions,
// and move over the original in one rename, so a crash never leaves a
// half-written unit.
func writeFile(path string, data []byte, perm fs.FileMode) error {
	dir, base := filepath.Split(path)
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(name, perm)
	}
	if werr == nil {
		werr = os.Rename(name, path)
	}
	if werr != nil {
		_ = os.Remove(name) // the write fault is the one worth reporting
		return fmt.Errorf("rewriting %s: %v", path, werr)
	}
	return nil
}
