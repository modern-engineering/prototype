package bind

// SetterFunc receives an assignment through a function. It implements Set only;
// the function decides how to validate and store each value.
// A nil SetterFunc is not usable.
type SetterFunc func(string) error

// Set calls f with text and returns its error unchanged.
func (f SetterFunc) Set(text string) error { return f(text) }

// GetterFunc supplies a value through a function. It implements Get only;
// each call invokes the function again.
// A nil GetterFunc is not usable.
type GetterFunc func() string

// Get calls f and returns its result.
func (f GetterFunc) Get() string { return f() }
