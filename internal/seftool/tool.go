// Package seftool hosts the SEF command with caller-supplied streams and an
// optional compiled descriptor registry. Linked commands can generate and build
// a separate Go host from the descriptors named by an artifact.
package seftool

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/modern-engineering/prototype/program"
	sefcue "github.com/modern-engineering/prototype/sef/cue"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution/build"
	"github.com/modern-engineering/prototype/solution/edit"
	"github.com/modern-engineering/prototype/solution/gen"
	"github.com/modern-engineering/prototype/solution/link"
)

// Command supplies streams and the catalogue available to one command host.
// Nil streams mean empty input or discarded output. A nil Registry selects
// generated hosts for linked operations. Context cancels host builds and runs;
// nil uses context.Background. The caller owns a supplied registry.
type Command struct {
	Context  context.Context
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Registry *program.Registry
}

// Run processes fmt, convert, check, inspect, dump, rename, gen, or build. It returns 0 on
// success, 1 on data or operation failure, and 2 on invalid command usage.
// Flags precede the optional input filename. A missing filename or "-" reads
// Stdin; missing output or "-" writes Stdout. Formats are inferred from
// .cue.sef, .cue, .json.sef, .gob.sef, or .sef; stdin defaults to text,
// overridable by -from. CUE formatting checks syntax without evaluating data.
// Streams are left open. Each Run operates independently.
//
// fmt preserves text comments and can replace a named file with -w. convert
// and dump preserve semantic content, not source comments. check validates
// structural relationships; -linked additionally checks the registry.
// inspect -inputs instantiates fresh I/O declarations, but never assigns,
// reads values, or executes runners. gen emits registration source; -main emits
// a complete host. build compiles a host without running it. rename rewrites an occurrence and all its relationships, preserving
// the input encoding on stdout or inferring it from the output filename. Module
// requirements select the isolated generated module. -dev opts into caller
// module source for its descriptors without inheriting its dependency graph.
// -toolkit and -replace
// provide explicit local source overrides, and -work retains generated files.
func (c Command) Run(args []string) int {
	if c.Context == nil {
		c.Context = context.Background()
	}
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
	case "fmt", "convert", "check", "inspect", "dump", "rename", "gen", "build":
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
	_, _ = fmt.Fprintln(c.Stderr, "usage: sef <fmt|convert|check|inspect|dump|rename|gen|build> [flags] [file|-]")
	_, _ = fmt.Fprintln(c.Stderr, "use sef <command> -h for flags; linked commands build a host when no registry is supplied")
}

func (c Command) run(name string, args []string) (int, error) {
	flags := flag.NewFlagSet("sef "+name, flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	from, to, output, packageName, functionName := "auto", "auto", "-", "catalogue", "Register"
	write, linked, inputs, mainSource, dev := false, false, false, false, false
	buildOptions := build.Options{ToolkitVersion: toolkitVersion()}
	oldName, newName := "", ""
	flags.Usage = func() {
		_, _ = fmt.Fprintf(c.Stderr, "usage: sef %s [flags] [file|-]\n", name)
		flags.PrintDefaults()
	}
	if name == "fmt" {
		flags.BoolVar(&write, "w", false, "replace a named source file")
	}
	if name != "rename" {
		flags.StringVar(&from, "from", "auto", "input format: auto, text, cue, json, gob")
	}
	if name == "convert" {
		flags.StringVar(&to, "to", "auto", "output format: text, cue, json, gob (default: output suffix, otherwise json)")
	}
	if name == "build" {
		output = "solution.host"
	}
	if name == "convert" || name == "dump" || name == "rename" || name == "gen" || name == "build" {
		flags.StringVar(&output, "o", output, "output file (build requires a new named file)")
	}
	if name == "check" || name == "inspect" {
		flags.BoolVar(&linked, "linked", false, "resolve descriptors through a compiled registry, building a host if needed")
	}
	if name == "inspect" {
		flags.BoolVar(&inputs, "inputs", false, "instantiate and display I/O declarations (implies -linked)")
	}
	if name == "rename" {
		flags.StringVar(&oldName, "from", "", "existing occurrence name")
		flags.StringVar(&newName, "to", "", "new occurrence name")
	}
	if name == "build" || name == "check" || name == "inspect" {
		flags.BoolVar(&dev, "dev", false, "use caller module source for its referenced descriptors")
		flags.BoolVar(&buildOptions.KeepWork, "work", false, "retain generated source, module, and binary")
		flags.StringVar(&buildOptions.ToolkitDir, "toolkit", "", "local toolkit module directory for generated hosts")
		flags.Func("replace", "local module replacement: module=directory (repeatable)", func(value string) error {
			module, dir, ok := strings.Cut(value, "=")
			if !ok || module == "" || dir == "" {
				return errors.New("replacement must be module=directory")
			}
			buildOptions.Replacements = append(buildOptions.Replacements, build.Replacement{Module: module, Dir: dir})
			return nil
		})
	}
	if name == "gen" {
		flags.BoolVar(&mainSource, "main", false, "generate a complete descriptor host")
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
		return 2, errors.New("format must be auto, text, cue, json, or gob")
	}
	if name == "build" && output == "-" {
		return 2, errors.New("build requires a named output file")
	}
	if mainSource && (renaming["package"] || renaming["func"]) {
		return 2, errors.New("gen -main cannot be combined with -package or -func")
	}
	if (dev || buildOptions.KeepWork || renaming["toolkit"] || renaming["replace"]) && name != "build" && (!linked && !inputs || c.Registry != nil) {
		return 2, errors.New("build options require a generated host")
	}
	data, err := c.read(path)
	if err != nil {
		return 1, err
	}
	var result bytes.Buffer
	if name == "fmt" {
		format := from
		if format == "auto" {
			format = formatFor(path)
		}
		switch format {
		case "cue":
			err = sefcue.FormatSource(&result, path, bytes.NewReader(data))
		case "text":
			var file *seftext.File
			file, err = seftext.Parse(path, bytes.NewReader(data))
			if err == nil {
				err = seftext.Format(&result, file)
			}
		default:
			return 2, errors.New("fmt requires text or cue source")
		}
		if err != nil {
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
		if name == "build" || (c.Registry == nil && (linked || inputs)) {
			return c.generated(name, artifact, inputs, output, buildOptions, dev)
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
			if mainSource {
				source, err = gen.Main(artifact.Definition)
			} else {
				source, err = gen.Source(packageName, functionName, artifact.Definition)
			}
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
