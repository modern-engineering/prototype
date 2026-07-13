package parameter

import (
	"flag"
)

// SetBoolean marks the named flag as boolean, so it may be set without
// a value. Marking is additive-only and tolerant: see the no-op cases
// below, pinned by the capability table in parameter_test.
func SetBoolean(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		// An unregistered name is a silent no-op — the permissive
		// prototype default, mirroring IsBooleanFlag's false for the
		// same name. Door: panic (or a Must variant) the first time a
		// misspelled marker burns a catalogue author.
		return
	}
	if _, ok := f.Value.(BoolFlag); ok {
		// A value that already answers the capability keeps its own
		// verdict; wrapping would drown an explicit false. Same door
		// as above if the silence ever hides a real conflict.
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
