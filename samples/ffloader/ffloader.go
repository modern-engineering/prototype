package ffloader

import (
	"context"
	"flag"
	"os"

	"github.com/modern-engineering/prototype/application"
)

func Load(app *application.Descriptor, opts ...interface{}) {
	ctx := context.Background()
	prog, flags := app.Make()
	parser := application.ParseTo(func(context.Context, *flag.FlagSet) error { return nil })
	parser = application.ParseFromEnvs(parser)
	parser = application.ParseFromArgs(parser, os.Args[1:])
	err := parser.Parse(ctx, flags)
	if err != nil {
		panic(err)
	}
	err = prog.Run(ctx)
	if err != nil {
		panic(err)
	}
}
