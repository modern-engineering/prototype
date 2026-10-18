package bind_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/program"
)

// Your own declaration collection can reuse JSON names and attach metadata.
// Even an empty field tagged omitempty remains available for configuration.
func ExampleJSONInputs() {
	cfg := struct {
		Region string `json:"region"`
		Token  string `json:"token,omitempty"`
	}{Region: "local"}
	fields := bind.JSONInputs(&cfg, program.Field{
		Name: "token", Help: "service credential", Sensitive: true,
	})
	for _, field := range fields {
		fmt.Println(field.Name, field.Sensitive)
	}
	// Output:
	// region false
	// token true
}

// JSON options describe serialization, not whether a program declares a field
// or which string representation its scalar setter accepts.
func TestJSONNamesDoNotDependOnValues(t *testing.T) {
	cfg := struct {
		Default string
		Empty   string `json:""`
		Zero    string `json:",omitempty"`
		Renamed string `json:"service,omitempty"`
		Count   int    `json:"count,string,omitzero"`
		Dash    string `json:"-,"`        //nolint:staticcheck // Exercise encoding/json v1's literal-dash name.
		Invalid string `json:"bad\\name"` //nolint:staticcheck // Exercise the documented fallback for invalid tag names.
		Unicode string `json:"地域"`
		Hidden  func() `json:"-"`
		private string
	}{private: "internal"}
	inputs := bind.JSONInputs(&cfg)
	outputs := bind.JSONOutputs(&cfg)
	var inNames, outNames []string
	for _, field := range inputs {
		inNames = append(inNames, field.Name)
	}
	for _, field := range outputs {
		outNames = append(outNames, field.Name)
	}
	want := []string{"Default", "Empty", "Zero", "service", "count", "-", "Invalid", "地域"}
	if !slices.Equal(inNames, want) || !slices.Equal(outNames, want) {
		t.Fatalf("names = %q, %q; want %q", inNames, outNames, want)
	}
	if err := inputs[4].Value.Set("23"); err != nil {
		t.Fatal(err)
	}
	if cfg.Count != 23 || outputs[4].Value.Get() != "23" || cfg.private != "internal" {
		t.Fatalf("binding did not preserve scalar semantics: %+v", cfg)
	}
}

type jsonConfig struct {
	Value program.Setter `json:"value"`
}

func (*jsonConfig) MarshalJSON() ([]byte, error) { panic("called MarshalJSON") }
func (*jsonConfig) UnmarshalJSON([]byte) error   { panic("called UnmarshalJSON") }

// Declaration construction must not trigger marshaling or invoke custom
// handles. Such methods can depend on runtime resources supplied later.
func TestJSONBindingDoesNotInvokeBehavior(t *testing.T) {
	cfg := jsonConfig{Value: program.SetterFunc(func(string) error {
		t.Fatal("binding called Set")
		return nil
	})}
	result := struct {
		Value program.Getter `json:"value"`
	}{Value: program.GetterFunc(func() string {
		t.Fatal("binding called Get")
		return ""
	})}
	metadata := []program.Field{{Name: "value", Help: "configured value", Sensitive: true}}
	inputs := bind.JSONInputs(&cfg, metadata...)
	outputs := bind.JSONOutputs(&result, metadata...)
	metadata[0].Help = "changed"
	metadata[0].Sensitive = false
	want := program.Field{Name: "value", Help: "configured value", Sensitive: true}
	if len(inputs) != 1 || len(outputs) != 1 || inputs[0].Field != want || outputs[0].Field != want {
		t.Fatalf("declarations = %+v, %+v; want independent metadata %v", inputs, outputs, want)
	}
}
