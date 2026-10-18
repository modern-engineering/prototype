// Package host supplies a command surface for module-resolution fixtures.
package host

import (
	"context"
	"io"

	"github.com/modern-engineering/prototype/program"
)

type Command struct {
	Context        context.Context
	Registry       *program.Registry
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

func (Command) Run([]string) int { return 0 }
