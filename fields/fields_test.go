package fields_test

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/fields"
	"github.com/modern-engineering/prototype/program"
)

// Declaring a surface cannot evaluate a function or replace constructor state.
// It must expose custom transfer capabilities on the original handles, with
// copied metadata and a consistent declaration order for lookup and enumeration.
func TestDeclarationsPreserveBoundValues(t *testing.T) {
	var inputs fields.Inputs
	var outputs fields.Outputs
	var in program.Inputs = &inputs
	var out program.Outputs = &outputs
	if got := slices.Collect(in.All()); len(got) != 0 {
		t.Fatalf("empty inputs=%v", got)
	}
	if got := slices.Collect(out.All()); len(got) != 0 {
		t.Fatalf("empty outputs=%v", got)
	}
	if _, ok := in.Lookup("missing"); ok {
		t.Fatal("empty inputs contain a field")
	}
	if _, ok := out.Lookup("missing"); ok {
		t.Fatal("empty outputs contain a field")
	}

	allInputs, allOutputs := in.All(), out.All()
	body := []byte("constructor default")
	value := bind.Bytes(&body)
	declaration := program.Field{Name: "body", Help: "message bytes", Sensitive: true}
	inputs.Define(declaration, value)
	outputs.Define(declaration, value)
	declaration.Name, declaration.Sensitive = "changed", false

	var assigned []string
	gets := 0
	inputs.Var(program.SetterFunc(func(text string) error {
		assigned = append(assigned, text)
		return nil
	}), "record", "record each assignment")
	outputs.Var(program.GetterFunc(func() string {
		gets++
		return "computed"
	}), "computed", "read on demand")
	if string(body) != "constructor default" || len(assigned) != 0 || gets != 0 {
		t.Fatal("declaration evaluated or changed a value")
	}

	inputFields, outputFields := slices.Collect(allInputs), slices.Collect(allOutputs)
	if len(inputFields) != 2 || inputFields[0].Name != "body" || inputFields[1].Name != "record" {
		t.Fatalf("input order=%v", inputFields)
	}
	if len(outputFields) != 2 || outputFields[0].Name != "body" || outputFields[1].Name != "computed" {
		t.Fatalf("output order=%v", outputFields)
	}
	input, ok := in.Lookup("body")
	if !ok || input.Value != value || input.Field != inputFields[0].Field {
		t.Fatalf("input lookup=%v,%v", input, ok)
	}
	output, ok := out.Lookup("body")
	if !ok || output.Value != value || output.Field != outputFields[0].Field {
		t.Fatalf("output lookup=%v,%v", output, ok)
	}
	if !input.Sensitive || !output.Sensitive || input.Help != "message bytes" {
		t.Fatal("declaration metadata was not copied")
	}
	inputFields[0].Name, outputFields[0].Sensitive = "modified", false
	if field, _ := in.Lookup("body"); field.Name != "body" {
		t.Fatal("enumeration changed input metadata")
	}
	if field, _ := out.Lookup("body"); !field.Sensitive {
		t.Fatal("enumeration changed output metadata")
	}

	reader, ok := input.Value.(io.ReaderFrom)
	if !ok {
		t.Fatal("input lost io.ReaderFrom")
	}
	if _, err := reader.ReadFrom(strings.NewReader("loaded")); err != nil {
		t.Fatal(err)
	}
	writer, ok := output.Value.(io.WriterTo)
	if !ok {
		t.Fatal("output lost io.WriterTo")
	}
	var written bytes.Buffer
	if _, err := writer.WriteTo(&written); err != nil {
		t.Fatal(err)
	}
	if string(body) != "loaded" || written.String() != "loaded" {
		t.Fatal("rich transfer did not use bound state")
	}

	record, _ := in.Lookup("record")
	if err := record.Value.Set("first"); err != nil {
		t.Fatal(err)
	}
	if err := record.Value.Set("second"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(assigned, []string{"first", "second"}) {
		t.Fatalf("assignments=%v", assigned)
	}
	computed, _ := out.Lookup("computed")
	if got := computed.Value.Get(); got != "computed" || gets != 1 {
		t.Fatalf("computed=%q calls=%d", got, gets)
	}
	visitedInputs, visitedOutputs := 0, 0
	for range in.All() {
		visitedInputs++
		break
	}
	for range out.All() {
		visitedOutputs++
		break
	}
	if visitedInputs != 1 || visitedOutputs != 1 {
		t.Fatal("enumeration did not stop early")
	}
}

// Invalid declarations fail before replacing an existing field or leaving a
// partially registered field visible to loaders.
func TestInvalidDeclarationsLeaveSurfacesUntouched(t *testing.T) {
	for _, kind := range []string{"duplicate input", "duplicate output", "nil input", "nil output", "nil input target", "nil output target"} {
		t.Run(kind, func(t *testing.T) {
			var inputValue, outputValue string
			input, output := bind.String(&inputValue), bind.String(&outputValue)
			var inputs fields.Inputs
			var outputs fields.Outputs
			inputs.Var(input, "value", "original input")
			outputs.Var(output, "value", "original output")
			defer func() {
				if recover() == nil {
					t.Fatal("invalid declaration did not panic")
				}
				gotInput, _ := inputs.Lookup("value")
				gotOutput, _ := outputs.Lookup("value")
				if gotInput.Value != input || gotOutput.Value != output {
					t.Fatal("invalid declaration replaced a handle")
				}
				if len(slices.Collect(inputs.All())) != 1 || len(slices.Collect(outputs.All())) != 1 {
					t.Fatal("invalid declaration changed the surfaces")
				}
			}()
			switch kind {
			case "duplicate input":
				inputs.Var(bind.String(new(string)), "value", "duplicate")
			case "duplicate output":
				outputs.Var(bind.String(new(string)), "value", "duplicate")
			case "nil input":
				inputs.Var(nil, "new", "nil")
			case "nil output":
				outputs.Var(nil, "new", "nil")
			case "nil input target":
				inputs.StringVar(nil, "new", "nil")
			case "nil output target":
				outputs.BoolVar(nil, "new", "nil")
			}
		})
	}
}
