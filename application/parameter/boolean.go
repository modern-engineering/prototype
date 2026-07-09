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

// IsBooleanFlag reports whether the named flag may be set without a
// value. An unregistered name is simply not boolean — tolerated the
// same way SetBoolean tolerates it, rather than a panic on the nil
// lookup.
func IsBooleanFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	return IsBoolean(f.Value)
}

func IsBoolean(v flag.Value) bool {
	if bf, ok := v.(BoolFlag); ok {
		return bf.IsBoolFlag()
	}
	return false
}
