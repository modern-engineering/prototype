package job_test

import (
	"testing"

	"github.com/modern-engineering/prototype/job"
)

// Adapting initialized state must preserve both the concrete runner and the
// connection from its fields to the returned handles.
func TestPreservesState(t *testing.T) {
	g := &greeting{Name: "World", Message: "pending"}
	runner, inputs, outputs := job.From(g)
	if runner != g {
		t.Fatalf("runner = %T %v, want original %p", runner, runner, g)
	}
	if g.Name != "World" || g.Message != "pending" {
		t.Fatalf("state after binding = %+v, want unchanged defaults", g)
	}
	name, ok := inputs.Lookup("name")
	if !ok {
		t.Fatal("missing name input")
	}
	if err := name.Value.Set("Ada"); err != nil {
		t.Fatal(err)
	}
	if g.Name != "Ada" {
		t.Fatalf("name = %q, want Ada", g.Name)
	}
	if err := runner.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	message, ok := outputs.Lookup("message")
	if !ok {
		t.Fatal("missing message output")
	}
	if got := message.Value.Get(); got != "Hello, Ada." {
		t.Fatalf("message = %q, want Hello, Ada.", got)
	}
}

// A reusable descriptor must isolate configuration and results between jobs.
func TestIndependentJobs(t *testing.T) {
	descriptor := job.Descriptor{MakeFunc: job.Make[greeting]}
	runner, inputs, outputs := descriptor.Make()
	otherRunner, otherInputs, otherOutputs := descriptor.Make()
	if runner == otherRunner {
		t.Fatal("Make reused its runner")
	}
	name, ok := inputs.Lookup("name")
	if !ok {
		t.Fatal("missing name input")
	}
	if err := name.Value.Set("Ada"); err != nil {
		t.Fatal(err)
	}
	otherName, ok := otherInputs.Lookup("name")
	if !ok {
		t.Fatal("missing second name input")
	}
	if err := otherName.Value.Set("Grace"); err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	message, ok := outputs.Lookup("message")
	if !ok {
		t.Fatal("missing message output")
	}
	if got := message.Value.Get(); got != "Hello, Ada." {
		t.Fatalf("message = %q, want Hello, Ada.", got)
	}
	otherMessage, ok := otherOutputs.Lookup("message")
	if !ok {
		t.Fatal("missing second message output")
	}
	if got := otherMessage.Value.Get(); got != "" {
		t.Fatalf("second message before Run = %q, want empty", got)
	}
	if err := otherRunner.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := otherMessage.Value.Get(); got != "Hello, Grace." {
		t.Fatalf("second message = %q, want Hello, Grace.", got)
	}
	if got := message.Value.Get(); got != "Hello, Ada." {
		t.Fatalf("first message after second Run = %q, want Hello, Ada.", got)
	}
}
