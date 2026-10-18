package input

import "github.com/modern-engineering/prototype/program"

// Option modifies declaration metadata without receiving the bound value.
// Options are applied in argument order before a declaration is registered;
// a nil option panics. Registration copies the resulting metadata, and Define
// checks for duplicates using the final name. If an option panics, the declaration is not registered.
type Option func(*program.Field)

// Sensitive marks a declaration as sensitive. Pass it as a trailing option to
// Var or a typed helper, for example StringVar(&token, "token", "credential", Sensitive).
func Sensitive(field *program.Field) {
	field.Sensitive = true
}
