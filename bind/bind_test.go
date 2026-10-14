package bind_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/modern-engineering/prototype/bind"
	"github.com/modern-engineering/prototype/program"
)

// One binding supplies input and output handles backed by the same Go variable.
func Example() {
	name := "Ada"
	value := bind.String(&name)
	var input program.Setter = value
	var output program.Getter = value
	fmt.Println(output.Get())
	if err := input.Set("Grace"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(name)
	name = "Lin"
	fmt.Println(output.Get())
	// Output:
	// Ada
	// Grace
	// Lin
}

// Rejected text must leave the previously configured variables usable, with
// parser causes available to callers that distinguish syntax from overflow.
func TestFailedAssignmentsPreserveVariables(t *testing.T) {
	count, enabled := 17, true
	number := bind.Int(&count)
	boolean := bind.Bool(&enabled)
	if count != 17 || !enabled {
		t.Fatal("binding changed the existing variables")
	}
	if err := number.Set("seventeen"); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("invalid integer error = %v; want syntax error", err)
	}
	if count != 17 || number.Get() != "17" {
		t.Errorf("integer after invalid text = %d, %q; want 17", count, number.Get())
	}
	if err := number.Set("999999999999999999999999999999999999"); !errors.Is(err, strconv.ErrRange) {
		t.Errorf("overflow error = %v; want range error", err)
	}
	if count != 17 || number.Get() != "17" {
		t.Errorf("integer after overflow = %d, %q; want 17", count, number.Get())
	}
	if err := boolean.Set("sometimes"); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("invalid boolean error = %v; want syntax error", err)
	}
	if !enabled || boolean.Get() != "true" {
		t.Errorf("boolean after invalid text = %t, %q; want true", enabled, boolean.Get())
	}

	copied := number
	if err := copied.Set("23"); err != nil {
		t.Fatal(err)
	}
	if count != 23 || number.Get() != "23" {
		t.Errorf("integer assigned through copied binding = %d, %q; want 23", count, number.Get())
	}
	if err := boolean.Set("false"); err != nil {
		t.Fatal(err)
	}
	if enabled || boolean.Get() != "false" {
		t.Errorf("boolean after valid assignment = %t, %q; want false", enabled, boolean.Get())
	}
}
