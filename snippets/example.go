package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/modern-engineering/prototype/application"
)

func main() {
	application.LoadHelp(PingApp)
	application.LoadApp(context.Background(), PingApp)
}

var PingApp = application.Register("ping", NewPing)

func NewPing(env *application.Environment) (application.MainFunc, error) {
	var (
		count    int
		payload  string
		interval time.Duration
	)
	params := env.FlagSet()
	params.IntVar(&count, "count", 0, "TODO")
	params.StringVar(&payload, "payload", "Hello, World!", "TODO")
	params.DurationVar(&interval, "interval", time.Second, "TODO")
	if err := env.Parameterize(); err != nil {
		return nil, err
	}

	return func(ctx context.Context) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for i := 0; i < count && count > 0; i++ {
			select {
			case <-ticker.C:
				fmt.Println("Ping:", payload)
			case <-ctx.Done():
				slog.DebugContext(ctx, "terminated", slog.Any("cause", context.Cause(ctx)))
			}
		}
		slog.DebugContext(ctx, "done")
	}, nil
}
