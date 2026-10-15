package input_test

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
)

// A tagged configuration remains ordinary Go state. Both declaration surfaces
// must observe the same storage, including defined types and byte capabilities.
func TestJSONBindsLiveState(t *testing.T) {
	type token string
	type enabled bool
	type payload []byte
	cfg := struct {
		Token   token    `json:"token"`
		Enabled enabled  `json:"enabled"`
		Body    payload  `json:"body"`
		Ignored chan int `json:"-"`
		Private string   `json:"-"`
	}{Token: "initial", Enabled: true, Body: payload("draft"), Private: "internal"}
	var inputs input.List
	var outputs output.List
	inputs.String("first", "existing", "")
	outputs.String("first", "existing", "")
	metadata := program.Field{Name: "token", Help: "credential", Sensitive: true}
	inputs.JSON(&cfg, metadata)
	outputs.JSON(&cfg, metadata)
	var inNames, outNames []string
	for field := range inputs.All() {
		inNames = append(inNames, field.Name)
	}
	for field := range outputs.All() {
		outNames = append(outNames, field.Name)
	}
	wantNames := []string{"first", "token", "enabled", "body"}
	if !slices.Equal(inNames, wantNames) || !slices.Equal(outNames, wantNames) {
		t.Fatalf("field order = %v, %v; want %v", inNames, outNames, wantNames)
	}
	in, _ := inputs.Lookup("token")
	out, _ := outputs.Lookup("token")
	wantField := program.Field{Name: "token", Help: "credential", Sensitive: true}
	if in.Field != wantField || out.Field != wantField {
		t.Fatalf("token metadata = %v, %v; want %v", in.Field, out.Field, wantField)
	}
	if cfg.Token != "initial" || !cfg.Enabled || string(cfg.Body) != "draft" || cfg.Private != "internal" {
		t.Fatal("registration changed configuration")
	}
	if err := in.Value.Set("updated"); err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "updated" || out.Value.Get() != "updated" {
		t.Fatal("input and output do not share configuration storage")
	}
	cfg.Token = "direct"
	if got := out.Value.Get(); got != "direct" {
		t.Fatalf("output after direct assignment = %q; want direct", got)
	}
	flag, _ := inputs.Lookup("enabled")
	if err := flag.Value.Set("false"); err != nil || cfg.Enabled {
		t.Fatalf("Set(false) = %v, enabled = %v", err, cfg.Enabled)
	}
	bodyIn, _ := inputs.Lookup("body")
	bodyOut, _ := outputs.Lookup("body")
	reader, ok := bodyIn.Value.(io.ReaderFrom)
	if !ok {
		t.Fatal("byte input lost io.ReaderFrom")
	}
	if _, err := reader.ReadFrom(strings.NewReader("loaded")); err != nil {
		t.Fatal(err)
	}
	writer, ok := bodyOut.Value.(io.WriterTo)
	if !ok {
		t.Fatal("byte output lost io.WriterTo")
	}
	var buffer bytes.Buffer
	if _, err := writer.WriteTo(&buffer); err != nil {
		t.Fatal(err)
	}
	if string(cfg.Body) != "loaded" || buffer.String() != "loaded" {
		t.Fatalf("byte transfer = %q, %q; want loaded", cfg.Body, buffer.String())
	}
}

// Reflection must retain named numeric types and their width, rather than
// quietly converting the configuration into machine-sized numbers.
func TestJSONNumericWidths(t *testing.T) {
	type retries int8
	type port uint16
	type ratio float32
	type phase complex64
	cfg := struct {
		Retries retries `json:"retries"`
		Port    port    `json:"port"`
		Ratio   ratio   `json:"ratio"`
		Phase   phase   `json:"phase"`
	}{Retries: 3, Port: 80, Ratio: 0.5, Phase: 1 + 2i}
	var inputs input.List
	var outputs output.List
	inputs.JSON(&cfg)
	outputs.JSON(&cfg)
	for _, test := range []struct{ name, text, want, overflow string }{
		{name: "retries", text: "127", want: "127", overflow: "128"},
		{name: "port", text: "65535", want: "65535", overflow: "65536"},
		{name: "ratio", text: "0.1", want: "0.1", overflow: "1e100"},
		{name: "phase", text: "2+3i", want: "(2+3i)", overflow: "1e100+3i"},
	} {
		t.Run(test.name, func(t *testing.T) {
			in, _ := inputs.Lookup(test.name)
			out, _ := outputs.Lookup(test.name)
			if err := in.Value.Set(test.text); err != nil {
				t.Fatal(err)
			}
			if got := out.Value.Get(); got != test.want {
				t.Fatalf("Get() = %q; want %q", got, test.want)
			}
			if err := in.Value.Set(test.overflow); err == nil {
				t.Fatal("out-of-range value was accepted")
			}
			if got := out.Value.Get(); got != test.want {
				t.Fatalf("failed Set changed value to %q; want %q", got, test.want)
			}
		})
	}
	if cfg.Retries != 127 || cfg.Port != 65535 || cfg.Ratio != 0.1 || cfg.Phase != 2+3i {
		t.Fatalf("configuration = %+v", cfg)
	}
}

// A bad batch must leave existing declarations intact, whether binding fails
// or the complete batch conflicts with names already registered by the caller.
func TestJSONRejectsInvalidBatches(t *testing.T) {
	type octet byte
	for _, test := range []struct {
		name     string
		target   any
		metadata []program.Field
	}{
		{name: "Nil", target: nil},
		{name: "NilPointer", target: (*struct{})(nil)},
		{name: "Value", target: struct{}{}},
		{name: "NonStruct", target: new(string)},
		{name: "Unsupported", target: &struct {
			First  string
			Values map[string]string
		}{}},
		{name: "PrimitivePointer", target: &struct {
			First   string
			Pointer *string
		}{}},
		{name: "DefinedByteElement", target: &struct {
			First string
			Bytes []octet
		}{}},
		{name: "ExistingName", target: &struct {
			First     string
			Duplicate string `json:"existing"`
		}{}},
		{name: "AbsentMetadata", target: &struct{ First string }{}, metadata: []program.Field{
			{Name: "absent", Sensitive: true},
		}},
		{name: "RepeatedMetadata", target: &struct{ First string }{}, metadata: []program.Field{
			{Name: "First", Sensitive: true}, {Name: "First", Help: "duplicate"},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var inputs input.List
			var outputs output.List
			inputs.String("existing", "original", "")
			outputs.String("existing", "original", "")
			for _, bind := range []struct {
				name string
				call func(any, ...program.Field)
			}{
				{"Input", inputs.JSON}, {"Output", outputs.JSON},
			} {
				t.Run(bind.name, func(t *testing.T) {
					defer func() {
						if recover() == nil {
							t.Fatal("invalid declaration did not panic")
						}
						if len(slices.Collect(inputs.All())) != 1 || len(slices.Collect(outputs.All())) != 1 {
							t.Fatal("invalid batch changed the declaration surface")
						}
					}()
					bind.call(test.target, test.metadata...)
				})
			}
		})
	}
}

// Custom handles keep their identity and optional capabilities. Registration
// may inspect their types, but must not invoke their behavior.
func TestJSONRetainsCustomHandles(t *testing.T) {
	body := []byte("body")
	handle := bind.Bytes(&body)
	var typedNil *bind.BytesValue
	cfg := struct {
		Value   bind.BytesValue  `json:"value"`
		Pointer *bind.BytesValue `json:"pointer"`
		Nil     *bind.BytesValue `json:"nil"`
		Handle  interface {
			program.Setter
			program.Getter
		} `json:"handle"`
	}{
		Value: bind.BytesValue("value"), Pointer: handle, Nil: typedNil, Handle: handle,
	}
	var inputs input.List
	var outputs output.List
	inputs.JSON(&cfg)
	outputs.JSON(&cfg)
	for _, test := range []struct {
		name string
		want *bind.BytesValue
	}{
		{"value", &cfg.Value}, {"pointer", handle}, {"nil", typedNil}, {"handle", handle},
	} {
		in, _ := inputs.Lookup(test.name)
		out, _ := outputs.Lookup(test.name)
		if in.Value != test.want || out.Value != test.want {
			t.Errorf("%s lost its original handle", test.name)
		}
	}
	cfg.Pointer = new(bind.BytesValue)
	cfg.Handle = new(bind.BytesValue)
	out, _ := outputs.Lookup("pointer")
	other, _ := outputs.Lookup("handle")
	if out.Value != handle || other.Value != handle {
		t.Fatal("replacing a handle rebound the declaration")
	}
}
