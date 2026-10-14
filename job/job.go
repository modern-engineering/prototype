// Package job describes programs with authored inputs and outputs.
package job

import "github.com/modern-engineering/prototype/program"

// MakeFunc constructs a job's runner, inputs, and outputs.
// Each call binds them to fresh invocation state without performing I/O.
// Construction is infallible; operational failures belong to the runner.
type MakeFunc = program.MakeFunc

// Descriptor identifies a job and supplies its invocation constructor.
// It implements program.Descriptor through its embedded MakeFunc.
type Descriptor struct {
	Name    string
	Summary string
	MakeFunc
}
