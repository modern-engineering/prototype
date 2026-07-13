package parameter

import (
	"flag"
)

// Require marks the named flag as required, so hosts refuse to run the
// application while it stays unbound. Marking is additive-only and
// tolerant: see the no-op cases below, pinned by the capability table
// in parameter_test.
func Require(fs *flag.FlagSet, name string) {
	f := fs.Lookup(name)
	if f == nil {
		// An unregistered name is a silent no-op — the permissive
		// prototype default, mirroring IsRequiredFlag's false for the
		// same name. Door: panic (or a Must variant) the first time a
		// misspelled marker burns a catalogue author.
		return
	}
	if _, ok := f.Value.(RequiredFlag); ok {
		// A value that already answers the capability keeps its own
		// verdict; wrapping would drown an explicit false. Same door
		// as above if the silence ever hides a real conflict.
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
