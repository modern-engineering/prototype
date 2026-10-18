// Command symbols registers and uses runtime Go values by name without an IR
// model, source discovery, or generated code.
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/modern-engineering/prototype/symbol"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var values symbol.Table
	for _, entry := range []struct {
		path, name string
		value      any
	}{
		{"strings", "TrimSpace", strings.TrimSpace},
		{"strings", "ToUpper", strings.ToUpper},
		{"os", "Stdout", os.Stdout},
	} {
		if err := values.Define(entry.path, entry.name, entry.value); err != nil {
			return err
		}
	}

	text := "  hello from named Go values  "
	for _, name := range []string{"TrimSpace", "ToUpper"} {
		value, found := values.Lookup("strings", name)
		if !found {
			return fmt.Errorf("missing strings.%s", name)
		}
		transform, ok := value.(func(string) string)
		if !ok {
			return fmt.Errorf("strings.%s has unsupported type %T", name, value)
		}
		text = transform(text)
	}

	value, found := values.Lookup("os", "Stdout")
	if !found {
		return fmt.Errorf("missing os.Stdout")
	}
	writer, ok := value.(io.Writer)
	if !ok {
		return fmt.Errorf("os.Stdout does not implement io.Writer: %T", value)
	}
	_, err := fmt.Fprintln(writer, text)
	return err
}
