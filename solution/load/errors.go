package load

import (
	"errors"
	"fmt"

	"github.com/modern-engineering/prototype/solution/link"
)

// Instantiation errors distinguish invalid declaration surfaces from unresolved
// binding endpoints. The associated Error identifies the occurrence and field.
var (
	ErrNilRunner       = errors.New("nil runner")
	ErrDuplicateInput  = errors.New("duplicate input")
	ErrDuplicateOutput = errors.New("duplicate output")
	ErrNilInput        = errors.New("nil input handle")
	ErrNilOutput       = errors.New("nil output handle")
	ErrMissingInput    = errors.New("missing input")
	ErrMissingOutput   = errors.New("missing output")
)

// Error describes one finding about an invocation's declarations or a binding.
// Node owns the affected field. Binding, when non-nil, identifies the authored
// assignment requiring that field; for a missing output its receiver is another
// occurrence. Err supports errors.Is matching against the instantiation errors.
type Error struct {
	Node    *link.Node
	Binding *link.Binding
	Input   string
	Output  string
	Err     error
}

func (e *Error) Error() string {
	location := fmt.Sprintf("occurrence %q", e.Node.Name())
	if e.Input != "" {
		location += fmt.Sprintf(" input %q", e.Input)
	}
	if e.Output != "" {
		location += fmt.Sprintf(" output %q", e.Output)
	}
	if e.Binding != nil {
		location += fmt.Sprintf(" (binding %s.%s)", e.Binding.Node().Name(), e.Binding.Input())
	}
	return fmt.Sprintf("%s: %v", location, e.Err)
}

// Unwrap returns the finding's cause.
func (e *Error) Unwrap() error { return e.Err }

// Errors is an ordered collection of instantiation findings. It implements error
// and preserves errors.Is and errors.As matching for every finding.
type Errors []*Error

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}
	return errors.Join(e.Unwrap()...).Error()
}

// Unwrap returns the individual findings in diagnostic order.
func (e Errors) Unwrap() []error {
	result := make([]error, len(e))
	for i, finding := range e {
		result[i] = finding
	}
	return result
}
