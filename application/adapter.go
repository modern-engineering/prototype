package application

import "github.com/modern-engineering/prototype/program"

// From returns value and the inputs declared by its Inputs method. It retains
// the original runner and input handles, including their additional capabilities.
// It neither copies value nor calls Run. Inputs must only declare fields bound
// to value, without performing I/O or changing configured values.
//
// Use From inside a MakeFunc to adapt a freshly constructed value with defaults.
// Reusing the same value across calls would share its state between runs.
func From(value interface {
	program.Runner
	Inputs() program.Inputs
}) (program.Runner, program.Inputs) {
	return value, value.Inputs()
}

// Make allocates a fresh zero-valued T and returns From applied to its pointer.
// A *T must implement program.Runner and declare its bound fields through Inputs.
// Inputs must be usable on the zero value and obey From's construction contract.
//
// Make[T] can be assigned directly to a descriptor's MakeFunc. For constructors
// that set defaults, use a MakeFunc calling From instead.
func Make[T any, P interface {
	*T
	program.Runner
	Inputs() program.Inputs
}]() (program.Runner, program.Inputs) {
	return From(P(new(T)))
}
