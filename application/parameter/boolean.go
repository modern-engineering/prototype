package parameter

import (
	"flag"
)

func SetBoolean(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		// TODO: maybe panic instead of silently ignoring?
		return
	}
	if _, ok := f.Value.(BoolFlag); ok {
		// TODO: maybe panic instead of silently ignoring?
		return
	}
	f.Value = boolValue{Value: f.Value}
}

type boolValue struct {
	flag.Value
}

func (boolValue) IsBoolFlag() bool {
	return true
}

type BoolFlag interface {
	IsBoolFlag() bool
}

func IsBooleanFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	return IsBoolean(f.Value)
}

func IsBoolean(v flag.Value) bool {
	if bf, ok := v.(BoolFlag); ok {
		return bf.IsBoolFlag()
	}
	return false
}

// TODO: test IsBooleanFlag with non-existing flag, existing flag without BoolFlag, and existing flag with BoolFlag (set both true and false).
// TODO: test IsBoolean with nil value, value without IsRequiredFlag, and value with IsRequiredFlag (set both true and false).
