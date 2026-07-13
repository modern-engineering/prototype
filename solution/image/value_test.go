// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// Every kind's wire form survives a marshal-unmarshal round trip at
// its edges — empty, quoted, zero, extreme, negative, fractional; the
// canonical spelling of each kind is ExampleValue's to pin.
func TestValueJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value *image.Value
		json  string
	}{
		{"string empty", image.String(""), `{"kind":"string","string":""}`},
		{"string quoted", image.String(`say "hi"`), `{"kind":"string","string":"say \"hi\""}`},
		{"int negative", image.Int(-42), `{"kind":"int","int":-42}`},
		{"int zero", image.Int(0), `{"kind":"int","int":0}`},
		{"int max", image.Int(math.MaxInt64), `{"kind":"int","int":9223372036854775807}`},
		{"bool false", image.Bool(false), `{"kind":"bool","bool":false}`},
		{"duration negative fractional", image.Duration(-90500 * time.Millisecond), `{"kind":"duration","duration":"-1m30.5s"}`},
		{"duration zero", image.Duration(0), `{"kind":"duration","duration":"0s"}`},
		{"duration max", image.Duration(math.MaxInt64), `{"kind":"duration","duration":"2562047h47m16.854775807s"}`},
		{"duration min", image.Duration(math.MinInt64), `{"kind":"duration","duration":"-2562047h47m16.854775808s"}`},
		{"token empty", image.Token(""), `{"kind":"token","token":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(data) != tt.json {
				t.Errorf("Marshal = %s, want %s", data, tt.json)
			}
			var got image.Value
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got != *tt.value {
				t.Errorf("round trip = %+v, want %+v", got, *tt.value)
			}
		})
	}
}

func TestValueMarshalUnknownKind(t *testing.T) {
	_, err := json.Marshal(&image.Value{Kind: "float"})
	if err == nil || !strings.Contains(err.Error(), `unknown value kind "float"`) {
		t.Errorf("Marshal unknown kind: err = %v, want unknown-kind error", err)
	}
}

func TestValueUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string // substring of the error
	}{
		{"unknown kind", `{"kind":"float","float":1.5}`, `unknown value kind "float"`},
		{"missing string arm", `{"kind":"string"}`, `missing its "string" field`},
		{"missing int arm", `{"kind":"int"}`, `missing its "int" field`},
		{"missing bool arm", `{"kind":"bool"}`, `missing its "bool" field`},
		{"missing duration arm", `{"kind":"duration"}`, `missing its "duration" field`},
		{"missing token arm", `{"kind":"token"}`, `missing its "token" field`},
		{"mismatched arm", `{"kind":"int","string":"5"}`, `missing its "int" field`},
		{"bad duration", `{"kind":"duration","duration":"soon"}`, `invalid duration value "soon"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v image.Value
			err := json.Unmarshal([]byte(tt.json), &v)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Unmarshal(%s): err = %v, want %q", tt.json, err, tt.want)
			}
		})
	}
}
