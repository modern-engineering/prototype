// Executable is a demonstrative command-line tool for loading one of multiple
// long-running programs in a single binary.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/samples/ffloader"
)

func main() {
	ffloader.Load(HelloWorld)
}

var HelloWorld = application.Register("hello-world", initHelloWorld)

func initHelloWorld(env *application.Environment) (application.MainFunc, error) {
	var (
		who      string
		interval time.Duration
	)

	fs := env.FlagSet()
	fs.StringVar(&who, "who", "World", "Who to greet")
	fs.DurationVar(&interval, "interval", time.Second, "Interval between greetings")

	if err := env.Parameterize(); err != nil {
		return nil, err
	}

	return func(ctx context.Context) {
		t := time.NewTicker(interval)
		for {
			select {
			case <-t.C:
				fmt.Println("Hello", who)
			case <-ctx.Done():
				return
			}
		}
	}, nil
}
