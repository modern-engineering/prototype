package command

import (
	"encoding/json"
	"io"

	"github.com/modern-engineering/prototype/program"
)

// WriteOutputs writes all declared outputs as a JSON object followed by a
// newline. Each member uses its declaration's name and the string returned by
// its original Getter. Names are sorted by encoding/json; strings are escaped
// without interpreting their content as JSON or inferring a Go type.
// A nil or empty surface writes {}. Names must be unique, as Outputs requires.
//
// Call WriteOutputs after a finite program completes successfully. Every output
// is included, including Sensitive fields: this writer carries program results.
// WriteOutputs does not run the program or close w. Writer errors are returned.
func WriteOutputs(w io.Writer, outputs program.Outputs) error {
	values := make(map[string]string)
	if outputs != nil {
		for field := range outputs.All() {
			values[field.Name] = field.Value.Get()
		}
	}
	return json.NewEncoder(w).Encode(values)
}
