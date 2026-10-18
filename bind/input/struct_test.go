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
func TestStructBindsLiveState(t *testing.T) {
	type token string
	type enabled bool
	type payload []byte
	cfg := struct {
		Token   token    `input:"token,sensitive" output:"token,sensitive" help:"credential"`
		Enabled enabled  `input:"enabled" output:"enabled"`
		Body    payload  `input:"body" output:"body"`
		Ignored chan int `input:"-" output:"-"`
		Private string
	}{Token: "initial", Enabled: true, Body: payload("draft"), Private: "internal"}
	var inputs input.List
	var outputs output.List
	inputs.String("first", "existing", "")
	outputs.String("first", "existing", "")
	inputs.Struct(&cfg)
	outputs.Struct(&cfg)
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
func TestStructNumericWidths(t *testing.T) {
	type retries int8
	type port uint16
	type ratio float32
	type phase complex64
	cfg := struct {
		Retries retries `input:"retries" output:"retries"`
		Port    port    `input:"port" output:"port"`
		Ratio   ratio   `input:"ratio" output:"ratio"`
		Phase   phase   `input:"phase" output:"phase"`
	}{Retries: 3, Port: 80, Ratio: 0.5, Phase: 1 + 2i}
	var inputs input.List
	var outputs output.List
	inputs.Struct(&cfg)
	outputs.Struct(&cfg)
	for _, test := range []struct{ name, text, want, overflow string }{
		{"retries", "127", "127", "128"},
		{"port", "65535", "65535", "65536"},
		{"ratio", "0.1", "0.1", "1e100"},
		{"phase", "2+3i", "(2+3i)", "1e100+3i"},
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

// Custom handles keep their identity and optional capabilities. Registration
// may inspect their types, but must not invoke their behavior.
func TestStructRetainsCustomHandles(t *testing.T) {
	body := []byte("body")
	handle := bind.Bytes(&body)
	var typedNil *bind.BytesValue
	cfg := struct {
		Value   bind.BytesValue  `input:"value" output:"value"`
		Pointer *bind.BytesValue `input:"pointer" output:"pointer"`
		Nil     *bind.BytesValue `input:"nil" output:"nil"`
		Handle  interface {
			program.Setter
			program.Getter
		} `input:"handle" output:"handle"`
		Setter program.Setter `input:"setter"`
		Getter program.Getter `output:"getter"`
	}{
		Value: bind.BytesValue("value"), Pointer: handle, Nil: typedNil, Handle: handle,
		Setter: program.SetterFunc(func(string) error { t.Fatal("registration called Set"); return nil }),
		Getter: program.GetterFunc(func() string { t.Fatal("registration called Get"); return "" }),
	}
	var inputs input.List
	var outputs output.List
	inputs.Struct(&cfg)
	outputs.Struct(&cfg)
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

// Declaration errors must not leave a partially registered surface. Invalid
// batches start with a valid field to exercise validation before append.
func TestStructRejectsInvalidDeclarations(t *testing.T) {
	type octet byte
	for _, test := range []struct {
		name   string
		target any
	}{
		{"Nil", nil},
		{"NilPointer", (*struct{})(nil)},
		{"Value", struct{}{}},
		{"NonStruct", new(string)},
		{"Unexported", &struct {
			First  string `input:"first" output:"first"`
			hidden string `input:"hidden" output:"hidden"`
		}{}},
		{"EmptyName", &struct {
			First string `input:"first" output:"first"`
			Empty string `input:"" output:""`
		}{}},
		{"MissingName", &struct {
			First string `input:"first" output:"first"`
			Empty string `input:",sensitive" output:",sensitive"`
		}{}},
		{"UnknownOption", &struct {
			First string `input:"first" output:"first"`
			Token string `input:"token,sensitve" output:"token,sensitve"`
		}{}},
		{"Unsupported", &struct {
			First  string            `input:"first" output:"first"`
			Values map[string]string `input:"values" output:"values"`
		}{}},
		{"PrimitivePointer", &struct {
			First   string  `input:"first" output:"first"`
			Pointer *string `input:"pointer" output:"pointer"`
		}{}},
		{"DefinedByteElement", &struct {
			First string  `input:"first" output:"first"`
			Bytes []octet `input:"bytes" output:"bytes"`
		}{}},
		{"Nested", &struct {
			First  string                 `input:"first" output:"first"`
			Nested struct{ Value string } `input:"nested" output:"nested"`
		}{}},
		{"ExistingName", &struct {
			First     string `input:"first" output:"first"`
			Duplicate string `input:"existing" output:"existing"`
		}{}},
		{"RepeatedName", &struct {
			First     string `input:"first" output:"first"`
			Duplicate string `input:"first" output:"first"`
		}{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, direction := range []string{"input", "output"} {
				t.Run(direction, func(t *testing.T) {
					var inputs input.List
					var outputs output.List
					inputs.String("existing", "original", "")
					outputs.String("existing", "original", "")
					defer func() {
						if recover() == nil {
							t.Fatal("invalid declaration did not panic")
						}
						if len(slices.Collect(inputs.All())) != 1 || len(slices.Collect(outputs.All())) != 1 {
							t.Fatal("invalid batch changed the declaration surface")
						}
					}()
					if direction == "input" {
						inputs.Struct(test.target)
					} else {
						outputs.Struct(test.target)
					}
				})
			}
		})
	}
}
