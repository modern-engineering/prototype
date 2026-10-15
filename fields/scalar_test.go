package fields_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/fields"
)

// Allocating and variable helpers must refer to the same ordinary Go state on
// both surfaces. Declaration preserves defaults; loader assignments and runner
// assignments remain visible through the handles, including byte transfers.
func TestTypedDeclarationsShareGoState(t *testing.T) {
	var inputs fields.Inputs
	var outputs fields.Outputs
	name := inputs.String("name", "world", "recipient")
	outputs.StringVar(name, "name", "recipient")
	enabled := outputs.Bool("enabled", true, "delivery enabled")
	inputs.BoolVar(enabled, "enabled", "delivery enabled")
	body := inputs.Bytes("body", []byte("draft"), "message")
	outputs.BytesVar(body, "body", "message")

	for _, test := range []struct{ name, initial, assigned string }{
		{"name", "world", "Ada"},
		{"enabled", "true", "false"},
		{"body", "draft", "ready"},
	} {
		output, ok := outputs.Lookup(test.name)
		if !ok || output.Value.Get() != test.initial {
			t.Fatalf("%s did not retain its initial value: %v", test.name, output)
		}
		input, ok := inputs.Lookup(test.name)
		if !ok {
			t.Fatalf("input %s is missing", test.name)
		}
		if err := input.Value.Set(test.assigned); err != nil {
			t.Fatal(err)
		}
		if got := output.Value.Get(); got != test.assigned {
			t.Errorf("%s = %q, want %q", test.name, got, test.assigned)
		}
	}
	if *name != "Ada" || *enabled || string(*body) != "ready" {
		t.Fatalf("bound variables = %q, %v, %q", *name, *enabled, *body)
	}
	*name = "Grace"
	if output, _ := outputs.Lookup("name"); output.Value.Get() != "Grace" {
		t.Fatal("getter did not see direct assignment")
	}

	input, _ := inputs.Lookup("body")
	if _, err := input.Value.(io.ReaderFrom).ReadFrom(strings.NewReader("loaded")); err != nil {
		t.Fatal(err)
	}
	output, _ := outputs.Lookup("body")
	var written bytes.Buffer
	if _, err := output.Value.(io.WriterTo).WriteTo(&written); err != nil {
		t.Fatal(err)
	}
	if written.String() != "loaded" || string(*body) != "loaded" {
		t.Fatalf("byte transfer = %q, bound bytes = %q", written.String(), *body)
	}
}
