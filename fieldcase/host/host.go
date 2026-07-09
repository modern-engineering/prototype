// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/loader/loaderflags"
	"github.com/modern-engineering/prototype/application/parameter"
	"github.com/modern-engineering/prototype/fieldcase/host/driver"
	"github.com/modern-engineering/prototype/fieldcase/host/site"
)

// The exit codes, a first cut of A-12's taxonomy: configuration faults
// must stay distinguishable from runtime failures so restart policies
// can tell "do not restart" from "do restart".
const (
	exitOK      = 0 // clean shutdown
	exitRuntime = 1 // an instance failed while running
	exitConfig  = 2 // site, gate, flag, or drift faults before running
)

// Main runs the baked solution: it is the generated main's whole body.
// Command-line arguments are read from os.Args; positional arguments
// name site files (merged, duplicate keys rejected). Diagnostics and
// the effective-value audit go to stderr; stdout stays reserved for
// measurements (A-12's stream discipline). The return value is the
// process exit code.
func Main(cfg *Config) int {
	log.SetFlags(0)
	log.SetPrefix(cfg.Solution + "-host: ")

	// The host's own surface: process flags, per-instance enable
	// flags, and one override flag per instance parameter, all with
	// ff-style env aliases (see the package comment's name mapping).
	fs := flag.NewFlagSet(cfg.Solution+"-host", flag.ContinueOnError)
	health := fs.String("health", envDefault("HEALTH", ":3000"), "listen address of the health endpoint (env HEALTH)")
	deploymentID := fs.String("deployment-id", envDefault("DEPLOYMENT_ID", cfg.Solution),
		"deployment identity namespacing consumer groups as <deployment-id>/<instance> (env DEPLOYMENT_ID)")
	grace := fs.Duration("grace", 20*time.Second, "graceful-shutdown budget before runners are cancelled")
	enables := make(map[string]*enableFlag, len(cfg.Instances))
	overrides := make(map[string]map[string]string, len(cfg.Instances))
	for _, inst := range cfg.Instances {
		e := &enableFlag{}
		enables[inst.Name] = e
		fs.Var(e, inst.Name, fmt.Sprintf("enable or disable instance %s (env %s; any true: only true ones run; any false: all but false ones run)",
			inst.Name, EnvName(inst.Name)))
		over := make(map[string]string)
		overrides[inst.Name] = over
		if flags := inst.App.Flags(); flags != nil {
			flags.VisitAll(func(f *flag.Flag) {
				name := inst.Name + "." + f.Name
				fs.Func(name, f.Usage+" (env "+EnvName(name)+")", func(v string) error {
					over[f.Name] = v
					return nil
				})
			})
		}
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return exitConfig
	}
	for name, e := range enables {
		if e.set {
			continue
		}
		if v, ok := os.LookupEnv(EnvName(name)); ok && v != "" {
			if err := e.Set(v); err != nil {
				log.Printf("env %s: %v", EnvName(name), err)
				return exitConfig
			}
		}
	}

	// Site configuration: the stage-(c) input.
	st, err := site.Load(fs.Args()...)
	if err != nil {
		log.Print(err)
		return exitConfig
	}
	if faults := cfg.CheckSite(st); len(faults) > 0 {
		for _, f := range faults {
			log.Printf("site: %s", f)
		}
		return exitConfig
	}

	enabled := cfg.selectInstances(enables)
	if len(enabled) == 0 {
		log.Print("no instances selected: the enable flags exclude every instance")
		return exitConfig
	}
	log.Printf("hosting %d of %d instances: %v (deployment id %s)", len(enabled), len(cfg.Instances), enabled, *deploymentID)

	// The required-extern gate (stage-(c) enforcement): every extern
	// this process consumes must be bound before any Runner starts.
	needs := cfg.Needs(enabled)
	if missing := cfg.UnboundExterns(st, needs); len(missing) > 0 {
		for _, sym := range missing {
			log.Printf("unbound extern %s (%s): the site must bind extern.%s", sym.Name, sym.Type, sym.Name)
		}
		log.Printf("%d unbound extern(s); refusing to start", len(missing))
		return exitConfig
	}

	// Effective symbol values: externs from the site, vars from the
	// site override or the image default.
	symbols := make(map[string]effectiveValue, len(cfg.Symbols))
	for _, sym := range cfg.Symbols {
		switch sym.Class {
		case ClassExtern:
			if !needs.Externs[sym.Name] {
				continue // not consumed by this process; possibly unbound
			}
			v, _ := st.Extern(sym.Name)
			symbols[sym.Name] = effectiveValue{value: v, source: "site", sensitive: sym.Sensitive, typ: sym.Type}
		case ClassVar:
			if v, ok := st.Var(sym.Name); ok {
				symbols[sym.Name] = effectiveValue{value: v, source: "site override"}
			} else {
				symbols[sym.Name] = effectiveValue{value: sym.Default, source: "image default"}
			}
		}
	}

	// Provisions: reconcile through the driver hook, in dependency
	// order (the compiler guarantees a DAG, so passes terminate).
	ctx := context.Background()
	outputs := make(map[string]map[string]string, len(cfg.Provisions))
	pending := make([]*Provision, 0, len(cfg.Provisions))
	for i := range cfg.Provisions {
		if needs.Provisions[cfg.Provisions[i].Name] {
			pending = append(pending, &cfg.Provisions[i])
		}
	}
	for len(pending) > 0 {
		progressed := false
		var next []*Provision
		for _, p := range pending {
			params, ok := resolveParams(p.Params, symbols, outputs)
			if !ok {
				next = append(next, p)
				continue
			}
			progressed = true
			rec := driver.Record{
				Name: p.Name, Type: p.Type, Kind: p.Kind,
				Site: st.Driver(p.Name),
				Need: sortedKeys(needs.Outputs[p.Name]),
			}
			outs, err := p.Driver.Provision(ctx, rec, params)
			if err != nil {
				log.Printf("provision %s: %v", p.Name, err)
				return exitConfig
			}
			for _, name := range rec.Need {
				if _, ok := outs[name]; !ok {
					log.Printf("provision %s: driver resolved no %q output", p.Name, name)
					return exitConfig
				}
			}
			outputs[p.Name] = outs
		}
		if !progressed {
			log.Printf("internal: provision references do not resolve (image cycle?)")
			return exitConfig
		}
		pending = next
	}

	// The A-10 audit, part one: every symbol and consumed output this
	// process resolved, sensitive values redacted.
	for _, sym := range cfg.Symbols {
		ev, ok := symbols[sym.Name]
		if !ok {
			continue
		}
		suffix := ""
		if ev.typ != "" {
			suffix = ", " + ev.typ
		}
		log.Printf("audit: %s %s = %s (%s%s)", sym.Class, sym.Name, ev.render(), ev.source, suffix)
	}
	for _, p := range cfg.Provisions {
		if !needs.Provisions[p.Name] {
			continue
		}
		sensitive := make(map[string]bool, len(p.Outputs))
		for _, out := range p.Outputs {
			sensitive[out.Name] = out.Sensitive
		}
		for _, name := range sortedKeys(needs.Outputs[p.Name]) {
			ev := effectiveValue{value: outputs[p.Name][name], sensitive: sensitive[name]}
			log.Printf("audit: output %s.%s = %s (driver, %s %s)", p.Name, name, ev.render(), p.Kind, p.Type)
		}
	}

	// Per-instance services: dry-construct, copy the flag surface into
	// a loaderflags parameter set (sharing flag.Values, so required
	// marks and the runner's closures see the bound values), and parse
	// the three sources in priority order.
	type running struct {
		name string
		svc  application.Service
	}
	var services []running
	var gateFaults []string
	for _, inst := range cfg.Instances {
		if !contains(enabled, inst.Name) {
			continue
		}
		svc := inst.App.Make()
		flags := svc.Flags()
		ps := loaderflags.NewFlagSet(inst.Name)
		if flags != nil {
			flags.VisitAll(func(f *flag.Flag) {
				ps.Var(f.Value, f.Name, f.Usage)
			})
		}

		// Compiled-image-vs-binary drift check: every baked binding
		// must land on a declared flag (digest-deployment §4(a).7).
		recordValues := make(map[string]string, len(inst.Params))
		recordDetails := make(map[string]string, len(inst.Params))
		sensitive := make(map[string]bool, len(inst.Params))
		for _, b := range inst.Params {
			if ps.Lookup(b.Key) == nil {
				log.Printf("%s: image binds %q but descriptor %s declares no such flag (image/catalogue drift)", inst.Name, b.Key, inst.App.Name)
				return exitConfig
			}
			v, ok := bindingValue(b, symbols, outputs)
			if !ok {
				log.Printf("%s: binding %s did not resolve", inst.Name, b.Key)
				return exitConfig
			}
			recordValues[b.Key] = v
			recordDetails[b.Key] = bindingDetail(b)
			sensitive[b.Key] = b.Sensitive
		}

		cmdline := &mapSource{label: "command line", values: overrides[inst.Name], applied: map[string]string{}}
		env := &mapSource{label: "environment", values: envValues(inst.Name, ps), applied: map[string]string{}}
		record := &mapSource{label: "image record", values: recordValues, details: recordDetails, applied: map[string]string{}}
		if err := loaderflags.Parse(ps, cmdline, env, record); err != nil {
			log.Printf("%s: %v", inst.Name, err)
			return exitConfig
		}

		// The required-parameter gate: parameter.Require marks with no
		// value from any source fail startup (loaderflags.PendingParams
		// over the shared values).
		for f := range loaderflags.PendingParams(ps) {
			if parameter.IsRequired(f.Value) {
				gateFaults = append(gateFaults, fmt.Sprintf("%s.%s is required and unbound", inst.Name, f.Name))
			}
		}

		// The A-10 audit, part two: the effective value of every flag
		// of every hosted instance, with its winning source.
		ps.VisitAll(func(f *flag.Flag) {
			detail, ok := firstApplied(f.Name, cmdline, env, record)
			if !ok {
				detail = "catalogue default"
			}
			ev := effectiveValue{value: f.Value.String(), sensitive: sensitive[f.Name]}
			log.Printf("audit: %s.%s = %s (%s)", inst.Name, f.Name, ev.render(), detail)
		})
		log.Printf("identity: %s consumer group = %s/%s", inst.Name, *deploymentID, inst.Name)

		services = append(services, running{name: inst.Name, svc: svc})
	}
	if len(gateFaults) > 0 {
		for _, f := range gateFaults {
			log.Print(f)
		}
		log.Printf("%d required parameter(s) unbound; refusing to start", len(gateFaults))
		return exitConfig
	}

	// Health: bind the listener before any Runner starts (A-12: a
	// handle is given, never a port assumed), so an occupied address
	// fails as configuration, and log the resolved coordinate.
	ln, err := net.Listen("tcp", *health)
	if err != nil {
		log.Printf("health: %v", err)
		return exitConfig
	}
	state := &healthState{}
	srv := &http.Server{Handler: state.handler()}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("health: %v", err)
		}
	}()
	defer srv.Close()
	log.Printf("health: listening on %s (/healthz, /readyz)", ln.Addr())

	// Run. The runtime's context is the hard-cancel line; signals do
	// NOT pre-cancel it — shutdown is Shutdown(ctx), then Cancel(),
	// then Wait(), in that order (the digest-application §4 trap:
	// Shutdown alone never releases runners blocked on their context).
	rt := application.RuntimeWithContext(context.Background())
	for _, r := range services {
		rt.Run(named{name: r.name, Runner: r.svc})
	}
	state.watch(rt)
	state.ready.Store(true)
	log.Printf("running %d instance(s); ready", len(services))

	waited := make(chan error, 1)
	go func() { waited <- rt.Wait() }()

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-waited:
		// No signal: an instance stopped on its own; that is a runtime
		// failure for a host of long-running services.
		state.ready.Store(false)
		log.Printf("instance terminated prematurely: %v", err)
		return exitRuntime
	case sig := <-signals:
		log.Printf("received %s; shutting down (grace %s)", sig, *grace)
		state.ready.Store(false) // stop admitting: readiness drops, liveness holds
		state.draining.Store(true)
		go func() {
			sig := <-signals
			log.Printf("received second %s; cancelling immediately", sig)
			rt.Cancel()
		}()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), *grace)
		if err := rt.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		cancel()
		rt.Cancel()
		err := <-waited
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, application.ErrCanceled) {
			log.Printf("shutdown finished with failure: %v", err)
			return exitRuntime
		}
		log.Print("shutdown complete")
		return exitOK
	}
}

// An effectiveValue is one resolved value with its audit rendering.
type effectiveValue struct {
	value     string
	source    string
	sensitive bool
	typ       string
}

// render quotes the value, or redacts it when the taint says so: the
// audit must prove a value was resolved without exposing it.
func (ev effectiveValue) render() string {
	if ev.sensitive {
		return "<redacted>"
	}
	return strconv.Quote(ev.value)
}

// resolveParams renders a provision's parameters to strings; ok is
// false while a referenced output is not yet resolved.
func resolveParams(params []Binding, symbols map[string]effectiveValue, outputs map[string]map[string]string) (map[string]string, bool) {
	m := make(map[string]string, len(params))
	for _, b := range params {
		v, ok := bindingValue(b, symbols, outputs)
		if !ok {
			return nil, false
		}
		m[b.Key] = v
	}
	return m, true
}

// bindingValue resolves one binding against the effective symbols and
// the provision outputs resolved so far.
func bindingValue(b Binding, symbols map[string]effectiveValue, outputs map[string]map[string]string) (string, bool) {
	switch b.Kind {
	case BindLiteral:
		return b.Value, true
	case BindSymbol:
		ev, ok := symbols[b.Value]
		return ev.value, ok
	case BindOutput:
		outs, ok := outputs[b.Instance]
		if !ok {
			return "", false
		}
		v, ok := outs[b.Output]
		return v, ok
	}
	return "", false
}

// bindingDetail renders a binding's provenance for the audit.
func bindingDetail(b Binding) string {
	switch b.Kind {
	case BindLiteral:
		return fmt.Sprintf("record literal, %s", b.Source)
	case BindSymbol:
		return fmt.Sprintf("record ref %s, %s", b.Value, b.Source)
	case BindOutput:
		return fmt.Sprintf("record output %s.%s, %s", b.Instance, b.Output, b.Source)
	}
	return b.Source
}

// envValues collects the instance's environment overrides: for every
// declared flag, the value of UPPER(instance_flag) if present.
func envValues(instance string, fs *flag.FlagSet) map[string]string {
	values := make(map[string]string)
	fs.VisitAll(func(f *flag.Flag) {
		if v, ok := os.LookupEnv(EnvName(instance + "." + f.Name)); ok {
			values[f.Name] = v
		}
	})
	return values
}

// A mapSource is a loaderflags.Parser feeding a flag set from a map,
// honoring the earlier-source-wins contract and recording what it
// applied for the audit.
type mapSource struct {
	label   string
	values  map[string]string
	details map[string]string // optional per-flag audit detail
	applied map[string]string // flag -> audit detail actually applied
}

func (s *mapSource) ParseParameters(fs *flag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}
		v, ok := s.values[f.Name]
		if !ok || loaderflags.IsSet(fs, f.Name) {
			return
		}
		if e := fs.Set(f.Name, v); e != nil {
			err = fmt.Errorf("%s: flag %s: %v", s.label, f.Name, e)
			return
		}
		detail := s.details[f.Name]
		if detail == "" {
			detail = s.label
		}
		s.applied[f.Name] = detail
	})
	return err
}

func (s *mapSource) String() string { return s.label }

// firstApplied returns the audit detail of the highest-priority source
// that set the flag.
func firstApplied(name string, sources ...*mapSource) (string, bool) {
	for _, s := range sources {
		if d, ok := s.applied[name]; ok {
			return d, true
		}
	}
	return "", false
}

// An enableFlag is the tri-state per-instance switch: unset, true, or
// false, with the host's selection semantics.
type enableFlag struct {
	set   bool
	value bool
}

func (e *enableFlag) String() string {
	if e == nil || !e.set {
		return ""
	}
	return strconv.FormatBool(e.value)
}

func (e *enableFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	e.set, e.value = true, v
	return nil
}

func (e *enableFlag) IsBoolFlag() bool { return true }

// named gives a service its instance name for logs and process
// listings, delegating the shutdown capability when the underlying
// runner has one.
type named struct {
	name string
	application.Runner
}

func (n named) String() string { return n.name }

func (n named) Shutdown(ctx context.Context) error {
	if s, ok := n.Runner.(application.Shutdowner); ok {
		return s.Shutdown(ctx)
	}
	return nil
}

// healthState aggregates instance health for the HTTP endpoint:
// readiness is "every instance launched and none stopped", liveness
// tolerates the deliberate stop of a drain.
type healthState struct {
	ready    atomic.Bool
	draining atomic.Bool
	rt       atomic.Pointer[application.Runtime]
}

func (h *healthState) watch(rt *application.Runtime) { h.rt.Store(rt) }

// dead reports whether any instance proc completed while the host was
// not draining.
func (h *healthState) dead() bool {
	rt := h.rt.Load()
	if rt == nil {
		return false
	}
	for _, p := range rt.Running() {
		select {
		case <-p.Done():
			if !h.draining.Load() {
				return true
			}
		default:
		}
	}
	return false
}

func (h *healthState) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if h.dead() {
			http.Error(w, "an instance died", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !h.ready.Load() || h.dead() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	return mux
}

// envDefault reads an environment default for a host flag.
func envDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// contains reports whether names includes name.
func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
