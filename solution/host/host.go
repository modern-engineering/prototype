// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/loader/loaderflags"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/enact"
	"github.com/modern-engineering/prototype/solution/image"
)

// A Config carries one hosting: the image to enact, the live catalogue
// this binary links, and the site's knobs.
type Config struct {
	// Image is the desired state to enact.
	Image *image.Image

	// Catalogue registers the element packages linked into this
	// binary — the citizens of the image's own pinned catalogue.
	// [enact.Load] holds the image to it before anything wet runs.
	Catalogue []solution.Package

	// Externs are the operator-supplied values of the image's extern
	// symbols, keyed by symbol name. They arrive in memory through
	// this field and nowhere else (A-12: injection over environment
	// scraping); every declared extern must be present. Supplied
	// names the image never declares are ignored — the door in the
	// package comment.
	Externs map[string]string

	// Grace is the budget [Main] grants running services between the
	// first signal and the hard cancel; zero means 10 seconds. [Run]
	// has no use for it: graceful shutdown belongs to the process
	// skin.
	Grace time.Duration

	// Log receives the audit lines; nil means os.Stderr. Stdout stays
	// the services' own (A-12's stream discipline).
	Log io.Writer
}

// log resolves the audit sink.
func (cfg Config) log() io.Writer {
	if cfg.Log == nil {
		return os.Stderr
	}
	return cfg.Log
}

// grace resolves the shutdown budget.
func (cfg Config) grace() time.Duration {
	if cfg.Grace == 0 {
		return 10 * time.Second
	}
	return cfg.Grace
}

// A configError marks a fault in the hosting's inputs — image, plan,
// extern coverage, a binding no flag accepts — as opposed to a wet
// failure of something that ran. The split is A-12's restartable
// taxonomy: a config fault reproduces until the inputs change, so
// [Main] maps it to exit 2, the do-not-restart code, and wet failures
// to exit 1.
type configError struct{ err error }

func (e *configError) Error() string { return e.err.Error() }
func (e *configError) Unwrap() error { return e.err }

// configf builds one formatted config fault.
func configf(format string, args ...any) error {
	return &configError{fmt.Errorf(format, args...)}
}

// Run enacts cfg.Image in this process: load the plan, gate the
// externs, PROVISION every step in plan order, DEPLOY every service
// onto one [application.Runtime], and wait until every service has
// returned. All services completing cleanly returns nil — Run imposes
// no long-runningness; a solution of finite work is a solution.
//
// ctx bounds the whole run: drivers attach under it and every
// service's context derives from it, so cancelling ctx is the caller's
// hard stop (Run then returns the cancellation the services report).
// Run installs no signal handling and grants no grace; that skin is
// [Main]'s.
func Run(ctx context.Context, cfg Config) error {
	rt, err := start(ctx, cfg)
	if err != nil {
		return err
	}
	return rt.Wait()
}

// start carries the wet half up to serving: it returns once every
// service is running (or nothing is left to run). The runtime is the
// caller's to wait on and wind down; Run waits verbatim, Main wraps
// the wait in signals and the grace budget.
func start(ctx context.Context, cfg Config) (*application.Runtime, error) {
	plan, err := enact.Load(cfg.Image, cfg.Catalogue)
	if err != nil {
		return nil, &configError{err}
	}

	s := &session{
		cfg:     cfg,
		log:     cfg.log(),
		names:   make(map[string]string, len(cfg.Image.Catalogue)),
		symbols: make(map[string]image.SymbolDef, len(cfg.Image.Symbols)),
		outputs: make(map[string]*solution.OutputWriter, len(plan.Provisions)),
	}
	for _, pkg := range cfg.Image.Catalogue {
		s.names[pkg.Path] = pkg.Name
	}
	for _, def := range cfg.Image.Symbols {
		s.symbols[def.Name] = def
	}

	s.printf("enact %s generation %d: %d provision step(s), %d deploy step(s)",
		plan.Solution, plan.Generation, len(plan.Provisions), len(plan.Deploys))
	if err := s.gateExterns(plan.Externs); err != nil {
		return nil, err
	}
	for _, step := range plan.Provisions {
		if err := s.provision(ctx, step); err != nil {
			return nil, err
		}
	}
	services, err := s.deployments(plan.Deploys)
	if err != nil {
		return nil, err
	}

	// Every service is bound before the first one starts, so a config
	// fault in the last record cannot leave the first ones running.
	rt := application.RuntimeWithContext(ctx)
	for _, svc := range services {
		rt.Run(svc)
	}
	s.printf("running %d instance(s)", len(services))
	return rt, nil
}

// A session is one wet enactment in progress: the resolved audit sink,
// the image's lookup tables, and the outputs the PROVISION phase has
// stored so far.
type session struct {
	cfg Config
	log io.Writer

	// names maps the pinned catalogue's package paths to package
	// names for rendering element refs, the same spelling the plan's
	// own faults use.
	names map[string]string

	// symbols indexes the image's symbol table; bindings resolve var
	// values out of it and extern values through it into cfg.Externs.
	symbols map[string]image.SymbolDef

	// outputs holds each provisioned instance's written outputs,
	// keyed by instance name — the store later bindings render from.
	outputs map[string]*solution.OutputWriter
}

// printf writes one audit line.
func (s *session) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(s.log, format+"\n", args...)
}

// display renders an element ref as package name dot element name,
// the spelling faults and audit lines share with the planner.
func (s *session) display(ref image.Ref) string {
	if name, ok := s.names[ref.Package]; ok {
		return name + "." + ref.Name
	}
	return strconv.Quote(ref.Package) + "." + ref.Name
}

// gateExterns is the site gate, the wet half's first act (the plan is
// deliberately site-agnostic): every extern the image declares must
// have a caller-supplied value before any driver runs. Misses report
// all at once, one line each, so the operator fixes the site in one
// round trip; the bound ones audit here — value redacted when the
// symbol type is sensitive — because this is the one moment externs
// resolve.
func (s *session) gateExterns(externs []enact.Extern) error {
	var faults []error
	for _, ext := range externs {
		value, ok := s.cfg.Externs[ext.Name]
		if !ok {
			faults = append(faults, fmt.Errorf("unbound extern %s (%s): pass -extern %s=<value>",
				ext.Name, s.display(ext.Type), ext.Name))
			continue
		}
		s.printf("audit: extern %s = %s (site, %s)", ext.Name, redact(value, ext.Sensitive), s.display(ext.Type))
	}
	if faults != nil {
		return &configError{errors.Join(faults...)}
	}
	return nil
}

// provision runs one PROVISION step: a fresh provisioner from the
// type's factory, the record's resolved bindings wet-parsed into the
// provisioner's own flags, the driver attached against the type's
// declared outputs, completeness enforced, and the outputs stored for
// the bindings downstream. Driver failures are wet failures; nothing
// before the Attach call is.
func (s *session) provision(ctx context.Context, step enact.Step) error {
	name, elem := step.Record.Name, s.display(step.Record.Element)
	s.printf("provision %s (%s %s)", name, step.Record.Kind, elem)
	prov := step.Provision.Make()
	if err := s.bind(prov.Flags(), step.Record); err != nil {
		return err
	}

	w := solution.NewOutputWriter(step.Provision.Outputs)
	if err := prov.Attach(ctx, w); err != nil {
		return fmt.Errorf("provision %s: %w", name, err)
	}
	var missing []error
	for _, out := range w.Missing() {
		missing = append(missing, fmt.Errorf("provisioner for %s did not write output %s", name, out))
	}
	if missing != nil {
		return errors.Join(missing...)
	}
	s.outputs[name] = w

	for _, out := range step.Provision.Outputs {
		rendered, err := w.Render(out.Name)
		if err != nil {
			return fmt.Errorf("provision %s: output %s: %v", name, out.Name, err)
		}
		s.printf("audit: output %s.%s = %s (%s)", name, out.Name, redact(rendered, out.Sensitive), elem)
	}
	return nil
}

// deployments binds every deploy step to a fresh service, in image
// order. Nothing starts here: all services bind before any runs, so
// the phase falls over as configuration, never half-deployed.
func (s *session) deployments(steps []enact.Step) ([]named, error) {
	services := make([]named, 0, len(steps))
	for _, step := range steps {
		name := step.Record.Name
		s.printf("deploy %s (%s)", name, s.display(step.Record.Element))
		svc := step.Component.Make()
		if err := s.bind(svc.Flags(), step.Record); err != nil {
			return nil, err
		}
		services = append(services, named{name: name, Runner: svc})
	}
	return services, nil
}

// bind wet-parses one record's params into an instance's own flags:
// the surface Make just declared is copied into a loaderflags set
// sharing the flag.Values — so defaults, required marks, and the
// instance's closures all see the bound values — and one source, the
// record's resolved bindings, populates it through [loaderflags.Parse].
// Each slot's own flag.Value.Set is the wet validator, the same code
// that judged the value at compile (A-14); a Set that refuses here is
// image/binary drift, a config fault. After the parse the whole
// effective surface audits, one line per flag, redacted where the
// image tainted the binding.
func (s *session) bind(fs *flag.FlagSet, rec image.Record) error {
	ps := loaderflags.NewFlagSet(rec.Name)
	if fs != nil {
		fs.VisitAll(func(f *flag.Flag) { ps.Var(f.Value, f.Name, f.Usage) })
	}

	src := &recordSource{
		values:  make(map[string]string, len(rec.Params)),
		details: make(map[string]string, len(rec.Params)),
		applied: make(map[string]string, len(rec.Params)),
	}
	sensitive := make(map[string]bool, len(rec.Params))
	for _, b := range rec.Params {
		text, detail, err := s.resolve(b)
		if err != nil {
			return fmt.Errorf("%s: %w", rec.Name, err)
		}
		src.values[b.Key] = text
		src.details[b.Key] = detail
		sensitive[b.Key] = b.Sensitive
	}
	if err := loaderflags.Parse(ps, src); err != nil {
		return &configError{fmt.Errorf("%s: %v", rec.Name, err)}
	}

	ps.VisitAll(func(f *flag.Flag) {
		detail, applied := src.applied[f.Name]
		// The bound flags audit the string their Set was fed rather
		// than their own rendering: a validate-only flag.Value (a
		// flag.Func slot) renders nothing back, and an audit line
		// proving that a value arrived must show the value (A-10).
		value := src.values[f.Name]
		if !applied {
			detail, value = "catalogue default", f.Value.String()
		}
		s.printf("audit: %s.%s = %s (%s)", rec.Name, f.Name, redact(value, sensitive[f.Name]), detail)
	})
	return nil
}

// resolve renders one binding to the flag-ready string and its audit
// provenance. Literals render by kind — the exact spellings the
// compiler validated; symbol references resolve through the image's
// symbol table to a var's pinned value or the caller's extern value;
// output references render from the store the PROVISION phase filled,
// which plan order guarantees ran first. Every failure here is a
// config fault: nothing has run, the inputs are short.
func (s *session) resolve(b image.Binding) (text, detail string, err error) {
	switch {
	case b.Ref == nil:
		if b.Value == nil {
			return "", "", configf("binding %s carries neither value nor reference", b.Key)
		}
		if b.Value.Kind == image.KindToken {
			// Tokens are the deployment/extensions compartments'
			// vocabulary; a params token means a hand-built image put
			// one where only resolvable values belong.
			return "", "", configf("opaque token %q reached wet binding for %s", b.Value.Tok, b.Key)
		}
		text, err := renderValue(b.Value)
		if err != nil {
			return "", "", configf("binding %s: %v", b.Key, err)
		}
		return text, fmt.Sprintf("literal, %s", b.Source), nil

	case b.Ref.Output != "":
		w, ok := s.outputs[b.Ref.Symbol]
		if !ok {
			return "", "", configf("binding %s references output %s.%s before instance %s provisioned",
				b.Key, b.Ref.Symbol, b.Ref.Output, b.Ref.Symbol)
		}
		text, err := w.Render(b.Ref.Output)
		if err != nil {
			return "", "", configf("binding %s: %v", b.Key, err)
		}
		return text, fmt.Sprintf("output %s.%s, %s", b.Ref.Symbol, b.Ref.Output, b.Source), nil

	default:
		def, ok := s.symbols[b.Ref.Symbol]
		if !ok {
			return "", "", configf("binding %s references undeclared symbol %s", b.Key, b.Ref.Symbol)
		}
		switch def.Class {
		case image.ClassExtern:
			value, ok := s.cfg.Externs[def.Name]
			if !ok {
				// The gate runs first, so only a symbol table the
				// planner never saw gets here; refuse all the same.
				return "", "", configf("binding %s references unbound extern %s", b.Key, def.Name)
			}
			return value, fmt.Sprintf("extern %s, %s", def.Name, b.Source), nil
		case image.ClassVar:
			if def.Value == nil {
				return "", "", configf("binding %s references var %s, which pins no value", b.Key, def.Name)
			}
			text, err := renderValue(def.Value)
			if err != nil {
				return "", "", configf("binding %s: var %s: %v", b.Key, def.Name, err)
			}
			return text, fmt.Sprintf("var %s, %s", def.Name, b.Source), nil
		}
		return "", "", configf("binding %s references symbol %s of unknown class %q", b.Key, def.Name, def.Class)
	}
}

// renderValue renders a literal image value to the string a wet
// binding hands flag.Value.Set — the same spellings the compiler's
// literalValue validated dry and OutputWriter.Render emits for
// outputs, so a slot meets one rendering wherever a value comes from.
func renderValue(v *image.Value) (string, error) {
	switch v.Kind {
	case image.KindString:
		return v.Str, nil
	case image.KindInt:
		return strconv.FormatInt(v.Int, 10), nil
	case image.KindBool:
		return strconv.FormatBool(v.Bool), nil
	case image.KindDuration:
		return v.Dur.String(), nil
	}
	return "", fmt.Errorf("unknown value kind %q", v.Kind)
}

// redact quotes a value for the audit, or hides it when the taint says
// so (A-10): the audit proves a value resolved without exposing it.
func redact(value string, sensitive bool) string {
	if sensitive {
		return "<redacted>"
	}
	return strconv.Quote(value)
}

// A recordSource is the one loaderflags source of a hosted instance:
// the record's bindings, already resolved to flag-ready strings. It
// honors the earlier-source-wins contract (cheap, with no source
// before it) and keeps what it applied for the audit.
type recordSource struct {
	values  map[string]string
	details map[string]string
	applied map[string]string
}

func (src *recordSource) ParseParameters(fs *flag.FlagSet) error {
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if err != nil {
			return
		}
		v, ok := src.values[f.Name]
		if !ok || loaderflags.IsSet(fs, f.Name) {
			return
		}
		if e := fs.Set(f.Name, v); e != nil {
			err = fmt.Errorf("flag %s: %v", f.Name, e)
			return
		}
		src.applied[f.Name] = src.details[f.Name]
	})
	return err
}

func (src *recordSource) String() string { return "image record" }

// A named runner carries its instance name into logs and process
// listings and hands the shutdown capability through to the service
// when the service has one — [application.Runtime.Shutdown] asks the
// tracked runner, so the wrapper must not hide the interface.
type named struct {
	name string
	application.Runner
}

func (n named) String() string { return n.name }

func (n named) Shutdown(ctx context.Context) error {
	if sd, ok := n.Runner.(application.Shutdowner); ok {
		return sd.Shutdown(ctx)
	}
	return nil
}
