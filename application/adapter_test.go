package application_test

import (
	"context"
	"io"
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/program"
)

type server struct {
	address string
	runs    int
	output  io.Writer
}

func (s *server) Inputs() program.Inputs {
	var inputs input.List
	inputs.StringVar(&s.address, "address", "Listen address.")
	return &inputs
}

func (s *server) UseWriter(output io.Writer) { s.output = output }

func (s *server) Run(context.Context) error {
	s.runs++
	return nil
}

// A catalogue can preserve explicit defaults or allocate zero-valued state;
// both paths must bind inputs to the exact runner returned to the loader.
func TestIndependentState(t *testing.T) {
	configured := &server{address: ":8080"}
	runner, inputs := application.From(configured)
	if runner != configured {
		t.Fatalf("From returned runner %v, want supplied server", runner)
	}
	if configured.address != ":8080" || configured.runs != 0 {
		t.Fatalf("From changed configured state: %+v", configured)
	}
	input, ok := inputs.Lookup("address")
	if !ok {
		t.Fatal("From omitted address input")
	}
	if err := input.Value.Set(":9090"); err != nil {
		t.Fatal(err)
	}
	if configured.address != ":9090" {
		t.Fatalf("configured address = %q, want :9090", configured.address)
	}

	descriptor := application.Descriptor{MakeFunc: application.Make[server]}
	firstRunner, firstInputs, outputs := descriptor.Make()
	secondRunner, _, _ := descriptor.Make()
	first, ok := firstRunner.(*server)
	if !ok {
		t.Fatalf("Make returned %T, want *server", firstRunner)
	}
	second, ok := secondRunner.(*server)
	if !ok {
		t.Fatalf("Make returned %T, want *server", secondRunner)
	}
	if first == second {
		t.Fatal("Make reused a server")
	}
	input, ok = firstInputs.Lookup("address")
	if !ok {
		t.Fatal("Make omitted address input")
	}
	if err := input.Value.Set(":7070"); err != nil {
		t.Fatal(err)
	}
	writerUser, ok := firstRunner.(interface{ UseWriter(io.Writer) })
	if !ok {
		t.Fatal("Make hid the runner's writer capability")
	}
	writerUser.UseWriter(io.Discard)
	if first.address != ":7070" || first.output != io.Discard || first.runs != 0 {
		t.Fatalf("first server = %+v, want configured but not run", first)
	}
	if second.address != "" || second.output != nil || second.runs != 0 {
		t.Fatalf("second server shares state: %+v", second)
	}
	for output := range outputs.All() {
		t.Errorf("application declared output %q", output.Name)
	}
	if err := firstRunner.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if first.runs != 1 || second.runs != 0 {
		t.Fatalf("run counts = (%d, %d), want (1, 0)", first.runs, second.runs)
	}
}
