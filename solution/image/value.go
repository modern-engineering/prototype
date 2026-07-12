// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

// The value kinds a binding may carry: the SDL literal kinds, plus the
// opaque tokens of the deployment and extension compartments.
const (
	KindString   = "string"
	KindInt      = "int"
	KindBool     = "bool"
	KindDuration = "duration"

	// KindToken marks an opaque token: a bare identifier in a
	// top-level field or a with-stanza, typed by the platform's
	// deployment profile or the stanza's scheme and never resolved
	// against the solution's symbols. Tokens appear in
	// [Record].Deployment and [Record].Extensions bindings only.
	KindToken = "token"
)

// A Value is one typed literal: Kind selects which of the value arms is
// meaningful, and the others are ignored. Construct values through
// [String], [Int], [Bool], [Duration], and [Token] so only the selected
// arm is populated.
//
// The JSON form carries the kind and exactly one value field named after
// it:
//
//	{"kind":"string","string":"com.acme.Echo"}
//	{"kind":"int","int":-1}
//	{"kind":"bool","bool":true}
//	{"kind":"duration","duration":"1.5s"}
//	{"kind":"token","token":"euCentral1"}
//
// Durations render as their canonical Go string (time.Duration.String)
// rather than raw nanoseconds so images stay human-readable; the form
// round-trips losslessly through time.ParseDuration across the full
// int64 range.
type Value struct {
	Kind string
	Str  string
	Int  int64
	Bool bool
	Dur  time.Duration
	Tok  string
}

// String returns a value of kind [KindString] holding s.
func String(s string) *Value { return &Value{Kind: KindString, Str: s} }

// Int returns a value of kind [KindInt] holding i.
func Int(i int64) *Value { return &Value{Kind: KindInt, Int: i} }

// Bool returns a value of kind [KindBool] holding b.
func Bool(b bool) *Value { return &Value{Kind: KindBool, Bool: b} }

// Duration returns a value of kind [KindDuration] holding d.
func Duration(d time.Duration) *Value { return &Value{Kind: KindDuration, Dur: d} }

// Token returns a value of kind [KindToken] holding the profile token
// tok.
func Token(tok string) *Value { return &Value{Kind: KindToken, Tok: tok} }

// MarshalJSON implements json.Marshaler, emitting the kind and the one
// value field it selects. Unknown kinds are marshalling errors, and so
// is a string value that is not valid UTF-8: encoding/json would
// silently rewrite its invalid bytes as U+FFFD, so the image would
// decode to a different value than it was built from. The compiler's
// front end already rejects such literals; failing loudly here keeps
// any other producer honest.
func (v Value) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case KindString:
		if !utf8.ValidString(v.Str) {
			return nil, fmt.Errorf("image: string value %q is not valid UTF-8", v.Str)
		}
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Str  string `json:"string"`
		}{v.Kind, v.Str})
	case KindInt:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Int  int64  `json:"int"`
		}{v.Kind, v.Int})
	case KindBool:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Bool bool   `json:"bool"`
		}{v.Kind, v.Bool})
	case KindDuration:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Dur  string `json:"duration"`
		}{v.Kind, v.Dur.String()})
	case KindToken:
		return json.Marshal(struct {
			Kind string `json:"kind"`
			Tok  string `json:"token"`
		}{v.Kind, v.Tok})
	}
	return nil, fmt.Errorf("image: unknown value kind %q", v.Kind)
}

// UnmarshalJSON implements json.Unmarshaler. The field named by the kind
// must be present; durations parse back through time.ParseDuration, so a
// marshal-unmarshal round trip is lossless.
func (v *Value) UnmarshalJSON(data []byte) error {
	var raw struct {
		Kind string  `json:"kind"`
		Str  *string `json:"string"`
		Int  *int64  `json:"int"`
		Bool *bool   `json:"bool"`
		Dur  *string `json:"duration"`
		Tok  *string `json:"token"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*v = Value{Kind: raw.Kind}
	switch raw.Kind {
	case KindString:
		if raw.Str == nil {
			return missingArm(raw.Kind)
		}
		v.Str = *raw.Str
	case KindInt:
		if raw.Int == nil {
			return missingArm(raw.Kind)
		}
		v.Int = *raw.Int
	case KindBool:
		if raw.Bool == nil {
			return missingArm(raw.Kind)
		}
		v.Bool = *raw.Bool
	case KindDuration:
		if raw.Dur == nil {
			return missingArm(raw.Kind)
		}
		d, err := time.ParseDuration(*raw.Dur)
		if err != nil {
			return fmt.Errorf("image: invalid duration value %q: %v", *raw.Dur, err)
		}
		v.Dur = d
	case KindToken:
		if raw.Tok == nil {
			return missingArm(raw.Kind)
		}
		v.Tok = *raw.Tok
	default:
		return fmt.Errorf("image: unknown value kind %q", raw.Kind)
	}
	return nil
}

// missingArm reports a value document whose kind names an absent field.
func missingArm(kind string) error {
	return fmt.Errorf("image: value of kind %q is missing its %q field", kind, kind)
}
