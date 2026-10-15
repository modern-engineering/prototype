package playground

import (
	"errors"
	"fmt"

	"github.com/modern-engineering/prototype/solution/link"
)

// Execution errors distinguish invalid policies, unsupported solution graphs,
// and applications that return before shutdown.
var (
	ErrInvalidTimeout        = errors.New("negative job timeout")
	ErrUnsupportedDescriptor = errors.New("unsupported descriptor type")
	ErrApplicationReference  = errors.New("application cannot produce referenced outputs")
	ErrDependencyCycle       = errors.New("dependency cycle")
	ErrUnexpectedExit        = errors.New("application returned before shutdown")
)

// Error identifies the occurrence and operation that failed.
// Binding, when non-nil, identifies the assignment being applied.
// Err is the original cause and can contain sensitive values.
// Error does not include Err in its text.
type Error struct {
	Node      *link.Node
	Binding   *link.Binding
	Operation string
	Err       error
}

func (e *Error) Error() string {
	location := fmt.Sprintf("occurrence %q", e.Node.Name())
	if e.Binding != nil {
		location += fmt.Sprintf(" input %q", e.Binding.Input())
	}
	return fmt.Sprintf("%s: %s failed", location, e.Operation)
}

// Unwrap returns the original cause, which can contain sensitive values.
func (e *Error) Unwrap() error { return e.Err }
