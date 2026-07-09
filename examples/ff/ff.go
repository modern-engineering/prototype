// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package ff is the sample application catalogue: a pair of toy
// services whose exported descriptors demonstrate what solution
// tooling discovers and links against. Everything a catalogue citizen
// must honour is on display — a pure Make factory that declares flags
// and does nothing else, and parameter surfaces ranging from plain
// typed flags to a flag.Func with its own validation.
package ff

import (
	"context"
	"errors"
	"flag"
	"log"
	"time"

	"github.com/modern-engineering/prototype/application"
)

// Ping emits a payload to a target at a fixed cadence.
//
// Its parameter surface deliberately mixes flag styles: count and
// interval are ordinary typed flags with defaults, while target is a
// flag.Func flag that validates but never renders a value — proving
// that solution tooling depends only on flag.Value.Set, not on any
// flag's ability to print itself back.
var Ping = &application.Descriptor{
	Name: "ping",
	Doc: "ping emits a payload to a target at a fixed cadence\n\n" +
		"Each tick, ping sends one payload and logs the delivery. A negative\n" +
		"count keeps it pinging until its context is cancelled.",
	Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		fs := flag.NewFlagSet("ping", flag.ContinueOnError)
		count := fs.Int("count", -1, "number of pings; negative means forever")
		interval := fs.Duration("interval", time.Second, "delay between pings")
		var target string
		fs.Func("target", "destination to ping; must not be empty", func(s string) error {
			if s == "" {
				return errors.New("must not be empty")
			}
			target = s
			return nil
		})
		return application.RunnerFunc(func(ctx context.Context) error {
			tick := time.NewTicker(*interval)
			defer tick.Stop()
			for sent := 0; *count < 0 || sent < *count; sent++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-tick.C:
					log.Printf("ping %s #%d", target, sent+1)
				}
			}
			return nil
		}), fs
	}),
}

// Pong answers pings on a subject.
var Pong = &application.Descriptor{
	Name: "pong",
	Doc: "pong answers pings on a subject\n\n" +
		"Pong waits for payloads on its subject and answers each one until\n" +
		"its context is cancelled.",
	Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		fs := flag.NewFlagSet("pong", flag.ContinueOnError)
		subject := fs.String("subject", "ping", "subject to answer on")
		return application.RunnerFunc(func(ctx context.Context) error {
			log.Printf("pong answering on %s", *subject)
			<-ctx.Done()
			return ctx.Err()
		}), fs
	}),
}
