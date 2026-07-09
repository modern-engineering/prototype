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

func TestValueJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value *image.Value
		json  string
	}{
		{"string", image.String("com.acme.Echo"), `{"kind":"string","string":"com.acme.Echo"}`},
		{"string empty", image.String(""), `{"kind":"string","string":""}`},
		{"string quoted", image.String(`say "hi"`), `{"kind":"string","string":"say \"hi\""}`},
		{"int", image.Int(-42), `{"kind":"int","int":-42}`},
		{"int zero", image.Int(0), `{"kind":"int","int":0}`},
		{"int max", image.Int(math.MaxInt64), `{"kind":"int","int":9223372036854775807}`},
		{"bool true", image.Bool(true), `{"kind":"bool","bool":true}`},
		{"bool false", image.Bool(false), `{"kind":"bool","bool":false}`},
		{"duration", image.Duration(1500 * time.Millisecond), `{"kind":"duration","duration":"1.5s"}`},
		{"duration negative", image.Duration(-90 * time.Second), `{"kind":"duration","duration":"-1m30s"}`},
		{"duration zero", image.Duration(0), `{"kind":"duration","duration":"0s"}`},
		{"duration max", image.Duration(math.MaxInt64), `{"kind":"duration","duration":"2562047h47m16.854775807s"}`},
		{"duration min", image.Duration(math.MinInt64), `{"kind":"duration","duration":"-2562047h47m16.854775808s"}`},
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
	_, err := json.Marshal(&image.Value{Kind: "token"})
	if err == nil || !strings.Contains(err.Error(), `unknown value kind "token"`) {
		t.Errorf("Marshal unknown kind: err = %v, want unknown-kind error", err)
	}
}

func TestValueUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string // substring of the error
	}{
		{"unknown kind", `{"kind":"token","token":"x"}`, `unknown value kind "token"`},
		{"missing string arm", `{"kind":"string"}`, `missing its "string" field`},
		{"missing int arm", `{"kind":"int"}`, `missing its "int" field`},
		{"missing bool arm", `{"kind":"bool"}`, `missing its "bool" field`},
		{"missing duration arm", `{"kind":"duration"}`, `missing its "duration" field`},
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
