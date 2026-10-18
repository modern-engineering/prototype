package input_test

import (
	"testing"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
)

// Numeric declarations retain the caller's chosen types and initial values.
// Input and output declarations may share storage, including storage allocated
// by the helpers, so loading and direct assignments must remain visible to both.
func TestNumericSurfacesShareTypedStorage(t *testing.T) {
	type count int8
	type quota uint16
	type fraction float32
	type phasor complex128

	var inputs input.List
	var outputs output.List
	limit := inputs.Int("limit", quota(200), "maximum items")
	outputs.IntVar(limit, "limit", "maximum items")
	offset := outputs.Int("offset", count(-3), "item offset")
	inputs.IntVar(offset, "offset", "item offset")
	ratio := inputs.Float("ratio", fraction(0.5), "sampling fraction")
	outputs.FloatVar(ratio, "ratio", "sampling fraction")
	score := outputs.Float("score", 1.25, "weight")
	inputs.FloatVar(score, "score", "weight")
	phase := inputs.Complex("phase", complex64(1+2i), "phase")
	outputs.ComplexVar(phase, "phase", "phase")
	gain := outputs.Complex("gain", phasor(3-4i), "gain")
	inputs.ComplexVar(gain, "gain", "gain")

	for _, tc := range []struct {
		name, initial, loaded string
	}{
		{"limit", "200", "257"},
		{"offset", "-3", "-7"},
		{"ratio", "0.5", "0.75"},
		{"score", "1.25", "2.5"},
		{"phase", "(1+2i)", "(2+3i)"},
		{"gain", "(3-4i)", "(4-5i)"},
	} {
		input, inputExists := inputs.Lookup(tc.name)
		output, outputExists := outputs.Lookup(tc.name)
		if !inputExists || !outputExists {
			t.Fatalf("%s: input exists=%v, output exists=%v", tc.name, inputExists, outputExists)
		}
		if got := output.Value.Get(); got != tc.initial {
			t.Fatalf("%s: initial value=%q, want %q", tc.name, got, tc.initial)
		}
		if err := input.Value.Set(tc.loaded); err != nil {
			t.Fatalf("%s: Set: %v", tc.name, err)
		}
		if got := output.Value.Get(); got != tc.loaded {
			t.Errorf("%s: loaded value=%q, want %q", tc.name, got, tc.loaded)
		}
	}
	if *limit != 257 || *offset != -7 || *ratio != 0.75 || *score != 2.5 || *phase != 2+3i || *gain != 4-5i {
		t.Fatal("loading did not update the returned storage")
	}

	*offset = 12
	output, _ := outputs.Lookup("offset")
	if got := output.Value.Get(); got != "12" {
		t.Fatalf("direct assignment produced %q, want 12", got)
	}
	input, _ := inputs.Lookup("offset")
	if err := input.Value.Set("128"); err == nil {
		t.Fatal("out-of-range int8 input succeeded")
	}
	if *offset != 12 {
		t.Fatalf("failed parsing changed offset to %d", *offset)
	}
	input, _ = inputs.Lookup("limit")
	if err := input.Value.Set("-1"); err == nil {
		t.Fatal("negative unsigned input succeeded")
	}
	if *limit != 257 {
		t.Fatalf("failed parsing changed limit to %d", *limit)
	}
}
