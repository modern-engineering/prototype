// Executable is a demonstrative command-line tool for loading one of multiple
// long-running programs in a single binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/samples/ffloader"
)

func main() {
	ffloader.Load(&HelloWorld)
	ffloader.Load(GoodbyeWorld)
}

// HelloWorld is a demonstrative long-running program that shows the most minimal
// syntax possible.
var HelloWorld = application.Descriptor{
	Name: "hello-world",
	Doc:  "",
	Make: application.MakeFor[HelloWorldApp](),
}

type HelloWorldApp struct {
	//flags    func() *flag.FlagSet
	who      string
	interval time.Duration

	application.ShutdownSignal
}

func (a *HelloWorldApp) Run(ctx context.Context) error {
	defer a.ShutdownSignal.Done()
	t := time.NewTicker(a.interval)
	for {
		select {
		case <-t.C:
			fmt.Println("Hello", a.who)
		case <-a.ShutdownSignal.C():
			fmt.Println("Graceful bye-bye")
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *HelloWorldApp) Flags() *flag.FlagSet {
	//a.flags = sync.OnceValue(func() *flag.FlagSet {
	var fs flag.FlagSet
	fs.StringVar(&a.who, "who", "World", "Who to greet")
	fs.DurationVar(&a.interval, "interval", time.Second, "Interval between greetings")
	return &fs
	//})
	//return a.flags()
}

// GoodbyeWorld is a similar long-running program that shows the most verbose
// syntax possible, with inline definitions for the moving parts.
var GoodbyeWorld = &application.Descriptor{
	Name: "goodbye-world",
	Doc:  "",
	Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		var (
			fs       flag.FlagSet
			who      string
			interval time.Duration
		)
		fs.StringVar(&who, "who", "World", "Who to greet")
		fs.DurationVar(&interval, "interval", time.Second, "Interval between greetings")
		return application.RunnerFunc(func(ctx context.Context) error {
			t := time.NewTicker(interval)
			for {
				select {
				case <-t.C:
					fmt.Println("Goodbye", who)
				case <-ctx.Done():
					return nil
				}
			}
		}), &fs
	}),
}
