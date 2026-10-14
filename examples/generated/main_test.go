package main

import (
	"errors"
	"testing"

	"github.com/modern-engineering/prototype/program"
)

func Example() {
	main()
	// Output:
	// database: job.Descriptor, can instantiate: true
	// backend: application.Descriptor, can instantiate: true
	// database.region = "eu-west"
	// backend.database <- database.address
}

// Generated definitions compose with manual values. A duplicate stops the
// registration sequence without replacing the duplicate or rolling back its
// successfully registered predecessors.
func TestRegistrationsRetainSuccessfulDefinitions(t *testing.T) {
	const cataloguePath = "github.com/modern-engineering/prototype/examples/catalogue"
	var values program.Registry
	manual := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) { return nil, nil, nil })
	config := &manual
	if err := values.Register("example.org/operator", "Config", config); err != nil {
		t.Fatal(err)
	}
	if err := values.Register(cataloguePath, "Database", config); err != nil {
		t.Fatal(err)
	}

	if err := registerDefinitions(&values); !errors.Is(err, program.ErrDuplicate) {
		t.Fatalf("registration error = %v, want ErrDuplicate", err)
	}
	if value, found := values.Lookup("example.org/operator", "Config"); !found || value != config {
		t.Fatalf("manual configuration = %v, %t, want original pointer", value, found)
	}
	value, found := values.Lookup(cataloguePath, "Backend")
	if !found || value == nil {
		t.Fatalf("earlier descriptor = %T, %t, want a registered descriptor", value, found)
	}
	if value, found := values.Lookup(cataloguePath, "Database"); !found || value != config {
		t.Fatalf("duplicate definition = %v, %t, want original value", value, found)
	}
}
