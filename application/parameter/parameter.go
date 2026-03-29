package parameter

import (
	"flag"
	"fmt"
)

// TODO: explain why a uniform name.
const flagsetName = "<PARAMETERS>"

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(flagsetName, flag.ContinueOnError)
	fs.Usage = usageOf(fs, name)
	return fs
}

func usageOf(fs *flag.FlagSet, name string) func() {
	return func() {
		_, _ = fmt.Fprintf(fs.Output(), "Parameterization of %s:\n", name)
		fs.PrintDefaults()
	}
}

func Parse(fs *flag.FlagSet, sources ...Parser) error {
	// TODO: explain why we reject non-parameter flag-sets. What's so special about them? Why can't we just accept any flag-set?
	// TODO: document that this function panics if called on a non-parameter flag-set.
	if fs.Name() != flagsetName {
		panic("parameter.Parse called on non-parameter flag-set")
	}

	if fs.Parsed() {
		// TODO: should we panic instead of exiting?
		return fmt.Errorf("already parsed")
	}

	for _, source := range sources {
		err := source.ParseParameters(fs)
		if err != nil {
			return err
		}
	}

	// TODO: explain why call Parse even though we don't use the 'arguments'; (to set Parsed).
	err := fs.Parse(nil)
	if err != nil {
		return err
	}
	return nil
}

type Parser interface {
	ParseParameters(fs *flag.FlagSet) error
}
