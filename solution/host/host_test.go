// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host_test

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
)

// ----------------------------------------------------------------------------
// The live test catalogue: package "mill", shaped like the pingpong
// solution — one component, one attach-only provision type, a plain
// and a sensitive symbol type. The services and the driver report what
// the wet binding handed them through channels, so tests observe the
// values that actually flowed, not just the audit's word for them.

const millPath = "example.com/acme/mill"

// A report is what one echo service observed at run time: the values
// its own flags carried when Run was called.
type report struct {
	subject, nats string
	count         int
}

// echoDescriptor declares the component surface: subject and nats
// mirror the pingpong parameters, count bounds the work (negative
// runs until cancelled — the long-running shape), tone stays unbound
// everywhere to prove catalogue defaults survive the parse, and mode
// is a validate-only flag.Func slot that renders nothing back — the
// audit must still show the value it was fed.
func echoDescriptor(reports chan<- report) *application.Descriptor {
	return &application.Descriptor{
		Name: "echo",
		Doc:  "report the bound surface, then echo count times",
		Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
			fs := flag.NewFlagSet("echo", flag.ContinueOnError)
			subject := fs.String("subject", "ping", "subject to echo on")
			nats := fs.String("nats", "", "account config; empty runs unconnected")
			count := fs.Int("count", -1, "echoes before stopping; negative means forever")
			fs.String("tone", "flat", "tone nobody binds")
			fs.Func("mode", "echo mode; must not be empty", func(s string) error {
				if s == "" {
					return errors.New("must not be empty")
				}
				return nil
			})
			return application.RunnerFunc(func(ctx context.Context) error {
				reports <- report{subject: *subject, nats: *nats, count: *count}
				if *count < 0 {
					<-ctx.Done()
					return ctx.Err()
				}
				return nil
			}), fs
		}),
	}
}

// A standControl steers the Stand driver from a test: the failure to
// inject, the outputs to withhold, and the channel every Attach
// reports its wet-bound endpoint on.
type standControl struct {
	attachErr error
	withhold  map[string]bool
	got       chan string
}

func newStandControl() *standControl {
	return &standControl{got: make(chan string, 8)}
}

// A standDriver is one Stand instance: a real one-slot flag surface
// and an Attach that emits fixed outputs, standing in for substrate
// the way examples/substrate.StandIn does.
type standDriver struct {
	ctl      *standControl
	flags    *flag.FlagSet
	endpoint string
}

func (d *standDriver) Flags() *flag.FlagSet { return d.flags }

func (d *standDriver) Attach(_ context.Context, w *solution.OutputWriter) error {
	d.ctl.got <- d.endpoint
	if d.ctl.attachErr != nil {
		return d.ctl.attachErr
	}
	if !d.ctl.withhold["config"] {
		if err := w.SetString("config", "stand://mill"); err != nil {
			return err
		}
	}
	if !d.ctl.withhold["token"] {
		if err := w.SetString("token", "hush-hush"); err != nil {
			return err
		}
	}
	return nil
}

// standType declares the provision surface: one endpoint parameter,
// one plain output, one sensitive output the audit must never print.
func standType(ctl *standControl) *solution.ProvisionType {
	return &solution.ProvisionType{
		Doc: "a stand-in attachment to the mill's substrate",
		Make: func() solution.Provisioner {
			d := &standDriver{ctl: ctl, flags: flag.NewFlagSet("stand", flag.ContinueOnError)}
			d.flags.StringVar(&d.endpoint, "endpoint", "", "endpoint to pretend to attach to")
			return d
		},
		Outputs: []solution.Output{
			{Name: "config", Doc: "the coordinate consumers receive", Type: solution.OutputString},
			{Name: "token", Doc: "the credential consumers receive", Sensitive: true},
		},
		Kinds: solution.Attach,
	}
}

func millCatalogue(ctl *standControl, reports chan<- report) []solution.Package {
	return []solution.Package{{
		Path: millPath,
		Name: "mill",
		Elements: []solution.Element{
			solution.App("Echo", echoDescriptor(reports)),
			solution.Provision("Stand", standType(ctl)),
			solution.Symbol("Endpoint", &solution.SymbolType{Doc: "a site-bound coordinate"}),
			solution.Symbol("Secret", &solution.SymbolType{Doc: "an operator-held credential", Sensitive: true}),
		},
	}}
}

// ----------------------------------------------------------------------------
// Image crafting, down to what the host reads: the pinned symbol
// types externs resolve against, package names for rendering, and the
// records with their taints stamped the way the compiler stamps them.

func millImage(symbols []image.SymbolDef, records ...image.Record) *image.Image {
	if symbols == nil {
		symbols = []image.SymbolDef{}
	}
	return &image.Image{
		Format:     image.Format,
		Solution:   "millpond",
		Generation: 4,
		Catalogue: []image.Package{{
			Path: millPath,
			Name: "mill",
			Elements: []image.ElementSchema{
				{Name: "Echo", Kind: image.KindComponent},
				{Name: "Endpoint", Kind: image.KindSymbol},
				{Name: "Secret", Kind: image.KindSymbol, Sensitive: true},
				{Name: "Stand", Kind: image.KindProvision, Kinds: []string{image.KindAttach}},
			},
		}},
		Symbols: symbols,
		Records: records,
	}
}

func ref(name string) image.Ref { return image.Ref{Package: millPath, Name: name} }

func literal(key string, v *image.Value) image.Binding {
	return image.Binding{Key: key, Value: v, Source: image.SourceInstance}
}

func symbolRef(key, symbol string, sensitive bool) image.Binding {
	return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: symbol}, Source: image.SourceInstance, Sensitive: sensitive}
}

func outputRef(key, instance, output string, sensitive bool) image.Binding {
	return image.Binding{Key: key, Ref: &image.SymbolRef{Symbol: instance, Output: output}, Source: image.SourceInstance, Sensitive: sensitive}
}

func deployRec(name string, params ...image.Binding) image.Record {
	return image.Record{Verb: image.VerbDeploy, Element: ref("Echo"), Name: name, Params: params}
}

func attachRec(name string, params ...image.Binding) image.Record {
	return image.Record{Verb: image.VerbProvision, Kind: image.KindAttach, Element: ref("Stand"), Name: name, Params: params}
}

// pingpongImage is the M1 shape in miniature: an extern-fed provision
// whose outputs wire into deployed services next to literals, var
// references, and a sensitive extern. The provision record sits last
// so the run order below proves the phases, not the record order.
func pingpongImage() *image.Image {
	return millImage(
		[]image.SymbolDef{
			{Name: "endpoint", Class: image.ClassExtern, Type: &image.Ref{Package: millPath, Name: "Endpoint"}},
			{Name: "credential", Class: image.ClassExtern, Type: &image.Ref{Package: millPath, Name: "Secret"}},
			{Name: "subject", Class: image.ClassVar, Value: image.String("wheel")},
		},
		deployRec("Ping1",
			literal("subject", image.String("ping-1")),
			outputRef("nats", "standIn", "config", false),
			literal("count", image.Int(1)),
			literal("mode", image.String("loud")),
		),
		deployRec("Ping2",
			symbolRef("subject", "subject", false),
			outputRef("nats", "standIn", "token", true),
			literal("count", image.Int(1)),
		),
		deployRec("Pong",
			literal("subject", image.String("pond")),
			symbolRef("nats", "credential", true),
			literal("count", image.Int(1)),
		),
		attachRec("standIn", symbolRef("endpoint", "endpoint", false)),
	)
}

// externs binds the pingpong image's site: the plain coordinate and
// the credential no audit line may expose.
func externs() map[string]string {
	return map[string]string{"endpoint": "mill:4222", "credential": "swordfish"}
}

// ----------------------------------------------------------------------------
// Assertion helpers

// wantLine asserts the audit carries the exact line: matching whole
// lines rather than substrings pins the audit format itself.
func wantLine(t *testing.T, log, want string) {
	t.Helper()
	if !slicesContains(strings.Split(log, "\n"), want) {
		t.Errorf("audit log misses the line %q; log:\n%s", want, log)
	}
}

// wantErrLine asserts one line of a (possibly joined) error.
func wantErrLine(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want the line %q", want)
	}
	if !slicesContains(strings.Split(err.Error(), "\n"), want) {
		t.Errorf("error misses the line %q; got:\n%s", want, err)
	}
}

func slicesContains(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

// drain collects n reports; anything already buffered is enough since
// Run has returned by the time tests call it.
func drain(t *testing.T, reports <-chan report, n int) []report {
	t.Helper()
	got := make([]report, 0, n)
	for len(got) < n {
		select {
		case r := <-reports:
			got = append(got, r)
		default:
			t.Fatalf("got %d report(s), want %d", len(got), n)
		}
	}
	return got
}

// ----------------------------------------------------------------------------
// Run

// TestRunEnactsSolution walks the whole wet half on the M1 shape:
// PROVISION feeds the extern into the driver and stores its outputs,
// DEPLOY binds literals, var and extern references, and provision
// outputs into three concurrent services that run to completion, and
// the audit shows every resolved value with the tainted ones redacted.
func TestRunEnactsSolution(t *testing.T) {
	ctl := newStandControl()
	reports := make(chan report, 8)
	var log strings.Builder
	err := host.Run(t.Context(), host.Config{
		Image:     pingpongImage(),
		Catalogue: millCatalogue(ctl, reports),
		Externs:   externs(),
		Log:       &log,
	})
	if err != nil {
		t.Fatalf("Run() = %v; log:\n%s", err, log.String())
	}

	// The driver saw the extern value the site supplied.
	if got := <-ctl.got; got != "mill:4222" {
		t.Errorf("driver endpoint = %q, want %q", got, "mill:4222")
	}

	// Every service ran with the values the image wired.
	want := map[string]report{
		"ping-1": {subject: "ping-1", nats: "stand://mill", count: 1},
		"wheel":  {subject: "wheel", nats: "hush-hush", count: 1},
		"pond":   {subject: "pond", nats: "swordfish", count: 1},
	}
	for _, got := range drain(t, reports, 3) {
		w, ok := want[got.subject]
		if !ok {
			t.Errorf("unexpected report %+v", got)
			continue
		}
		if got != w {
			t.Errorf("report = %+v, want %+v", got, w)
		}
		delete(want, got.subject)
	}

	// The audit names every phase and value, provenance attached.
	audit := log.String()
	wantLine(t, audit, `enact millpond generation 4: 1 provision step(s), 3 deploy step(s)`)
	wantLine(t, audit, `audit: extern endpoint = "mill:4222" (site, mill.Endpoint)`)
	wantLine(t, audit, `provision standIn (attach mill.Stand)`)
	wantLine(t, audit, `audit: standIn.endpoint = "mill:4222" (extern endpoint, instance)`)
	wantLine(t, audit, `audit: output standIn.config = "stand://mill" (mill.Stand)`)
	wantLine(t, audit, `deploy Ping1 (mill.Echo)`)
	wantLine(t, audit, `audit: Ping1.nats = "stand://mill" (output standIn.config, instance)`)
	wantLine(t, audit, `audit: Ping1.subject = "ping-1" (literal, instance)`)
	wantLine(t, audit, `audit: Ping1.count = "1" (literal, instance)`)
	wantLine(t, audit, `audit: Ping1.tone = "flat" (catalogue default)`)
	// A validate-only slot renders nothing back; the audit shows what
	// its Set was fed all the same.
	wantLine(t, audit, `audit: Ping1.mode = "loud" (literal, instance)`)
	wantLine(t, audit, `audit: Ping2.subject = "wheel" (var subject, instance)`)
	wantLine(t, audit, `running 3 instance(s)`)

	// Taint redacts wherever the sensitive values surface (A-10)...
	wantLine(t, audit, `audit: extern credential = <redacted> (site, mill.Secret)`)
	wantLine(t, audit, `audit: output standIn.token = <redacted> (mill.Stand)`)
	wantLine(t, audit, `audit: Ping2.nats = <redacted> (output standIn.token, instance)`)
	wantLine(t, audit, `audit: Pong.nats = <redacted> (extern credential, instance)`)
	// ...so the values themselves appear nowhere.
	for _, secret := range []string{"hush-hush", "swordfish"} {
		if strings.Contains(audit, secret) {
			t.Errorf("audit log leaks %q:\n%s", secret, audit)
		}
	}
}

// TestRunEmptyImage: an empty image is a valid solution with nothing
// to do.
func TestRunEmptyImage(t *testing.T) {
	var log strings.Builder
	err := host.Run(t.Context(), host.Config{
		Image:     millImage(nil),
		Catalogue: millCatalogue(newStandControl(), nil),
		Log:       &log,
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	wantLine(t, log.String(), `running 0 instance(s)`)
}

// TestRunChainsProvisionOutputs wires one provision's output into the
// next one's parameter: the store must feed provision steps in plan
// order, not just deploys, and plan order must win over record order.
func TestRunChainsProvisionOutputs(t *testing.T) {
	ctl := newStandControl()
	img := millImage(nil,
		attachRec("second", outputRef("endpoint", "first", "config", false)),
		attachRec("first", literal("endpoint", image.String("root"))),
	)
	err := host.Run(t.Context(), host.Config{
		Image:     img,
		Catalogue: millCatalogue(ctl, nil),
		Log:       &strings.Builder{},
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if first, second := <-ctl.got, <-ctl.got; first != "root" || second != "stand://mill" {
		t.Errorf("driver endpoints = %q, %q; want %q then %q", first, second, "root", "stand://mill")
	}
}

// TestRunGatesUnboundExterns: the gate lists every miss at once, in
// the exact teaching vocabulary, and nothing wet runs.
func TestRunGatesUnboundExterns(t *testing.T) {
	ctl := newStandControl()
	reports := make(chan report, 8)
	var log strings.Builder
	err := host.Run(t.Context(), host.Config{
		Image:     pingpongImage(),
		Catalogue: millCatalogue(ctl, reports),
		Log:       &log,
	})
	wantErrLine(t, err, `unbound extern endpoint (mill.Endpoint): pass -extern endpoint=<value>`)
	wantErrLine(t, err, `unbound extern credential (mill.Secret): pass -extern credential=<value>`)
	if len(ctl.got) != 0 || len(reports) != 0 {
		t.Error("the gate let drivers or services run")
	}
	if audit := log.String(); strings.Contains(audit, "provision standIn") {
		t.Errorf("audit shows wet work past the gate:\n%s", audit)
	}
}

// TestRunReportsDriverRefusal: a refusing driver is a wet failure
// named after its step.
func TestRunReportsDriverRefusal(t *testing.T) {
	ctl := newStandControl()
	ctl.attachErr = errors.New("substrate said no")
	reports := make(chan report, 8)
	err := host.Run(t.Context(), host.Config{
		Image:     pingpongImage(),
		Catalogue: millCatalogue(ctl, reports),
		Externs:   externs(),
		Log:       &strings.Builder{},
	})
	wantErrLine(t, err, `provision standIn: substrate said no`)
	if len(reports) != 0 {
		t.Error("services ran after the PROVISION phase failed")
	}
}

// TestRunRequiresDeclaredOutputs: a driver that returns without
// writing its whole declared scheme fails its step, one line per
// missing output.
func TestRunRequiresDeclaredOutputs(t *testing.T) {
	ctl := newStandControl()
	ctl.withhold = map[string]bool{"token": true}
	err := host.Run(t.Context(), host.Config{
		Image:     millImage(nil, attachRec("standIn", literal("endpoint", image.String("e")))),
		Catalogue: millCatalogue(ctl, nil),
		Log:       &strings.Builder{},
	})
	wantErrLine(t, err, `provisioner for standIn did not write output token`)
}

// TestRunRefusesOpaqueTokens: a params token is the deployment
// compartments' vocabulary leaking into the binding namespace — a
// hand-built image's fault, refused at the boundary.
func TestRunRefusesOpaqueTokens(t *testing.T) {
	img := millImage(nil, deployRec("Ping1",
		image.Binding{Key: "nats", Value: image.Token("euCentral1"), Source: image.SourceInstance},
	))
	err := host.Run(t.Context(), host.Config{
		Image:     img,
		Catalogue: millCatalogue(newStandControl(), nil),
		Log:       &strings.Builder{},
	})
	wantErrLine(t, err, `Ping1: opaque token "euCentral1" reached wet binding for nats`)
}

// TestRunValidatesWetValues: the slot's own flag.Value.Set judges the
// rendered value at wet time (A-14), so a value no compiler vetted —
// here a string where an int flag lives — fails as configuration.
func TestRunValidatesWetValues(t *testing.T) {
	img := millImage(nil, deployRec("Ping1", literal("count", image.String("many"))))
	err := host.Run(t.Context(), host.Config{
		Image:     img,
		Catalogue: millCatalogue(newStandControl(), nil),
		Log:       &strings.Builder{},
	})
	if err == nil || !strings.Contains(err.Error(), "flag count") {
		t.Errorf("Run() = %v, want a wet Set refusal for flag count", err)
	}
}
