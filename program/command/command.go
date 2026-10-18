// Package command adapts declared program fields for command-line use.
// FromFlags exposes existing flag values as inputs; Parse exposes declared inputs
// as flags. WriteOutputs formats all declared results as a JSON object.
// Commands retain control of construction, resources, execution, and writers.
package command

import (
	"flag"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/program"
)

// Parse declares a flag for each input and parses args, which excludes the
// command name. The caller may declare additional flags in fs first. Names must
// be unique across both sets; flag's usual declaration panics apply. Call Parse
// once per FlagSet. A nil inputs value declares no flags.
//
// Registration never calls Set, so omitted arguments preserve constructor
// values. Repeated arguments call the original setter in order. Setters with
// IsBoolFlag() bool retain flag's boolean shorthand. Ordinary defaults come from
// program.Getter.Get or, failing that, flag.Value.String. Setter-only inputs
// have no displayed default. Sensitive defaults are never read or displayed.
//
// fs must use flag.ContinueOnError; other modes panic. Parse suppresses flag's
// automatic error diagnostics and returns errors for the caller to report.
// Rejected input values return an *InputError, whose message omits a sensitive
// value and its underlying error. On flag.ErrHelp, Parse invokes fs.Usage (or
// prints the standard usage when nil) after restoring fs.Output. Custom usage
// functions remain responsible for any additional information they display.
// Other parsing failures return a generic message without argument text, since
// malformed arguments may contain sensitive values before an input is identified.
// These errors unwrap to flag's original diagnostic, which may contain sensitive
// values and must not be logged.
//
// Parse follows flag's argument grammar, including stopping at the first
// positional argument. Remaining arguments are available through fs.Args.
// Successful assignments preceding an error remain applied. Parse neither
// constructs nor runs a program and does not acquire resources.
func Parse(fs *flag.FlagSet, inputs program.Inputs, args []string) error {
	if fs.ErrorHandling() != flag.ContinueOnError {
		panic("command: Parse requires flag.ContinueOnError")
	}
	var failed *InputError
	if inputs != nil {
		for input := range inputs.All() {
			fs.Var(&value{input: input, failed: &failed}, input.Name, input.Help)
		}
	}
	err := parse(fs, args)
	if failed != nil {
		return failed
	}
	if err == flag.ErrHelp {
		if fs.Usage != nil {
			fs.Usage()
		} else {
			// Match FlagSet.PrintDefaults: usage reporting has no error result.
			if fs.Name() == "" {
				_, _ = fmt.Fprintln(fs.Output(), "Usage:")
			} else {
				_, _ = fmt.Fprintf(fs.Output(), "Usage of %s:\n", fs.Name())
			}
			fs.PrintDefaults()
		}
		return flag.ErrHelp
	}
	if err != nil {
		return &syntaxError{err: err}
	}
	return nil
}

type syntaxError struct{ err error }

func (e *syntaxError) Error() string { return "invalid command-line arguments; use -help for usage" }

func (e *syntaxError) Unwrap() error { return e.err }

// InputError reports a rejected input assignment. Error omits Err when Sensitive
// is true. Unwrap deliberately exposes the original error for errors.Is and
// errors.As; that error may contain the sensitive value and must not be logged.
type InputError struct {
	Name      string
	Sensitive bool
	Err       error
}

func (e *InputError) Error() string {
	if e.Sensitive {
		return fmt.Sprintf("invalid value for -%s", e.Name)
	}
	return fmt.Sprintf("invalid value for -%s: %v", e.Name, e.Err)
}

func (e *InputError) Unwrap() error { return e.Err }

func parse(fs *flag.FlagSet, args []string) error {
	output, usage := fs.Output(), fs.Usage
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	defer func() {
		fs.SetOutput(output)
		fs.Usage = usage
	}()
	return fs.Parse(args)
}

type value struct {
	input  program.Input
	failed **InputError
}

func (v *value) String() string {
	// PrintDefaults calls String on a newly allocated zero value too.
	if v.input.Sensitive || v.input.Value == nil {
		return ""
	}
	if getter, ok := v.input.Value.(program.Getter); ok {
		return getter.Get()
	}
	if original, ok := v.input.Value.(flag.Value); ok {
		return original.String()
	}
	return ""
}

func (v *value) Set(text string) error {
	if err := v.input.Value.Set(text); err != nil {
		failure := &InputError{Name: v.input.Name, Sensitive: v.input.Sensitive, Err: err}
		*v.failed = failure
		return failure
	}
	return nil
}

func (v *value) IsBoolFlag() bool {
	original, ok := v.input.Value.(interface{ IsBoolFlag() bool })
	return ok && original.IsBoolFlag()
}
