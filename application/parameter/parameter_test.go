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

// A capable value answers both capability interfaces itself, with a
// configured verdict, proving the predicates read the answer rather
// than the interface's mere presence.
type capable struct {
	stringValue
	answer bool
}

func (v *capable) IsBoolFlag() bool     { return v.answer }
func (v *capable) IsRequiredFlag() bool { return v.answer }

// The package's two capabilities — boolean-ness and required-ness —
// share one mechanism, exercised here through every doorway at once: a
// value may answer for itself (either way), a marker upgrades a bare
// value, and everything else is a deliberate silent no-op. The no-ops
// pin the permissive prototype default the markers document: marking an
// unregistered name does nothing, and marking a flag whose value
// already answers keeps that value's own verdict — even an explicit
// false. The recorded door out of the silence is a panic (or a Must
// variant), triggered the first time it burns a catalogue author.
// Capabilities stay independent: granting one never grants the other.
func TestCapabilityMarkers(t *testing.T) {
	capabilities := []struct {
		name     string
		mark     func(*flag.FlagSet, string)
		ask      func(*flag.FlagSet, string) bool
		askValue func(flag.Value) bool
		sibling  func(*flag.FlagSet, string) bool
	}{
		{"boolean", parameter.SetBoolean, parameter.IsBooleanFlag, parameter.IsBoolean, parameter.IsRequiredFlag},
		{"required", parameter.Require, parameter.IsRequiredFlag, parameter.IsRequired, parameter.IsBooleanFlag},
	}

	for _, c := range capabilities {
		t.Run(c.name, func(t *testing.T) {
			fs := flag.NewFlagSet("t", flag.ContinueOnError)
			fs.Var(new(stringValue), "plain", "no capability interfaces")
			fs.Var(&capable{answer: true}, "yes", "answers true itself")
			fs.Var(&capable{answer: false}, "no", "answers false itself")
			fs.Var(new(stringValue), "marked", "bare until marked")
			c.mark(fs, "marked")
			c.mark(fs, "absent") // no such flag: the tolerated no-op
			c.mark(fs, "no")     // already answering: its own false stands

			verdicts := []struct {
				flag string
				want bool
			}{
				{"plain", false},
				{"yes", true},
				{"no", false}, // marking must not overrule the value's answer
				{"marked", true},
				{"absent", false}, // unregistered names are simply not capable
			}
			for _, v := range verdicts {
				if got := c.ask(fs, v.flag); got != v.want {
					t.Errorf("%s(%q) = %v, want %v", c.name, v.flag, got, v.want)
				}
			}

			if c.sibling(fs, "marked") {
				t.Errorf("marking %q as %s granted the sibling capability too", "marked", c.name)
			}

			// The value-level predicates answer without any flag set.
			if c.askValue(nil) || c.askValue(new(stringValue)) {
				t.Errorf("%s predicate claimed capability for a nil or bare value", c.name)
			}
			if !c.askValue(&capable{answer: true}) || c.askValue(&capable{answer: false}) {
				t.Errorf("value-level %s predicate ignored the value's own answer", c.name)
			}
		})
	}
}
