// Package seftool hosts the SEF command with caller-supplied streams and an
// optional compiled descriptor registry. It does not dynamically import Go code.
package seftool

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/modern-engineering/prototype/program"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution/edit"
	"github.com/modern-engineering/prototype/solution/gen"
	"github.com/modern-engineering/prototype/solution/link"
)

// Command supplies streams and the catalogue available to one command host.
// Nil streams mean empty input or discarded output. Registry is needed only
// for -linked or -inputs operations; the caller owns its registrations.
type Command struct {
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Registry *program.Registry
}

// Run processes fmt, convert, check, inspect, dump, rename, or gen. It returns 0 on
// success, 1 on data or operation failure, and 2 on invalid command usage.
// Flags precede the optional input filename. A missing filename or "-" reads
// Stdin; missing output or "-" writes Stdout. Formats are inferred from
// .json.sef, .gob.sef, or .sef; stdin defaults to text, overridable by -from.
// Streams are left open. Each Run operates independently.
//
// fmt preserves text comments and can replace a named file with -w. convert
// and dump preserve semantic content, not source comments. check validates
// structural relationships; -linked additionally checks the registry.
// inspect -inputs instantiates fresh I/O declarations, but never assigns,
// reads values, or executes runners. gen emits registration source for a later
// Go build. rename rewrites an occurrence and all its relationships, preserving
// the input encoding on stdout or inferring it from the output filename. Module
// selection and execution are outside these commands.
func (c Command) Run(args []string) int {
	if c.Stdin == nil {
		c.Stdin = strings.NewReader("")
	}
	if c.Stdout == nil {
		c.Stdout = io.Discard
	}
	if c.Stderr == nil {
		c.Stderr = io.Discard
	}
	if len(args) == 0 {
		c.usage()
		return 2
	}
	if args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		c.usage()
		return 0
	}
	switch args[0] {
	case "fmt", "convert", "check", "inspect", "dump", "rename", "gen":
	default:
		_, _ = fmt.Fprintf(c.Stderr, "sef: unknown command %q\n", args[0])
		c.usage()
		return 2
	}
	status, err := c.run(args[0], args[1:])
	if err != nil {
		_, _ = fmt.Fprintf(c.Stderr, "sef: %v\n", err)
	}
	return status
}

func (c Command) usage() {
	_, _ = fmt.Fprintln(c.Stderr, "usage: sef <fmt|convert|check|inspect|dump|rename|gen> [flags] [file|-]")
	_, _ = fmt.Fprintln(c.Stderr, "use sef <command> -h for flags; linked inspection needs a host with a compiled registry")
}

func (c Command) run(name string, args []string) (int, error) {
	flags := flag.NewFlagSet("sef "+name, flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	from, to, output, packageName, functionName := "auto", "auto", "-", "catalogue", "Register"
	write, linked, inputs := false, false, false
	oldName, newName := "", ""
	flags.Usage = func() {
		_, _ = fmt.Fprintf(c.Stderr, "usage: sef %s [flags] [file|-]\n", name)
		flags.PrintDefaults()
	}
	if name == "fmt" {
		flags.BoolVar(&write, "w", false, "replace a named text file")
	} else if name != "rename" {
		flags.StringVar(&from, "from", "auto", "input format: auto, text, json, gob")
	}
	if name == "convert" {
		flags.StringVar(&to, "to", "auto", "output format: text, json, gob (default: output suffix, otherwise json)")
	}
	if name == "convert" || name == "dump" || name == "rename" || name == "gen" {
		flags.StringVar(&output, "o", "-", "output file or - for stdout")
	}
	if name == "check" || name == "inspect" {
		flags.BoolVar(&linked, "linked", false, "resolve descriptors through the host's compiled registry")
	}
	if name == "inspect" {
		flags.BoolVar(&inputs, "inputs", false, "instantiate and display I/O declarations (implies -linked)")
	}
	if name == "rename" {
		flags.StringVar(&oldName, "from", "", "existing occurrence name")
		flags.StringVar(&newName, "to", "", "new occurrence name")
	}
	if name == "gen" {
		flags.StringVar(&packageName, "package", "catalogue", "generated Go package name")
		flags.StringVar(&functionName, "func", "Register", "generated registration function")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, nil
	}
	if flags.NArg() > 1 {
		return 2, errors.New("expected at most one input file")
	}
	path := "-"
	if flags.NArg() == 1 {
		path = flags.Arg(0)
	}
	renaming := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { renaming[f.Name] = true })
	if name == "rename" && (!renaming["from"] || !renaming["to"]) {
		return 2, errors.New("rename requires -from and -to occurrence names")
	}
	if write && path == "-" {
		return 2, errors.New("fmt -w requires a named file")
	}
	if !validFormat(from) || !validFormat(to) {
		return 2, errors.New("format must be auto, text, json, or gob")
	}
	if (linked || inputs) && c.Registry == nil {
		return 2, errors.New("this host has no compiled descriptor registry; use structural inspection or build a catalogue-aware host")
	}
	data, err := c.read(path)
	if err != nil {
		return 1, err
	}
	var result bytes.Buffer
	if name == "fmt" {
		file, err := seftext.Parse(path, bytes.NewReader(data))
		if err != nil {
			return 1, err
		}
		if err := seftext.Format(&result, file); err != nil {
			return 1, err
		}
		if write {
			output = path
		}
	} else {
		artifact, err := decode(path, from, data)
		if err != nil {
			return 1, err
		}
		switch name {
		case "convert", "dump":
			if name == "dump" {
				to = "json"
			}
			if to == "auto" {
				to = formatFor(output)
				if output == "-" {
					to = "json"
				}
			}
			err = encode(&result, to, artifact)
		case "rename":
			artifact.Definition, err = edit.Rename(artifact.Definition, oldName, newName)
			if err == nil {
				format := formatFor(path)
				if output != "-" {
					format = formatFor(output)
				}
				err = encode(&result, format, artifact)
			}
		case "check":
			if linked {
				err = link.Check(artifact.Definition, c.Registry)
			} else {
				err = link.CheckStructure(artifact.Definition)
			}
			if err == nil {
				fmt.Fprintln(&result, "ok")
			}
		case "inspect":
			err = c.inspect(&result, artifact, linked || inputs, inputs)
		case "gen":
			var source []byte
			source, err = gen.Source(packageName, functionName, artifact.Definition)
			if err == nil {
				_, err = result.Write(source)
			}
		}
		if err != nil {
			return 1, err
		}
	}
	if err := c.write(output, result.Bytes()); err != nil {
		return 1, err
	}
	return 0, nil
}
