package loaderflags

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"iter"
)

const (
	// All parameter flag-sets share the same internal name so that [Parse]
	// can distinguish them from flag-sets created outside this package.
	// The display name (passed to [NewFlagSet]) appears only in usage output.
	paramSetName = "<APPLICATION-PARAMETERS>"
)

// NewFlagSet creates a parameter flag-set whose usage output is labeled with
// name. The returned flag-set uses [flag.ContinueOnError] so that [Parse] can
// handle errors without terminating the process.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(paramSetName, flag.ContinueOnError)
	fs.Usage = usageOf(fs, name)
	return fs
}

func usageOf(fs *flag.FlagSet, name string) func() {
	return func() {
		_, _ = fmt.Fprintf(fs.Output(), "Parameterization of %s:\n", name)
		fs.PrintDefaults()
	}
}

// Parse populates fs by calling each source's [Parser.ParseParameters] in order.
// Sources listed first take precedence: a flag set by an earlier source will not
// be overwritten by a later one (provided each source honors [IsSet]).
//
// Parse panics if fs was not created by [NewFlagSet]. It returns an error if fs
// has already been parsed.
func Parse(fs *flag.FlagSet, sources ...Parser) error {
	// Reject foreign flag-sets: they may use ExitOnError or PanicOnError,
	// which would prevent Parse from handling errors gracefully.
	if fs.Name() != paramSetName {
		panic("parameter.Parse called on non-parameter flag-set")
	}

	if fs.Parsed() {
		return fmt.Errorf("already parsed")
	}

	for _, source := range sources {
		err := source.ParseParameters(fs)
		if err != nil {
			return fmt.Errorf("source %s: %w", source, err)
		}
	}

	// Call Parse with no arguments to flip Parsed() to true, signaling that
	// the flag-set is fully populated and preventing double-parsing.
	err := fs.Parse(nil)
	if err != nil {
		return err
	}
	return nil
}

// Defaults return the output of [flag.FlagSet.PrintDefaults] as a string without
// disturbing the flag-set's current output sink.
//
// Must not be called concurrently with any other flag-set methods.
func Defaults(fs *flag.FlagSet) string {
	// Restore the original output sink when returning.
	defer func(w io.Writer) {
		fs.SetOutput(w)
	}(fs.Output())
	// Capture the output from methods on the flag-set.
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.PrintDefaults()
	return buf.String()
}

// A Parser is a source of parameter values that plugs into [Parse]. Multiple
// parsers populate a shared [flag.FlagSet] in priority order; earlier parsers
// take precedence over later ones for any given flag.
//
// Use [CheckParser] to verify compliance with the priority contract.
//
// Implementations should also implement [fmt.Stringer] for meaningful
// diagnostics from [Parse] (e.g. in errors, logs, etc.).
type Parser interface {
	// ParseParameters populates flags in fs from the parser's underlying source
	// (environment variables, a config file, command-line args, etc.).
	//
	// The flag-set may already contain flags set by sources with higher-priority:
	// implementations MUST always check with [IsSet] before setting a flag.
	ParseParameters(fs *flag.FlagSet) error
}

// IsSet reports whether the named flag has been set. A [Parser] must check this
// before calling Set on a flag to respect the priority contract expected by
// [Parse].
//
// Must not be called concurrently with any other flag-set methods.
func IsSet(fs *flag.FlagSet, name string) bool {
	var found bool
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// PendingParams returns an iterator over registered flags that have not yet been
// set.
//
// Must not be called concurrently with any other flag-set methods.
func PendingParams(fs *flag.FlagSet) iter.Seq[*flag.Flag] {
	return func(yield func(*flag.Flag) bool) {
		more := true
		fs.VisitAll(func(f *flag.Flag) {
			if more && !IsSet(fs, f.Name) {
				more = yield(f)
			}
		})
	}
}
