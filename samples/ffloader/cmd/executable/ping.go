package main

import (
	"context"
	"flag"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
)

var PingApp = application.Descriptor{
	Name: "ping",
	Doc:  "",
	Make: application.MakeFor[Ping](),
}

type Ping struct {
	fs     flag.FlagSet
	fsOnce sync.Once

	Count    int
	Interval time.Duration
	Target   url.URL
}

func (p *Ping) Flags() *flag.FlagSet {
	p.fsOnce.Do(func() {
		p.fs.IntVar(&p.Count, "count", 0, "number of pings")
		p.fs.DurationVar(&p.Interval, "interval", 0, "interval between pings")
		p.fs.Func("target", "target URL", func(s string) error {
			u, err := url.Parse(s)
			if err != nil {
				return err
			}
			p.Target = *u
			return nil
		})
	})
	return &p.fs
}

func (p *Ping) Run(ctx context.Context) error {
	cycle := time.NewTicker(p.Interval)
	for range p.loop(ctx) {
		select {
		case <-cycle.C:
			p.ping(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *Ping) loop(ctx context.Context) iter.Seq[int] {
	if p.Count == -1 {
		return application.Loop(ctx)
	}
	return application.LoopCount(ctx, p.Count)
}

func (p *Ping) ping(ctx context.Context) {
	logger := slog.With("when", time.Now())

	resp, err := http.Get(p.Target.String())
	if err != nil {
		logger.ErrorContext(ctx, "Ping failed", "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to read response body", "error", err)
		return
	}
	logger.InfoContext(ctx, "Ping succeeded", slog.Group("response",
		"status", resp.Status,
		"code", resp.StatusCode,
	))
	slog.DebugContext(ctx, "Details of ping", "body", string(body))
}

type PingComponent struct {
}

func (p *PingComponent) ComponentDefinition(d *solution.Designer, w solution.CanvasWriter) error {
	spec := w.WriteApplication(PingApp)
	flag.fla
}
