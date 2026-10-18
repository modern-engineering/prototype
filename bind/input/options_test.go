package input_test

import (
	"io"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
)

// Metadata options must survive the convenience layers without changing
// constructor state or hiding capabilities used by a loader.
func TestOptionsPreserveBoundValues(t *testing.T) {
	type count uint16
	type ratio float32
	type phase complex64
	var inputs input.List
	var outputs output.List
	token := inputs.String("token", "initial", "credential", input.Sensitive)
	outputs.StringVar(token, "token", "credential", output.Sensitive)
	valid := outputs.Bool("valid", true, "credential status", output.Sensitive)
	inputs.BoolVar(valid, "valid", "credential status", input.Sensitive)
	body := inputs.Bytes("body", []byte("draft"), "message", input.Sensitive)
	outputs.BytesVar(body, "body", "message", output.Sensitive)
	limit := inputs.Int("limit", count(4), "attempts", input.Sensitive)
	outputs.IntVar(limit, "limit", "attempts", output.Sensitive)
	weight := outputs.Float("weight", ratio(0.5), "weight", output.Sensitive)
	inputs.FloatVar(weight, "weight", "weight", input.Sensitive)
	position := inputs.Complex("position", phase(1+2i), "position", input.Sensitive)
	outputs.ComplexVar(position, "position", "position", output.Sensitive)

	for field := range inputs.All() {
		if !field.Sensitive {
			t.Errorf("input %q lost Sensitive", field.Name)
		}
	}
	for field := range outputs.All() {
		if !field.Sensitive {
			t.Errorf("output %q lost Sensitive", field.Name)
		}
	}
	if *token != "initial" || !*valid || string(*body) != "draft" || *limit != 4 || *weight != 0.5 || *position != 1+2i {
		t.Fatal("options changed initial values")
	}

	custom := bind.Bytes(body)
	inputs.Var(custom, "raw", "original bytes", input.Sensitive)
	outputs.Var(custom, "raw", "original bytes", output.Sensitive)
	in, _ := inputs.Lookup("raw")
	out, _ := outputs.Lookup("raw")
	if !in.Sensitive || !out.Sensitive || in.Value != custom || out.Value != custom {
		t.Fatal("options changed the custom handle or lost metadata")
	}
	if _, ok := in.Value.(io.ReaderFrom); !ok {
		t.Fatal("input lost io.ReaderFrom")
	}
	if _, ok := out.Value.(io.WriterTo); !ok {
		t.Fatal("output lost io.WriterTo")
	}
	inputs.Var(program.SetterFunc(func(string) error {
		t.Fatal("registration called Set")
		return nil
	}), "write", "write on demand", input.Sensitive)
	outputs.Var(program.GetterFunc(func() string {
		t.Fatal("registration called Get")
		return ""
	}), "read", "read on demand", output.Sensitive)
}

// Options compose in source order, and retained option arguments cannot mutate
// the copy of metadata already exposed to loaders.
func TestOptionsProduceCopiedDeclarations(t *testing.T) {
	var inputs input.List
	var outputs output.List
	var retained *program.Field
	customize := func(field *program.Field) {
		if !field.Sensitive {
			t.Fatal("options did not run in order")
		}
		field.Name, field.Help, field.Sensitive = "final", "final help", false
		retained = field
	}
	inputs.String("original", "input", "original help", input.Sensitive, customize)
	*retained = program.Field{}
	outputs.String("original", "output", "original help", output.Sensitive, customize)
	*retained = program.Field{}
	in, inOK := inputs.Lookup("final")
	out, outOK := outputs.Lookup("final")
	want := program.Field{Name: "final", Help: "final help"}
	if !inOK || !outOK || in.Field != want || out.Field != want {
		t.Fatalf("final declarations = %v, %v; want %v", in.Field, out.Field, want)
	}
}

// Registration must validate the final metadata and cannot expose a partial
// declaration when an option panics or is nil.
func TestInvalidOptionsLeaveDeclarationsUntouched(t *testing.T) {
	duplicate := func(field *program.Field) { field.Name = "existing" }
	fail := func(*program.Field) { panic("invalid metadata") }
	for _, test := range []struct {
		name     string
		register func(*input.List, *output.List)
	}{
		{"NilInput", func(in *input.List, _ *output.List) { in.String("new", "value", "", nil) }},
		{"NilOutput", func(_ *input.List, out *output.List) { out.String("new", "value", "", nil) }},
		{"PanicInput", func(in *input.List, _ *output.List) { in.String("new", "value", "", input.Sensitive, fail) }},
		{"PanicOutput", func(_ *input.List, out *output.List) { out.String("new", "value", "", output.Sensitive, fail) }},
		{"DuplicateInput", func(in *input.List, _ *output.List) { in.String("new", "value", "", duplicate) }},
		{"DuplicateOutput", func(_ *input.List, out *output.List) { out.String("new", "value", "", duplicate) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var inputs input.List
			var outputs output.List
			value := "original"
			handle := bind.String(&value)
			inputs.Var(handle, "existing", "original help", input.Sensitive)
			outputs.Var(handle, "existing", "original help", output.Sensitive)
			defer func() {
				if recover() == nil {
					t.Fatal("invalid option did not panic")
				}
				in := slices.Collect(inputs.All())
				out := slices.Collect(outputs.All())
				if len(in) != 1 || len(out) != 1 {
					t.Fatalf("declaration counts = %d, %d; want 1, 1", len(in), len(out))
				}
				want := program.Field{Name: "existing", Help: "original help", Sensitive: true}
				if in[0].Field != want || out[0].Field != want || in[0].Value != handle || out[0].Value != handle || value != "original" {
					t.Fatal("invalid option changed an existing declaration")
				}
			}()
			test.register(&inputs, &outputs)
		})
	}
}
