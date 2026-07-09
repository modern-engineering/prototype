package parameter

import (
	"flag"
)

func Require(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		// TODO: maybe panic instead of silently ignoring?
		return
	}
	if _, ok := f.Value.(RequiredFlag); ok {
		// TODO: maybe panic instead of silently ignoring?
		return
	}
	f.Value = requiredValue{Value: f.Value}
}

type requiredValue struct {
	flag.Value
}

func (requiredValue) IsRequiredFlag() bool {
	return true
}

type RequiredFlag interface {
	IsRequiredFlag() bool
}

// IsRequiredFlag reports whether the named flag must be set before the
// application runs. An unregistered name is simply not required —
// tolerated the same way Require tolerates it, rather than a panic on
// the nil lookup.
func IsRequiredFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	return IsRequired(f.Value)
}

func IsRequired(v flag.Value) bool {
	if rf, ok := v.(RequiredFlag); ok {
		return rf.IsRequiredFlag()
	}
	return false
}
