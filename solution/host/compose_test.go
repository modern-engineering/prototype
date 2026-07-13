// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The peer-frontend proof: a solution image composed by struct
// literal — no SDL text anywhere in its life — plans through
// enact.Load and runs to completion on the wet path, against the same
// live ff and substrate catalogue the compiled examples enact against.
// Programmatic composition is a frontend in its own right exactly
// because this seam cannot tell the difference.

package host_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/substrate"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/enact"
	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

const (
	ffPath        = "github.com/modern-engineering/prototype/examples/ff"
	substratePath = "github.com/modern-engineering/prototype/examples/substrate"
)

// liveCatalogue registers the example packages whole, the same
// registrations examples/host links and sdl build discovers — the
// composed image must meet the real catalogue, not one trimmed to fit.
func liveCatalogue() []solution.Package {
	return []solution.Package{
		{Path: ffPath, Name: "ff", Elements: []solution.Element{
			solution.App("Ping", ff.Ping),
			solution.App("Pong", ff.Pong),
		}},
		{Path: substratePath, Name: "substrate", Elements: []solution.Element{
			solution.Provision("NATS", substrate.NATS),
			solution.Provision("Postgres", substrate.Postgres),
			solution.Provision("StandIn", substrate.StandIn),
			solution.Symbol("Endpoint", substrate.Endpoint),
			solution.Symbol("NATSCluster", substrate.NATSCluster),
			solution.Symbol("PostgresServer", substrate.PostgresServer),
			solution.Symbol("Secret", substrate.Secret),
		}},
	}
}

// composedImage builds a count:1 pingpong by struct literal: a
// stand-in provision fed by the site's extern, one ping wired through
// the provision output and a var, one all-literal ping — each with
// count 1 and a short interval, so the whole solution runs to
// completion and Run returning nil is the proof.
func composedImage() *image.Image {
	symbol := func(key, name string) image.Binding {
		return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: name}, Source: image.SourceInstance}
	}
	output := func(key, instance, output string) image.Binding {
		return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: instance, Output: output}, Source: image.SourceInstance}
	}
	return &image.Image{
		Format:     image.Format,
		Solution:   "pingpong",
		Generation: 1,
		Catalogue: []image.Package{
			{Path: ffPath, Name: "ff", Elements: []image.ElementSchema{
				{Name: "Ping", Kind: image.KindComponent},
			}},
			{Path: substratePath, Name: "substrate", Elements: []image.ElementSchema{
				{Name: "Endpoint", Kind: image.KindSymbol},
				{Name: "StandIn", Kind: image.KindProvision,
					Outputs: []image.OutputSchema{{Name: "config", Type: "string"}},
					Kinds:   []string{image.KindAttach}},
			}},
		},
		Symbols: []image.SymbolDef{
			{Name: "echoSubject", Class: image.ClassVar, Value: image.String("ping")},
			{Name: "natsEndpoint", Class: image.ClassExtern, Type: &image.Ref{Package: substratePath, Name: "Endpoint"}},
		},
		Records: []image.Record{
			{
				Verb:    image.VerbProvision,
				Kind:    image.KindAttach,
				Element: image.Ref{Package: substratePath, Name: "StandIn"},
				Name:    "natsStandIn",
				Params:  []image.Binding{symbol("endpoint", "natsEndpoint")},
			},
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: ffPath, Name: "Ping"},
				Name:    "Ping1",
				Params: []image.Binding{
					{Key: "count", Value: image.Int(1), Source: image.SourceInstance},
					{Key: "interval", Value: image.Duration(10 * time.Millisecond), Source: image.SourceInstance},
					output("nats", "natsStandIn", "config"),
					symbol("target", "echoSubject"),
				},
			},
			{
				Verb:    image.VerbDeploy,
				Element: image.Ref{Package: ffPath, Name: "Ping"},
				Name:    "Ping2",
				Params: []image.Binding{
					{Key: "count", Value: image.Int(1), Source: image.SourceInstance},
					{Key: "interval", Value: image.Duration(10 * time.Millisecond), Source: image.SourceInstance},
					{Key: "target", Value: image.String("pong"), Source: image.SourceInstance},
				},
			},
		},
	}
}

// The peer-frontend proof runs end to end: the composed image passes
// the composer's own gates (Canonicalize, Validate), plans through
// enact.Load against the live catalogue, and the wet path provisions
// the real StandIn driver, wires its output into the services, and
// runs the solution to completion.
func TestRunComposedImage(t *testing.T) {
	img := composedImage()
	img.Canonicalize()
	if err := img.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	plan, err := enact.Load(img, liveCatalogue())
	if err != nil {
		t.Fatalf("enact.Load rejected the composed image: %v", err)
	}
	if len(plan.Externs) != 1 || plan.Externs[0].Name != "natsEndpoint" {
		t.Errorf("plan externs = %+v, want the one natsEndpoint", plan.Externs)
	}
	if len(plan.Provisions) != 1 || len(plan.Deploys) != 2 {
		t.Errorf("plan = %d provision(s), %d deploy(s), want 1 and 2", len(plan.Provisions), len(plan.Deploys))
	}

	var log strings.Builder
	err = host.Run(t.Context(), host.Config{
		Image:     img,
		Catalogue: liveCatalogue(),
		Externs:   map[string]string{"natsEndpoint": "localhost:0"},
		Log:       &log,
	})
	if err != nil {
		t.Fatalf("Run() = %v; log:\n%s", err, log.String())
	}

	audit := log.String()
	wantLine(t, audit, `enact pingpong generation 1: 1 provision step(s), 2 deploy step(s)`)
	wantLine(t, audit, `audit: extern natsEndpoint = "localhost:0" (site, substrate.Endpoint)`)
	wantLine(t, audit, `provision natsStandIn (attach substrate.StandIn)`)
	wantLine(t, audit, `audit: natsStandIn.endpoint = "localhost:0" (extern natsEndpoint, instance)`)
	wantLine(t, audit, fmt.Sprintf(`audit: output natsStandIn.config = %q (substrate.StandIn)`, substrate.StandInConfig))
	wantLine(t, audit, `deploy Ping1 (ff.Ping)`)
	wantLine(t, audit, fmt.Sprintf(`audit: Ping1.nats = %q (output natsStandIn.config, instance)`, substrate.StandInConfig))
	wantLine(t, audit, `audit: Ping1.target = "ping" (var echoSubject, instance)`)
	wantLine(t, audit, `audit: Ping1.count = "1" (literal, instance)`)
	wantLine(t, audit, `audit: Ping2.target = "pong" (literal, instance)`)
	wantLine(t, audit, `running 2 instance(s)`)
}
