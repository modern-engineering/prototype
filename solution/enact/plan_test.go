// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package enact_test

import (
	"context"
	"flag"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/enact"
	"github.com/modern-engineering/prototype/solution/image"
)

// ----------------------------------------------------------------------------
// The live test catalogue: one package "kit" holding a component, a
// provisionable type, a driverless type, and two symbol types. The
// tests craft images against it by hand — enactment must stand on any
// image, not only what today's compiler emits.

const kitPath = "example.com/acme/kit"

func widgetDescriptor() *application.Descriptor {
	return &application.Descriptor{
		Name: "widget",
		Doc:  "a widget that waits",
		Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
			fs := flag.NewFlagSet("widget", flag.ContinueOnError)
			fs.String("target", "", "subject to work on")
			fs.String("nats", "", "bus configuration")
			return application.RunnerFunc(func(context.Context) error { return nil }), fs
		}),
	}
}

// A poolProvisioner is the test driver seam: a real flag surface, an
// Attach the dry half must never reach.
type poolProvisioner struct {
	flags      *flag.FlagSet
	size, seed string
}

func makePool() solution.Provisioner {
	p := &poolProvisioner{flags: flag.NewFlagSet("pool", flag.ContinueOnError)}
	p.flags.StringVar(&p.size, "size", "", "pool size")
	p.flags.StringVar(&p.seed, "seed", "", "seed configuration")
	return p
}

func (p *poolProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *poolProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	panic("planning must never run a driver")
}

func poolProvisionType() *solution.ProvisionType {
	return &solution.ProvisionType{
		Doc:  "a pool carved from shared substrate",
		Make: makePool,
		Outputs: []solution.Output{
			{Name: "config", Doc: "pool coordinates"},
			{Name: "ready", Doc: "warm-up state", Type: solution.OutputBool},
		},
		Kinds: solution.Slice | solution.Attach,
	}
}

// ghostProvisionType has no Make at all: a legal dry citizen —
// parameterless, driverless — that only enactment refuses.
func ghostProvisionType() *solution.ProvisionType {
	return &solution.ProvisionType{
		Doc:     "a type nobody wrote a driver for",
		Outputs: []solution.Output{{Name: "dsn"}},
		Kinds:   solution.Attach,
	}
}

// A kit is one live catalogue with the values it registers held on
// the side, so tests can assert steps resolve to the very pointers
// the catalogue carries.
type kit struct {
	widget *application.Descriptor
	pool   *solution.ProvisionType
	ghost  *solution.ProvisionType
	cat    []solution.Package
}

func newKit() *kit {
	k := &kit{
		widget: widgetDescriptor(),
		pool:   poolProvisionType(),
		ghost:  ghostProvisionType(),
	}
	k.cat = []solution.Package{{
		Path: kitPath,
		Name: "kit",
		Elements: []solution.Element{
			solution.App("Widget", k.widget),
			solution.Provision("Pool", k.pool),
			solution.Provision("Ghost", k.ghost),
			solution.Symbol("Endpoint", &solution.SymbolType{Doc: "a plain endpoint"}),
			solution.Symbol("Secret", &solution.SymbolType{Doc: "a credential", Sensitive: true}),
		},
	}}
	return k
}

// ref addresses a kit element from a record or symbol.
func ref(name string) image.Ref { return image.Ref{Package: kitPath, Name: name} }

// kitImage crafts an image against the kit catalogue: the pinned
// section a compilation would emit (down to what enactment reads —
// identity, kinds, sensitivity), plus the given symbols and records.
func kitImage(symbols []image.SymbolDef, records ...image.Record) *image.Image {
	if symbols == nil {
		symbols = []image.SymbolDef{}
	}
	return &image.Image{
		Format:     image.Format,
		Solution:   "demo",
		Generation: 3,
		Catalogue: []image.Package{{
			Path: kitPath,
			Name: "kit",
			Elements: []image.ElementSchema{
				{Name: "Endpoint", Kind: image.KindSymbol},
				{Name: "Ghost", Kind: image.KindProvision, Kinds: []string{image.KindAttach}},
				{Name: "Pool", Kind: image.KindProvision, Kinds: []string{image.KindSlice, image.KindAttach}},
				{Name: "Secret", Kind: image.KindSymbol, Sensitive: true},
				{Name: "Widget", Kind: image.KindComponent},
			},
		}},
		Symbols: symbols,
		Records: records,
	}
}

// Binding constructors, Source stamped the way the compiler stamps
// instance bindings.

func literal(key, value string) image.Binding {
	return image.Binding{Key: key, Value: image.String(value), Source: image.SourceInstance}
}

func outputRef(key, instance, output string) image.Binding {
	return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: instance, Output: output}, Source: image.SourceInstance}
}

func deployRec(name string, params ...image.Binding) image.Record {
	return image.Record{Verb: image.VerbDeploy, Element: ref("Widget"), Name: name, Params: params}
}

func attachRec(name string, params ...image.Binding) image.Record {
	return image.Record{Verb: image.VerbProvision, Kind: image.KindAttach, Element: ref("Pool"), Name: name, Params: params}
}

// provisionNames flattens a phase's step order for comparison.
func provisionNames(steps []enact.Step) []string {
	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Record.Name
	}
	return names
}

// wantFault loads an image expected to fail and asserts the joined
// error carries the exact fault line — vocabulary is contract, hosts
// and tests downstream match on the words.
func wantFault(t *testing.T, img *image.Image, cat []solution.Package, want string) {
	t.Helper()
	plan, err := enact.Load(img, cat)
	if err == nil {
		t.Fatalf("Load() = %+v, want fault %q", plan, want)
	}
	if plan != nil {
		t.Errorf("Load() returned a plan alongside the fault %v", err)
	}
	for _, line := range strings.Split(err.Error(), "\n") {
		if line == want {
			return
		}
	}
	t.Errorf("Load() faults:\n%s\nwant the line %q", err, want)
}

// ----------------------------------------------------------------------------
// Planning

// TestLoadResolvesSteps proves a step carries its record verbatim and
// the very live value the catalogue registers — the seam the wet half
// Makes instances through.
func TestLoadResolvesSteps(t *testing.T) {
	k := newKit()
	img := kitImage(nil,
		attachRec("pool1", literal("size", "3")),
		deployRec("w1", literal("target", "ping"), outputRef("nats", "pool1", "config")),
	)
	plan, err := enact.Load(img, k.cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if plan.Solution != "demo" || plan.Generation != 3 {
		t.Errorf("plan header = %s/%d, want demo/3", plan.Solution, plan.Generation)
	}
	if len(plan.Provisions) != 1 || len(plan.Deploys) != 1 {
		t.Fatalf("plan = %d provisions, %d deploys, want 1 and 1", len(plan.Provisions), len(plan.Deploys))
	}
	prov, dep := plan.Provisions[0], plan.Deploys[0]
	if prov.Provision != k.pool || prov.Component != nil {
		t.Errorf("provision step payload = (%p, %p), want the live Pool type and no component", prov.Provision, prov.Component)
	}
	if dep.Component != k.widget || dep.Provision != nil {
		t.Errorf("deploy step payload = (%p, %p), want the live Widget descriptor and no provision", dep.Component, dep.Provision)
	}
	if got := dep.Record.Params; len(got) != 2 || got[0].Key != "target" || got[1].Key != "nats" {
		t.Errorf("deploy step record params = %+v, want the record verbatim", got)
	}
}

// TestLoadOrdersProvisions pins the dependency order: a chain
// reverses image order, and every dependency precedes its dependents.
func TestLoadOrdersProvisions(t *testing.T) {
	k := newKit()
	img := kitImage(nil,
		attachRec("pa", outputRef("seed", "pb", "config")),
		attachRec("pb", outputRef("seed", "pc", "config")),
		attachRec("pc"),
		deployRec("w1", outputRef("nats", "pa", "config")),
	)
	plan, err := enact.Load(img, k.cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got, want := provisionNames(plan.Provisions), []string{"pc", "pb", "pa"}; !slices.Equal(got, want) {
		t.Errorf("provision order = %v, want %v", got, want)
	}
}

// TestLoadOrderBreaksTiesByImageOrder pins the tie-break semantics:
// at every extraction the earliest ready step in image record order
// runs next, so a step freed by a placement runs before later
// independent steps — not merely appended after the pass that freed
// it. Image order py (waits on px), px, pz: px is the earliest ready
// step, and placing it frees py, which outranks pz.
func TestLoadOrderBreaksTiesByImageOrder(t *testing.T) {
	k := newKit()
	img := kitImage(nil,
		attachRec("py", outputRef("seed", "px", "config")),
		attachRec("px"),
		attachRec("pz"),
	)
	plan, err := enact.Load(img, k.cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got, want := provisionNames(plan.Provisions), []string{"px", "py", "pz"}; !slices.Equal(got, want) {
		t.Errorf("provision order = %v, want %v", got, want)
	}
}

// TestLoadKeepsIndependentOrder: with no edges at all, both phases
// keep the image's record order.
func TestLoadKeepsIndependentOrder(t *testing.T) {
	k := newKit()
	img := kitImage(nil,
		attachRec("p2"),
		attachRec("p1"),
		deployRec("w2"),
		deployRec("w1"),
	)
	plan, err := enact.Load(img, k.cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got, want := provisionNames(plan.Provisions), []string{"p2", "p1"}; !slices.Equal(got, want) {
		t.Errorf("provision order = %v, want %v", got, want)
	}
	if got, want := provisionNames(plan.Deploys), []string{"w2", "w1"}; !slices.Equal(got, want) {
		t.Errorf("deploy order = %v, want %v", got, want)
	}
}

// TestLoadListsExterns: externs come out in symbol-table order with
// the pinned type identity and sensitivity; vars stay out of the
// list.
func TestLoadListsExterns(t *testing.T) {
	k := newKit()
	endpoint, secret := ref("Endpoint"), ref("Secret")
	img := kitImage([]image.SymbolDef{
		{Name: "apiKey", Class: image.ClassExtern, Type: &secret},
		{Name: "greeting", Class: image.ClassVar, Value: image.String("hello")},
		{Name: "natsEndpoint", Class: image.ClassExtern, Type: &endpoint},
	})
	plan, err := enact.Load(img, k.cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	want := []enact.Extern{
		{Name: "apiKey", Type: secret, Sensitive: true},
		{Name: "natsEndpoint", Type: endpoint},
	}
	if len(plan.Externs) != len(want) {
		t.Fatalf("externs = %+v, want %+v", plan.Externs, want)
	}
	for i, w := range want {
		if plan.Externs[i] != w {
			t.Errorf("externs[%d] = %+v, want %+v", i, plan.Externs[i], w)
		}
	}
}

// TestLoadEmptyImage: an image with no records plans to an empty,
// valid plan — nothing to run is not a fault.
func TestLoadEmptyImage(t *testing.T) {
	plan, err := enact.Load(kitImage(nil), newKit().cat)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if len(plan.Externs)+len(plan.Provisions)+len(plan.Deploys) != 0 {
		t.Errorf("plan = %+v, want empty", plan)
	}
}

// TestLoadNilImage: a nil image is refused, not dereferenced.
func TestLoadNilImage(t *testing.T) {
	if _, err := enact.Load(nil, newKit().cat); err == nil {
		t.Fatal("Load(nil) succeeded")
	}
}

// ----------------------------------------------------------------------------
// Refusals

// TestLoadRefusesSlices asserts the teaching error verbatim: slice
// records compile — citizenship is dry — and are refused only here.
func TestLoadRefusesSlices(t *testing.T) {
	k := newKit()
	img := kitImage(nil, image.Record{
		Verb:    image.VerbProvision,
		Kind:    image.KindSlice,
		Element: ref("Pool"),
		Name:    "poolSlice",
	})
	wantFault(t, img, k.cat,
		"slice provisioning is not implemented: poolSlice provisions kit.Pool as a slice (attach only)")
}

// TestLoadResolutionFaults drives the catalogue-drift refusals: an
// element the catalogue does not register, and records whose verb
// disagrees with the element's kind.
func TestLoadResolutionFaults(t *testing.T) {
	k := newKit()
	tests := []struct {
		name string
		rec  image.Record
		want string
	}{
		{
			"missing element",
			image.Record{Verb: image.VerbDeploy, Element: ref("Gone"), Name: "g1"},
			"image references kit.Gone but the catalogue registers no such element",
		},
		{
			"missing package",
			image.Record{Verb: image.VerbDeploy, Element: image.Ref{Package: "example.com/none", Name: "X"}, Name: "n1"},
			`image references "example.com/none".X but the catalogue registers no such element`,
		},
		{
			"deploy a provision type",
			image.Record{Verb: image.VerbDeploy, Element: ref("Pool"), Name: "w1"},
			"cannot deploy w1: kit.Pool is a provision type, not a component",
		},
		{
			"provision a component",
			image.Record{Verb: image.VerbProvision, Kind: image.KindAttach, Element: ref("Widget"), Name: "p1"},
			"cannot provision p1: kit.Widget is a component, not a provision type",
		},
		{
			"unknown verb",
			image.Record{Verb: "banana", Element: ref("Widget"), Name: "b1"},
			`record b1 declares unknown verb "banana"`,
		},
		{
			"unknown provision kind",
			image.Record{Verb: image.VerbProvision, Kind: "own", Element: ref("Pool"), Name: "p2"},
			`record p2 declares unknown provision kind "own"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantFault(t, kitImage(nil, tt.rec), k.cat, tt.want)
		})
	}
}

// TestLoadRefusesDuplicateInstances: instance names are the
// reconciliation keys; two records under one name cannot both hold.
func TestLoadRefusesDuplicateInstances(t *testing.T) {
	k := newKit()
	img := kitImage(nil, deployRec("w1"), deployRec("w1"))
	wantFault(t, img, k.cat, "image declares instance w1 twice")
}

// TestLoadRefusesCycles: the linker guarantees acyclic references, so
// a cycle marks a hand-crafted image; the plan refuses rather than
// guesses an order. The self-loop is the smallest case.
func TestLoadRefusesCycles(t *testing.T) {
	k := newKit()
	t.Run("pair", func(t *testing.T) {
		img := kitImage(nil,
			attachRec("p1", outputRef("seed", "p2", "config")),
			attachRec("p2", outputRef("seed", "p1", "config")),
		)
		wantFault(t, img, k.cat, "provision reference cycle: no order settles p1, p2")
	})
	t.Run("self", func(t *testing.T) {
		img := kitImage(nil, attachRec("p1", outputRef("seed", "p1", "config")))
		wantFault(t, img, k.cat, "provision reference cycle: no order settles p1")
	})
}
