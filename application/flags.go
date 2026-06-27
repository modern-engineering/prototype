package application

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"iter"
	"os"
	"strings"
)

//type Maker interface {
//	Make(context.Context) (Service, error)
//}
//
//func MakeFromArgs(next Maker, args []string) func(context.Context) (Service, error) {
//	return func(ctx context.Context) (Service, error) {
//		svc, err := next.Make(ctx)
//		if err != nil {
//			return nil, err
//		}
//		parser := argsParser{fs: svc.Flags()}
//		if err := parser.Parse(args); err != nil {
//			return nil, fmt.Errorf("parse args: %w", err)
//		}
//		return svc, nil
//	}
//}

func ParseFromArgs(next ParamParser, args []string) ParseTo {
	return func(ctx context.Context, flags *flag.FlagSet) error {
		parser := argsParser{fs: flags}
		if err := parser.Parse(args); err != nil {
			return fmt.Errorf("parse args: %w", err)
		}
		return next.Parse(ctx, flags)
	}
}

type argsParser struct {
	fs   *flag.FlagSet
	args []string
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the [FlagSet]
// are defined and before flags are accessed by the program.
// The return value will be [ErrHelp] if -help or -h were set but not defined.
func (p *argsParser) Parse(arguments []string) error {
	p.args = arguments
	for {
		seen, err := p.parseOne()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		return err
	}
	return nil
}

// parseOne parses one flag. It reports whether a flag was seen.
func (p *argsParser) parseOne() (bool, error) {
	if len(p.args) == 0 {
		return false, nil
	}
	s := p.args[0]
	if len(s) < 2 || s[0] != '-' {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			p.args = p.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, fmt.Errorf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument?
	p.args = p.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	if p.fs.Lookup(name) == nil {
		if name == "help" || name == "h" { // special case for nice help message.
			return false, flag.ErrHelp
		}
		return false, fmt.Errorf("flag provided but not defined: -%s", name)
	}

	if isBoolFlag(p.fs, name) { // special case: doesn't need an arg
		if hasValue {
			if err := p.fs.Set(name, value); err != nil {
				return false, fmt.Errorf("invalid boolean value %q for -%s: %v", value, name, err)
			}
		} else {
			if err := p.fs.Set(name, "true"); err != nil {
				return false, fmt.Errorf("invalid boolean flag %s: %v", name, err)
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		if !hasValue && len(p.args) > 0 {
			// value is the next arg
			hasValue = true
			value, p.args = p.args[0], p.args[1:]
		}
		if !hasValue {
			return false, fmt.Errorf("flag needs an argument: -%s", name)
		}
		if err := p.fs.Set(name, value); err != nil {
			return false, fmt.Errorf("invalid value %q for flag -%s: %v", value, name, err)
		}
	}
	return true, nil
}

func isBoolFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	if fv, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
		return fv.IsBoolFlag()
	}
	return false
}
func ParseFromEnvs(next ParamParser) ParseTo {
	return func(ctx context.Context, flags *flag.FlagSet) error {
		var errs error
		flags.VisitAll(func(f *flag.Flag) {
			varName := flagEnvVar(f)
			s, ok := os.LookupEnv(varName)
			if ok {
				if err := flags.Set(f.Name, s); err != nil {
					errs = errors.Join(errs, fmt.Errorf("set %q: %w", f.Name, err))
				}
			}
		})
		if errs != nil {
			return fmt.Errorf("set flags from envs: %w", errs)
		}
		return next.Parse(ctx, flags)
	}
}

func iterFlags(fs *flag.FlagSet) iter.Seq[*flag.Flag] {
	return func(yield func(*flag.Flag) bool) {
		more := true
		fs.VisitAll(func(f *flag.Flag) {
			if more {
				more = yield(f)
			}
		})
	}
}

func flagEnvVar(f *flag.Flag) string {
	var envSafeChar = func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z':
			return 'A' + r - 'a'
		case 'A' <= r && r <= 'Z':
			return r
		case '0' <= r && r <= '9':
			return r
		default:
			return '_'
		}
	}
	return strings.Map(envSafeChar, f.Name)
}
