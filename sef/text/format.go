package text

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/modern-engineering/prototype/sef"
)

// Format normalizes whitespace and quotes data arguments while preserving
// comments, declaration order, and the original grouping of directives into
// blocks. Runs of blank lines become one blank line; leading and trailing blank
// lines are removed. It formats the parsed source, not an edited Artifact
// snapshot. Invalid or nil files return an error before writing. It leaves w open.
func Format(w io.Writer, file *File) error {
	if file == nil {
		return fmt.Errorf("sef/text: cannot format a nil file")
	}
	if len(file.errors) > 0 {
		return file.errors
	}
	if file.artifact.Version != sef.Version {
		return fmt.Errorf("%w: %q", sef.ErrVersion, file.artifact.Version)
	}
	var output bytes.Buffer
	blank := false
	for _, item := range file.lines {
		if item.kind == 0 && item.comment == "" {
			blank = output.Len() > 0
			continue
		}
		if blank {
			output.WriteByte('\n')
			blank = false
		}
		if item.inBlock {
			output.WriteByte('\t')
		}
		switch item.kind {
		case '(':
			output.WriteString(item.directive + " (")
		case ')':
			output.WriteByte(')')
		case 's':
			if !item.inBlock {
				output.WriteString(item.directive + " ")
			}
			output.WriteString(arguments(item.directive, item.arguments))
		}
		if item.comment != "" {
			if item.kind != 0 {
				output.WriteByte(' ')
			}
			output.WriteString(strings.TrimSpace(item.comment))
		}
		output.WriteByte('\n')
	}
	_, err := output.WriteTo(w)
	return err
}
