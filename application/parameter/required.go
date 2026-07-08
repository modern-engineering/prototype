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

func IsRequiredFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	return IsRequired(f.Value)
}

func IsRequired(v flag.Value) bool {
	if rf, ok := v.(RequiredFlag); ok {
		return rf.IsRequiredFlag()
	}
	return false
}

// TODO: test IsRequiredFlag with non-existing flag, existing flag without RequiredFlag, and existing flag with RequiredFlag (set both true and false).
// TODO: test IsRequired with nil value, value without IsRequiredFlag, and value with IsRequiredFlag (set both true and false).
