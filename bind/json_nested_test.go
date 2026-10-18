package bind_test

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// Your configuration can keep the same structure and JSON names used elsewhere
// in your organization. The command derives dotted flags from those declarations;
// result state is bound separately from connection configuration.
func ExampleJSONInputs_nested() {
	cfg := struct {
		Database struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"database"`
	}{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	var inputs input.List
	inputs.JSON(&cfg, program.Field{Name: "database.host", Help: "database host"})
	flags := flag.NewFlagSet("connect", flag.ContinueOnError)
	if err := command.Parse(flags, &inputs, []string{"-database.host", "db.example"}); err != nil {
		fmt.Println("configure:", err)
		return
	}

	result := struct {
		Address string `json:"address"`
	}{Address: fmt.Sprintf("%s:%d", cfg.Database.Host, cfg.Database.Port)}
	var outputs output.List
	outputs.JSON(&result)
	if err := command.WriteOutputs(os.Stdout, &outputs); err != nil {
		fmt.Println("outputs:", err)
	}
	// Output: {"address":"db.example:5432"}
}

// Two occurrences of one configuration type bind independently. Retained handles
// must continue to refer to their original storage if a parent pointer changes.
func TestJSONNestedState(t *testing.T) {
	type database struct {
		Host string `json:"host,omitempty"`
		Port int    `json:"port,string"`
	}
	original := &database{Host: "backup", Port: 15432}
	cfg := struct {
		Primary database  `json:"primary"`
		Backup  *database `json:"backup,omitempty"`
	}{Primary: database{Host: "primary", Port: 5432}, Backup: original}
	metadata := program.Field{Name: "backup.host", Help: "backup database", Sensitive: true}
	inputs := program.InputList(bind.JSONInputs(&cfg, metadata))
	outputs := program.OutputList(bind.JSONOutputs(&cfg, metadata))
	var names []string
	for field := range inputs.All() {
		names = append(names, field.Name)
	}
	wantNames := []string{"primary.host", "primary.port", "backup.host", "backup.port"}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("input names = %q, want %q", names, wantNames)
	}
	names = nil
	for field := range outputs.All() {
		names = append(names, field.Name)
	}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("output names = %q, want %q", names, wantNames)
	}
	if cfg.Primary.Host != "primary" || cfg.Primary.Port != 5432 || *original != (database{Host: "backup", Port: 15432}) {
		t.Fatal("binding changed configuration")
	}
	port, _ := inputs.Lookup("primary.port")
	if err := port.Value.Set("6432"); err != nil {
		t.Fatal(err)
	}
	if cfg.Primary.Port != 6432 || original.Port != 15432 {
		t.Fatalf("ports = %d, %d; want 6432, 15432", cfg.Primary.Port, original.Port)
	}
	backup, _ := inputs.Lookup("backup.host")
	backupOut, _ := outputs.Lookup("backup.host")
	if backup.Field != metadata || backupOut.Field != metadata {
		t.Fatalf("backup metadata = %+v, %+v; want %+v", backup.Field, backupOut.Field, metadata)
	}
	cfg.Backup = &database{Host: "replacement", Port: 25432}
	if err := backup.Value.Set("retained"); err != nil {
		t.Fatal(err)
	}
	if original.Host != "retained" || cfg.Backup.Host != "replacement" || backupOut.Value.Get() != "retained" {
		t.Fatal("replacing the parent pointer changed retained bindings")
	}
	cfg.Primary.Host = "direct"
	host, _ := outputs.Lookup("primary.host")
	if got := host.Value.Get(); got != "direct" {
		t.Fatalf("primary host = %q, want direct", got)
	}
}

// Field selection happens before following embedded pointers. An ignored or
// shadowed nil branch must not make an otherwise usable configuration fail.
func TestJSONPromotedFields(t *testing.T) {
	type Base struct{ Name string }
	type hidden struct{ Name string }
	type Tagged struct {
		Alias string `json:"Name"`
	}
	for _, test := range []struct {
		name   string
		target any
	}{
		{name: "ShallowerField", target: &struct {
			*Base
			Name string
		}{Name: "selected"}},
		{name: "TaggedField", target: &struct {
			*Base
			Tagged
		}{Tagged: Tagged{Alias: "selected"}}},
		{name: "ExcludedField", target: &struct {
			*Base `json:"-"`
			Name  string
		}{Name: "selected"}},
		{name: "PromotedField", target: &struct {
			*Base
		}{Base: &Base{Name: "selected"}}},
		{name: "UnexportedEmbedding", target: &struct {
			*hidden
		}{hidden: &hidden{Name: "selected"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			inputs := bind.JSONInputs(test.target)
			outputs := bind.JSONOutputs(test.target)
			if len(inputs) != 1 || len(outputs) != 1 || inputs[0].Name != "Name" || outputs[0].Name != "Name" {
				t.Fatalf("fields = %+v, %+v; want only Name", inputs, outputs)
			}
			if got := outputs[0].Value.Get(); got != "selected" {
				t.Fatalf("selected output = %q, want selected", got)
			}
			if err := inputs[0].Value.Set("updated"); err != nil {
				t.Fatal(err)
			}
			if got := outputs[0].Value.Get(); got != "updated" {
				t.Fatalf("bound output = %q, want updated", got)
			}
		})
	}
}

// Each promoted leaf reaches the same pointer slot. The first handles returned
// must all refer to the single pointee installed by a successful binding.
func TestJSONPromotedAllocationSharesStorage(t *testing.T) {
	type Connection struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	var configured struct{ *Connection }
	inputs := bind.JSONInputs(&configured)
	if len(inputs) != 2 {
		t.Fatalf("got %d inputs, want 2", len(inputs))
	}
	if err := inputs[0].Value.Set("db.example"); err != nil {
		t.Fatal(err)
	}
	if err := inputs[1].Value.Set("5432"); err != nil {
		t.Fatal(err)
	}
	if configured.Connection == nil || configured.Host != "db.example" || configured.Port != 5432 {
		t.Fatalf("materialized input state = %+v", configured.Connection)
	}

	var result struct{ *Connection }
	outputs := bind.JSONOutputs(&result)
	if result.Connection == nil || len(outputs) != 2 {
		t.Fatal("output binding did not materialize both fields")
	}
	result.Host, result.Port = "db.example", 5432
	if outputs[0].Value.Get() != "db.example" || outputs[1].Value.Get() != "5432" {
		t.Fatal("returned outputs do not share the installed pointee")
	}
}

// Nil nested pointers need real storage for their declared handles. Binding in
// either direction must initialize that storage without replacing existing state
// or invoking a custom handle's behavior.
func TestJSONMaterializesNestedState(t *testing.T) {
	type credentials struct {
		Token string `json:"token"`
	}
	type database struct {
		Host   string        `json:"host"`
		Auth   *credentials  `json:"auth"`
		Custom *nestedHandle `json:"custom"`
	}
	type Promoted struct {
		Region string `json:"region"`
	}
	for _, operation := range []struct {
		name string
		bind func(any)
	}{
		{"Inputs", func(target any) { bind.JSONInputs(target) }},
		{"Outputs", func(target any) { bind.JSONOutputs(target) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			auth := &credentials{Token: "existing token"}
			existing := &database{Host: "existing host", Auth: auth}
			cfg := struct {
				*Promoted
				Existing *database `json:"existing"`
				Missing  *database `json:"missing,omitempty"`
				Ignored  *database `json:"-"`
			}{Existing: existing}
			operation.bind(&cfg)
			if cfg.Promoted == nil || cfg.Missing == nil || cfg.Missing.Auth == nil {
				t.Fatalf("binding left nested storage nil: %+v", cfg)
			}
			if cfg.Region != "" || cfg.Missing.Host != "" || cfg.Missing.Auth.Token != "" {
				t.Fatal("new nested state did not start with zero values")
			}
			if cfg.Existing != existing || existing.Auth != auth || existing.Host != "existing host" || auth.Token != "existing token" {
				t.Fatal("binding changed initialized nested state")
			}
			if cfg.Existing.Custom != nil || cfg.Missing.Custom != nil {
				t.Fatal("binding allocated a typed nil custom handle")
			}
			if cfg.Ignored != nil {
				t.Fatal("binding allocated excluded state")
			}

			inputs := program.InputList(bind.JSONInputs(&cfg))
			outputs := program.OutputList(bind.JSONOutputs(&cfg))
			region, ok := inputs.Lookup("region")
			if !ok {
				t.Fatal("missing promoted input")
			}
			if err := region.Value.Set("west"); err != nil {
				t.Fatal(err)
			}
			token, ok := inputs.Lookup("missing.auth.token")
			if !ok {
				t.Fatal("missing nested input")
			}
			if err := token.Value.Set("new token"); err != nil {
				t.Fatal(err)
			}
			tokenOut, ok := outputs.Lookup("missing.auth.token")
			if !ok {
				t.Fatal("missing nested output")
			}
			if cfg.Region != "west" || cfg.Missing.Auth.Token != "new token" || tokenOut.Value.Get() != "new token" {
				t.Fatal("nested handles do not refer to materialized storage")
			}
		})
	}
}

// Ambiguous names and recursive nested types cannot produce usable field handles.
func TestJSONRejectsInvalidNestedState(t *testing.T) {
	type Left struct{ Name string }
	type Right struct{ Name string }
	type hidden struct{ Name string }
	type recursivePointer *recursivePointer
	type recursive struct {
		Next *recursive `json:"next"`
	}
	type database struct {
		Host string `json:"host"`
	}
	cycle := new(recursive)
	cycle.Next = cycle
	for _, test := range []struct {
		name   string
		target any
	}{
		{name: "AmbiguousPromotion", target: &struct {
			Left
			Right
		}{}},
		{name: "DottedCollision", target: &struct {
			Database database `json:"database"`
			Host     string   `json:"database.host"`
		}{}},
		{name: "RecursiveType", target: cycle},
		{name: "RecursivePointerType", target: &struct{ Pointer recursivePointer }{}},
		{name: "UnexportedNilEmbedding", target: &struct{ *hidden }{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, operation := range []struct {
				name string
				bind func()
			}{
				{"Inputs", func() { bind.JSONInputs(test.target) }},
				{"Outputs", func() { bind.JSONOutputs(test.target) }},
			} {
				t.Run(operation.name, func(t *testing.T) {
					defer func() {
						if recover() == nil {
							t.Fatal("invalid configuration did not panic")
						}
					}()
					operation.bind()
				})
			}
		})
	}
}

// Allocation must wait until the complete binding is valid. Otherwise a
// declaration mistake could leave configuration partly initialized after panic.
func TestJSONFailedBindingPreservesNilState(t *testing.T) {
	type database struct {
		Host string `json:"host"`
	}
	for _, operation := range []struct {
		name string
		bind func(any, ...program.Field)
	}{
		{"Inputs", func(target any, metadata ...program.Field) { bind.JSONInputs(target, metadata...) }},
		{"Outputs", func(target any, metadata ...program.Field) { bind.JSONOutputs(target, metadata...) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			unsupported := &struct {
				Database *database `json:"database"`
				Values   map[string]string
			}{}
			unknown := &struct {
				Database *database `json:"database"`
			}{}
			for _, test := range []struct {
				name     string
				target   any
				pointer  **database
				metadata []program.Field
			}{
				{name: "UnsupportedField", target: unsupported, pointer: &unsupported.Database},
				{name: "UnknownMetadata", target: unknown, pointer: &unknown.Database, metadata: []program.Field{{Name: "absent"}}},
			} {
				t.Run(test.name, func(t *testing.T) {
					defer func() {
						if recover() == nil {
							t.Fatal("invalid declaration did not panic")
						}
						if *test.pointer != nil {
							t.Fatal("failed binding allocated nested configuration")
						}
					}()
					operation.bind(test.target, test.metadata...)
				})
			}
		})
	}
}

type nestedSetter struct {
	Text string `json:"text"`
}

func (s *nestedSetter) Set(text string) error { s.Text = text; return nil }

type nestedGetter struct {
	Text string `json:"text"`
}

func (g *nestedGetter) Get() string { return g.Text }

type nestedHandle struct{ Next *nestedHandle }

func (*nestedHandle) Set(string) error { panic("binding called Set") }
func (*nestedHandle) Get() string      { panic("binding called Get") }

// A custom handle supplies its own meaning for its direction. Its internal
// structure and typed-nil usability remain the handle author's responsibility.
func TestJSONCustomHandlesStopDescent(t *testing.T) {
	cfg := struct {
		Input  nestedSetter  `json:"input"`
		Output nestedGetter  `json:"output"`
		Nil    *nestedHandle `json:"nil"`
	}{}
	inputs := bind.JSONInputs(&cfg)
	outputs := bind.JSONOutputs(&cfg)
	var inNames, outNames []string
	for _, field := range inputs {
		inNames = append(inNames, field.Name)
	}
	for _, field := range outputs {
		outNames = append(outNames, field.Name)
	}
	if !slices.Equal(inNames, []string{"input", "output.text", "nil"}) {
		t.Fatalf("input names = %q, want input, output.text, nil", inNames)
	}
	if !slices.Equal(outNames, []string{"input.text", "output", "nil"}) {
		t.Fatalf("output names = %q, want input.text, output, nil", outNames)
	}
	if inputs[0].Value != &cfg.Input || outputs[1].Value != &cfg.Output {
		t.Fatal("custom struct handle was replaced")
	}
	if inputs[2].Value != cfg.Nil || outputs[2].Value != cfg.Nil {
		t.Fatal("typed nil custom handle was replaced")
	}
}
