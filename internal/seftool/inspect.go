package seftool

import (
	"bytes"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
)

func (c Command) inspect(w *bytes.Buffer, artifact solution.Artifact, linked, inputs bool) error {
	var registry *program.Registry
	if linked {
		registry = c.Registry
	}
	return host.Inspect(w, artifact, registry, inputs)
}
