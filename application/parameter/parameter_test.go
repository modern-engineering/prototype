// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package parameter_test

import (
	"flag"
	"testing"

	"github.com/modern-engineering/prototype/application/parameter"
)

// A stringValue is a minimal flag.Value carrying no capability
// interfaces at all.
type stringValue string

func (v *stringValue) String() string     { return string(*v) }
func (v *stringValue) Set(s string) error { *v = stringValue(s); return nil }

// A boolCapable answers IsBoolFlag with a configured verdict, proving
// the predicates read the answer rather than the interface's presence.
type boolCapable struct {
	stringValue
	answer bool
}

func (v *boolCapable) IsBoolFlag() bool { return v.answer }

// A requiredCapable is boolCapable's twin for the required marker.
type requiredCapable struct {
	stringValue
	answer bool
}

func (v *requiredCapable) IsRequiredFlag() bool { return v.answer }

func TestIsBooleanFlag(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(new(stringValue), "plain", "no capability interfaces")
	fs.Var(&boolCapable{answer: true}, "yes", "IsBoolFlag says true")
	fs.Var(&boolCapable{answer: false}, "no", "IsBoolFlag says false")
	fs.Var(new(stringValue), "marked", "boolean by SetBoolean")
	parameter.SetBoolean(fs, "marked")
	parameter.SetBoolean(fs, "absent") // tolerated: nothing to mark

	tests := []struct {
		name string
		want bool
	}{
		{"absent", false}, // unregistered names are simply not boolean
		{"plain", false},
		{"yes", true},
		{"no", false},
		{"marked", true},
	}
	for _, tt := range tests {
		if got := parameter.IsBooleanFlag(fs, tt.name); got != tt.want {
			t.Errorf("IsBooleanFlag(fs, %q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsBoolean(t *testing.T) {
	tests := []struct {
		name string
		v    flag.Value
		want bool
	}{
		{"nil value", nil, false},
		{"no capability", new(stringValue), false},
		{"IsBoolFlag true", &boolCapable{answer: true}, true},
		{"IsBoolFlag false", &boolCapable{answer: false}, false},
	}
	for _, tt := range tests {
		if got := parameter.IsBoolean(tt.v); got != tt.want {
			t.Errorf("%s: IsBoolean = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsRequiredFlag(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(new(stringValue), "plain", "no capability interfaces")
	fs.Var(&requiredCapable{answer: true}, "yes", "IsRequiredFlag says true")
	fs.Var(&requiredCapable{answer: false}, "no", "IsRequiredFlag says false")
	fs.Var(new(stringValue), "marked", "required by Require")
	parameter.Require(fs, "marked")
	parameter.Require(fs, "absent") // tolerated: nothing to mark

	tests := []struct {
		name string
		want bool
	}{
		{"absent", false}, // unregistered names are simply not required
		{"plain", false},
		{"yes", true},
		{"no", false},
		{"marked", true},
	}
	for _, tt := range tests {
		if got := parameter.IsRequiredFlag(fs, tt.name); got != tt.want {
			t.Errorf("IsRequiredFlag(fs, %q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsRequired(t *testing.T) {
	tests := []struct {
		name string
		v    flag.Value
		want bool
	}{
		{"nil value", nil, false},
		{"no capability", new(stringValue), false},
		{"IsRequiredFlag true", &requiredCapable{answer: true}, true},
		{"IsRequiredFlag false", &requiredCapable{answer: false}, false},
	}
	for _, tt := range tests {
		if got := parameter.IsRequired(tt.v); got != tt.want {
			t.Errorf("%s: IsRequired = %v, want %v", tt.name, got, tt.want)
		}
	}
}
