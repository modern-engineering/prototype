// Package host serves descriptor-aware tooling in a compiled Go executable.
// Generated hosts accept materialized artifacts through gob, independently of
// their authoring format. Catalogue package initializers run when the executable
// starts; commands never execute program runners or transfer configuration.
package host

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution/link"
)

// Command supplies a compiled registry and streams. Nil streams mean empty input
// or discarded output. Registry must be non-nil for check and inspect.
type Command struct {
	Registry *program.Registry
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

// Run accepts check or inspect, with -inputs supported only by inspect. It reads
// one gob artifact from Stdin and leaves all streams open. Check and ordinary
// inspection do not call Make; -inputs constructs invocations but never calls
// Set, Get, or Run. Status is 0 for success, 1 for artifact/operation failure, or
// 2 for invalid usage. Unsupported commands are rejected before reading input.
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
	if len(args) == 0 || (args[0] != "check" && args[0] != "inspect") {
		_, _ = fmt.Fprintln(c.Stderr, "usage: sef-host <check|inspect> [-inputs] < artifact.gob")
		return 2
	}
	flags := flag.NewFlagSet("sef-host "+args[0], flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	inputs := false
	if args[0] == "inspect" {
		flags.BoolVar(&inputs, "inputs", false, "instantiate and inspect I/O declarations")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || c.Registry == nil {
		_, _ = fmt.Fprintln(c.Stderr, "host requires a compiled registry and reads its artifact from stdin")
		return 2
	}
	artifact, err := sef.DecodeGob(c.Stdin)
	if err != nil {
		_, _ = fmt.Fprintln(c.Stderr, "decode artifact:", err)
		return 1
	}
	if args[0] == "check" {
		err = link.Check(artifact.Definition, c.Registry)
		if err == nil {
			_, err = fmt.Fprintln(c.Stdout, "ok")
		}
	} else {
		err = Inspect(c.Stdout, artifact, c.Registry, inputs)
	}
	if err != nil {
		_, _ = fmt.Fprintln(c.Stderr, args[0]+" artifact:", err)
		return 1
	}
	return 0
}
