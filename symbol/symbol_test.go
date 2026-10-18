package symbol_test

import (
	"errors"
	"testing"

	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/symbol"
)

// Registration preserves identity and capabilities without invoking values;
// duplicate rejection must work even when those values cannot be compared.
func TestDefinitionsPreserveValues(t *testing.T) {
	var values, other symbol.Table
	calls := 0
	getter := program.GetterFunc(func() string {
		calls++
		return "original"
	})
	if err := values.Define("example.org/catalogue", "Value", getter); err != nil {
		t.Fatal(err)
	}
	if err := values.Define("example.org/catalogue", "Value", []string{"replacement"}); !errors.Is(err, symbol.ErrDuplicate) {
		t.Fatalf("duplicate definition error = %v, want ErrDuplicate", err)
	}
	value, found := values.Lookup("example.org/catalogue", "Value")
	if !found {
		t.Fatal("registered getter not found")
	}
	if calls != 0 {
		t.Fatalf("registration or lookup invoked the getter %d times", calls)
	}
	if got := value.(program.Getter).Get(); got != "original" || calls != 1 {
		t.Fatalf("getter returned %q after %d calls, want original after one call", got, calls)
	}

	data := map[string]string{"message": "before"}
	if err := other.Define("example.org/catalogue", "Value", data); err != nil {
		t.Fatal(err)
	}
	data["message"] = "after"
	value, found = other.Lookup("example.org/catalogue", "Value")
	if !found || value.(map[string]string)["message"] != "after" {
		t.Fatalf("shared map lookup = %v, %t", value, found)
	}
	if _, found := values.Lookup("another.org/catalogue", "Value"); found {
		t.Fatal("lookup found a definition from another package path")
	}
}

// Presence is independent of the value, including an untyped or typed nil.
func TestNilIsARegisteredValue(t *testing.T) {
	var values symbol.Table
	if value, found := values.Lookup("p", "Nil"); value != nil || found {
		t.Fatalf("empty table lookup = %v, %t", value, found)
	}
	if err := values.Define("p", "Nil", nil); err != nil {
		t.Fatal(err)
	}
	if err := values.Define("p", "Nil", 1); !errors.Is(err, symbol.ErrDuplicate) {
		t.Fatalf("duplicate nil definition error = %v, want ErrDuplicate", err)
	}
	if value, found := values.Lookup("p", "Nil"); value != nil || !found {
		t.Fatalf("registered nil lookup = %v, %t", value, found)
	}

	var pointer *int
	if err := values.Define("p", "Pointer", pointer); err != nil {
		t.Fatal(err)
	}
	value, found := values.Lookup("p", "Pointer")
	if !found || value == nil || value.(*int) != nil {
		t.Fatalf("typed nil lookup = %#v, %t", value, found)
	}
}
