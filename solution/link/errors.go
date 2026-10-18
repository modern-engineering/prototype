package link

import "errors"

// Errors is an ordered collection of findings about a definition. Each item
// retains its occurrence or binding context. The collection implements error
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
