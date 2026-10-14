package program

import "flag"

// FromFlags exposes the flags already declared in fs as bound inputs, in
// flag.FlagSet's lexical order. Each Value is the original flag.Value, preserving
// any additional capabilities it implements. Setter calls act on those values
// directly; they do not update FlagSet's command-line parsing or Visit history.
//
// Optional declarations match flags by name and provide Sensitive and nonempty
// Help overrides. Other flags retain their Usage and unmarked
// metadata. The adapter does not change flag defaults or parse arguments.
// Declaration names must be unique and name flags already present in fs.
func FromFlags(fs *flag.FlagSet, declarations ...Field) Inputs {
	metadata := make(map[string]Field, len(declarations))
	for _, declaration := range declarations {
		metadata[declaration.Name] = declaration
	}
	var fields InputFields
	fs.VisitAll(func(f *flag.Flag) {
		declaration := metadata[f.Name]
		declaration.Name = f.Name
		if declaration.Help == "" {
			declaration.Help = f.Usage
		}
		fields = append(fields, InputField{Field: declaration, Target: f.Value})
	})
	return fields
}
