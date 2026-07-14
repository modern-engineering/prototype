// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Command host is the committed Mode-P demonstration: ONE prebuilt
// binary, built once against a fixed catalogue, that serves ANY
// solution image compiled against that catalogue. Application packages
// and solutions move at disparate cadences — the platform ships this
// binary; solution engineers ship images; nothing is generated or
// compiled per solution.
//
// The catalogue literal below is this binary's whole identity: the ff
// components and the substrate types, the same registrations `sdl
// build` discovers for the example solutions, so an image built there
// enacts here.
//
//	host -image <file> [-extern name=value]... [-externs file]... [-grace duration]
//
// -externs reads bindings from a file of name=value lines
// (host.ParseExterns) — the same namespace the -extern flag feeds, so
// an environment may mount values where an operator would type them;
// a name bound by two files is refused. Exit codes follow host.Main's
// contract: 0 clean, 1 wet failure, 2 configuration fault.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/substrate"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

// catalogue registers what this host binary links: every exported
// citizen of the example packages, under the identifiers solutions
// reference.
var catalogue = []solution.Package{
	{Path: "github.com/modern-engineering/prototype/examples/ff", Name: "ff", Elements: []solution.Element{
		solution.App("Ping", ff.Ping),
		solution.App("Pong", ff.Pong),
	}},
	{Path: "github.com/modern-engineering/prototype/examples/substrate", Name: "substrate", Elements: []solution.Element{
		solution.Provision("NATS", substrate.NATS),
		solution.Provision("Postgres", substrate.Postgres),
		solution.Provision("StandIn", substrate.StandIn),
		solution.Symbol("Endpoint", substrate.Endpoint),
		solution.Symbol("NATSCluster", substrate.NATSCluster),
		solution.Symbol("PostgresServer", substrate.PostgresServer),
		solution.Symbol("Secret", substrate.Secret),
	}},
}

func main() {
	path := flag.String("image", "", "path to the solution image to enact")
	grace := flag.Duration("grace", 10*time.Second, "graceful-shutdown budget after the first signal")
	externs := make(map[string]string)
	flag.Func("extern", "bind one extern symbol as `name=value` (repeatable)", func(arg string) error {
		name, value, ok := strings.Cut(arg, "=")
		if !ok || name == "" {
			return fmt.Errorf("%q is not name=value", arg)
		}
		externs[name] = value
		return nil
	})
	flag.Func("externs", "read extern bindings from `file` (name=value lines, repeatable)", func(path string) error {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		bound, err := host.ParseExterns(f)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		for name, value := range bound {
			if _, dup := externs[name]; dup {
				return fmt.Errorf("%s: extern %s already bound", path, name)
			}
			externs[name] = value
		}
		return nil
	})
	flag.Parse() // flag.ExitOnError: bad arguments exit 2 here
	if *path == "" {
		fmt.Fprintln(os.Stderr, "host: -image is required")
		os.Exit(2)
	}

	f, err := os.Open(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "host: %v\n", err)
		os.Exit(2)
	}
	img, err := image.Decode(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "host: %v\n", err)
		os.Exit(2)
	}

	os.Exit(host.Main(host.Config{
		Image:     img,
		Catalogue: catalogue,
		Externs:   externs,
		Grace:     *grace,
	}))
}
