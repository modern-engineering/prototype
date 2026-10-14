package bind_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/program"
)

// Separate functions can normalize incoming text and compute an outgoing value.
func ExampleSetterFunc() {
	name := "Ada"
	input := bind.SetterFunc(func(text string) error {
		name = strings.TrimSpace(text)
		return nil
	})
	output := bind.GetterFunc(func() string { return "Hello, " + name })
	fmt.Println(output.Get())
	if err := input.Set(" Lin "); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(output.Get())
	// Output:
	// Hello, Ada
	// Hello, Lin
}

// A callback adapter exposes only the direction its function actually supplies.
func TestFunctionDirections(t *testing.T) {
	rejected := errors.New("assignment rejected")
	var received string
	var input program.Setter = bind.SetterFunc(func(text string) error {
		received = text
		return rejected
	})
	if err := input.Set("value"); err != rejected || received != "value" {
		t.Fatalf("assignment = %q, %v; want original text and callback error", received, err)
	}
	if _, ok := input.(program.Getter); ok {
		t.Error("setter callback also exposes a getter")
	}
	var output program.Getter = bind.GetterFunc(func() string { return received })
	if _, ok := output.(program.Setter); ok {
		t.Error("getter callback also exposes a setter")
	}
}
