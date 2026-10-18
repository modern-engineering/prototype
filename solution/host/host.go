// Package host serves descriptor-aware tooling in a compiled Go executable.
// Generated hosts accept materialized artifacts through gob, independently of
// their authoring format. Catalogue package initializers run when the executable
// starts; check and inspect never execute program runners or transfer
// configuration. Playground runs the artifact's jobs and applications.
package host

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/modern-engineering/prototype/playground"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

// Command supplies a compiled registry and streams. Nil streams mean empty input
// or discarded output. Registry must be non-nil. A nil Context means
// context.Background; playground passes it to every invocation.
type Command struct {
	Context  context.Context
	Registry *program.Registry
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

// Run accepts check, inspect, or playground. Inspect supports -inputs;
// playground supports a positive -job-timeout, defaulting to one minute per job.
// It reads one gob artifact from Stdin and leaves all streams open. Check and
// ordinary inspection do not call Make; -inputs constructs invocations but never calls
// Set, Get, or Run. Status is 0 for success, 1 for artifact/operation failure, or
// 2 for invalid usage. Unsupported commands are rejected before reading input.
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
	if len(args) == 0 || (args[0] != "check" && args[0] != "inspect" && args[0] != "playground") {
		_, _ = fmt.Fprintln(c.Stderr, "usage: sef-host <check|inspect|playground> [flags] < artifact.gob")
		return 2
	}
	flags := flag.NewFlagSet("sef-host "+args[0], flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	inputs := false
	jobTimeout := time.Minute
	if args[0] == "inspect" {
		flags.BoolVar(&inputs, "inputs", false, "instantiate and inspect I/O declarations")
	}
	if args[0] == "playground" {
		flags.DurationVar(&jobTimeout, "job-timeout", time.Minute, "cooperative execution deadline for each job")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if args[0] == "playground" && jobTimeout <= 0 {
		_, _ = fmt.Fprintln(c.Stderr, "job-timeout must be positive")
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
	switch args[0] {
	case "check":
		err = link.Check(artifact.Definition, c.Registry)
		if err == nil {
			_, err = fmt.Fprintln(c.Stdout, "ok")
		}
	case "inspect":
		err = Inspect(c.Stdout, artifact, c.Registry, inputs)
	case "playground":
		linked, resolveErr := link.Resolve(artifact.Definition, c.Registry)
		err = resolveErr
		if err == nil {
			instance, instantiateErr := load.Instantiate(linked)
			err = instantiateErr
			if err == nil {
				err = playground.Run(c.Context, instance, playground.Options{JobTimeout: jobTimeout})
			}
		}
	}
	if err != nil {
		_, _ = fmt.Fprintln(c.Stderr, args[0]+" artifact:", err)
		return 1
	}
	return 0
}
