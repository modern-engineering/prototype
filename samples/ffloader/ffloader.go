package ffloader

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/modern-engineering/prototype/application"
)

func Load(app *application.Descriptor, opts ...interface{}) {
	ctx := context.Background()
	svc := app.Make()
	parser := application.ParseTo(func(context.Context, *flag.FlagSet) error { return nil })
	parser = application.ParseFromEnvs(parser)
	parser = application.ParseFromArgs(parser, os.Args[1:])
	err := parser.Parse(ctx, svc.Flags())
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		err = svc.Run(ctx)
		if err != nil {
			panic(err)
		}
	})
	wg.Go(func() {
		shutdowner, ok := svc.(application.Shutdowner)
		if !ok {
			return
		}

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		defer signal.Stop(signalChan)
		select {
		case <-ctx.Done():
			return
		case <-signalChan:
		}

		err := shutdowner.Shutdown(ctx)
		if err != nil {
			panic(err)
		}
	})
	wg.Wait()
}
