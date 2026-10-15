// Command playground hosts an organization's catalogue in one process.
// Acme compiles its private catalogue into the host; a serialized artifact
// supplies the occurrence names and bindings.
// Two jobs resolve a development environment and its database address before
// the API and reporting applications start. Interrupting the host stops both.
package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"example.com/acme/catalogue"
	"github.com/modern-engineering/prototype/playground"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution/link"
	"github.com/modern-engineering/prototype/solution/load"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Stdout); err != nil && (ctx.Err() == nil || !errors.Is(err, context.Canceled)) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output io.Writer) error {
	logger := log.New(output, "", 0)
	var registry program.Registry
	for _, entry := range []struct {
		name       string
		descriptor program.Descriptor
	}{
		{"Environment", catalogue.Environment},
		{"Database", catalogue.Database},
		{"API", catalogue.API},
		{"Reports", catalogue.Reports},
	} {
		if err := registry.Register(cataloguePath, entry.name, entry.descriptor); err != nil {
			return err
		}
	}
	file, err := text.Parse("service.sef", strings.NewReader(serviceSource))
	if err != nil {
		return err
	}
	graph, err := link.Resolve(file.Artifact().Definition, &registry)
	if err != nil {
		return err
	}
	instance, err := load.Instantiate(graph)
	if err != nil {
		return err
	}
	for invocation := range instance.Invocations() {
		if runner, ok := invocation.Runner().(loggerUser); ok {
			runner.UseLogger(logger)
		}
	}
	return playground.Run(ctx, instance, playground.Options{})
}

type loggerUser interface {
	UseLogger(*log.Logger)
}

const cataloguePath = "example.com/acme/catalogue"

//go:embed testdata/service.sef
var serviceSource string
