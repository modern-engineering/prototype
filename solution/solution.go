package solution

import (
	"flag"

	"github.com/modern-engineering/prototype/application"
)

type Component interface {
	ComponentDefinition(d *Designer, w CanvasWriter) error
}

type Designer struct {
}

type CanvasWriter interface {
	WriteApplication(app application.Descriptor) *AppSpec
}

type AppSpec struct {
	Descriptor *application.Descriptor
	Input      flag.FlagSet
	Output     flag.FlagSet
}

func NewAppSpec(desc *application.Descriptor) *AppSpec {
	flags := desc.Flags()
	return &AppSpec{}
}
