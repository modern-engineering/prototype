// Package application describes applications with authored inputs.
package application

import "github.com/modern-engineering/prototype/program"

// MakeFunc constructs an application's runner and inputs.
// Each call binds them to fresh invocation state without performing I/O.
// Construction is infallible; operational failures belong to the runner.
// Applications do not declare outputs.
type MakeFunc func() (program.Runner, program.Inputs)

// Make constructs a fresh application invocation for the common program contract.
func (make MakeFunc) Make() (program.Runner, program.Inputs, program.Outputs) {
	runner, inputs := make()
	return runner, inputs, program.OutputFields(nil)
}

// Descriptor identifies an application and supplies its invocation constructor.
// It implements program.Factory through its embedded MakeFunc.
type Descriptor struct {
	Name    string
	Summary string
	MakeFunc
}
