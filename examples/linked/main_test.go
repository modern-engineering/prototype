package main

import (
	"errors"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/symbol"
)

func Example() {
	main()
	// Output:
	// database: job.Descriptor, can instantiate: true
	// backend: application.Descriptor, can instantiate: true
	// placement: catalogue.Placement, can instantiate: false
	// database.region = "eu-west"
	// backend.database <- database.address
}

// Generated definitions compose with manual values. A duplicate stops the
// registration sequence without replacing the duplicate or rolling back its
// successfully registered predecessors.
func TestRegistrationsRetainSuccessfulDefinitions(t *testing.T) {
	const cataloguePath = "github.com/modern-engineering/prototype/examples/linked/catalogue"
	var values symbol.Table
	config := &struct{ Region string }{Region: "eu-west"}
	if err := values.Define("example.org/operator", "Config", config); err != nil {
		t.Fatal(err)
	}
	if err := values.Define(cataloguePath, "Database", "manual definition"); err != nil {
		t.Fatal(err)
	}

	if err := registerDefinitions(&values); !errors.Is(err, symbol.ErrDuplicate) {
		t.Fatalf("registration error = %v, want ErrDuplicate", err)
	}
	if value, found := values.Lookup("example.org/operator", "Config"); !found || value != config {
		t.Fatalf("manual configuration = %v, %t, want original pointer", value, found)
	}
	value, found := values.Lookup(cataloguePath, "Backend")
	if _, ok := value.(program.Descriptor); !found || !ok {
		t.Fatalf("earlier definition = %T, %t, want program.Descriptor", value, found)
	}
	if value, found := values.Lookup(cataloguePath, "Database"); !found || value != "manual definition" {
		t.Fatalf("duplicate definition = %v, %t, want original value", value, found)
	}
	if _, found := values.Lookup(cataloguePath, "Europe"); found {
		t.Fatal("definition after the duplicate was registered")
	}
}
