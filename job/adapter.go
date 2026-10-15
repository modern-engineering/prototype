package job

import "github.com/modern-engineering/prototype/program"

// From returns value as the runner with the inputs and outputs it declares.
// It preserves the original runner and handles, including their state and
// additional capabilities.
// Inputs and Outputs must only declare handles bound to that state; they must
// not perform I/O, change configured values, or execute the job.
//
// A MakeFunc can call From with a newly initialized value to supply defaults.
// From itself neither allocates a fresh value nor copies the supplied one.
func From(value interface {
	program.Runner
	Inputs() program.Inputs
	Outputs() program.Outputs
}) (program.Runner, program.Inputs, program.Outputs) {
	return value, value.Inputs(), value.Outputs()
}

// Make constructs a job from a fresh, zero-valued T on every call.
// A pointer to T must implement program.Runner and declare its bound inputs and
// outputs through Inputs and Outputs. Those methods must be usable on the zero
// value and obey From's construction contract.
//
// Make[T] can be used directly as a descriptor's MakeFunc. When T needs defaults
// or other initialization, use a MakeFunc that constructs T and calls From.
func Make[T any, P interface {
	*T
	program.Runner
	Inputs() program.Inputs
	Outputs() program.Outputs
}]() (program.Runner, program.Inputs, program.Outputs) {
	return From(P(new(T)))
}
