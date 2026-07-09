// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package host runs a compiled solution image as a single process.
// A generated program supplies the image records and catalogue imports;
// this package loads site bindings, resolves provision outputs, configures
// instances, redacts sensitive audit values, reports health, and coordinates
// ordered shutdown. Regeneration changes solution data without re-emitting
// the host engine.
//
// # Runtime names
//
// Instance identifiers determine flag and environment namespaces.
// Flags use -<instance>.<flag>; environment variables uppercase the same
// spelling and replace punctuation with underscores. Consumer groups use
// <deployment-id>/<instance>, keeping runtime identity stable across restarts.
package host

import (
	"sort"
	"strings"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/fieldcase/host/driver"
	"github.com/modern-engineering/prototype/fieldcase/host/site"
)

// The symbol classes of [Symbol], mirroring the image's.
const (
	ClassVar    = "var"
	ClassExtern = "extern"
)

// A Config is one solution's baked desired state: what the generator
// reads out of the image and writes into the generated main. Slices
// keep the image's order (symbols sorted by name, records in
// unit-then-statement order) so every report is deterministic.
type Config struct {
	// Solution is the solution name; it prefixes logs and defaults the
	// deployment identity.
	Solution string

	// Symbols is the image's symbol table.
	Symbols []Symbol

	// Provisions are the image's provision records, each wired to the
	// driver that reconciles it.
	Provisions []Provision

	// Instances are the image's deploy records, each wired to its
	// catalogue descriptor.
	Instances []Instance
}

// A Symbol is one row of the image's symbol table.
type Symbol struct {
	// Name is the solution-wide symbol name.
	Name string

	// Class is [ClassVar] or [ClassExtern].
	Class string

	// Type is an extern's symbol-type display name (for example
	// "substrate.Secret"), used in gate and audit messages; empty for
	// vars.
	Type string

	// Sensitive marks externs of sensitive symbol types; their values
	// are redacted in the audit.
	Sensitive bool

	// Default is a var's compile-bound default in canonical string
	// form; empty for externs.
	Default string
}

// The binding kinds of [Binding]: exactly one is meaningful per
// binding, matching the image's literal-or-reference payload.
const (
	BindLiteral = "literal"
	BindSymbol  = "symbol"
	BindOutput  = "output"
)

// A Binding is one bound parameter of a record, in the canonical
// string form the flag surface consumes.
type Binding struct {
	// Key is the parameter (flag) name.
	Key string

	// Kind selects the payload: [BindLiteral], [BindSymbol], or
	// [BindOutput].
	Kind string

	// Value is the literal's canonical string ([BindLiteral]) or the
	// referenced symbol's name ([BindSymbol]).
	Value string

	// Instance and Output name a provision instance's output
	// ([BindOutput]).
	Instance string
	Output   string

	// Sensitive carries the image's taint: the resolved value must not
	// be logged or exposed.
	Sensitive bool

	// Source is the image's binding provenance (instance,
	// default-deploy, ...), carried into the audit.
	Source string
}

// A Provision is one provision record and its reconciliation hook.
type Provision struct {
	// Name is the provision instance name.
	Name string

	// Type is the provision type's display name, e.g. "substrate.Redis".
	Type string

	// Kind is "slice" or "attach".
	Kind string

	// Params are the record's bound parameters.
	Params []Binding

	// Outputs is the type's output scheme.
	Outputs []Output

	// Driver reconciles the record at startup. The generator wires the
	// development drivers; adopter drivers may reconcile real infrastructure.
	Driver driver.Driver
}

// An Output is one output of a provision type's scheme.
type Output struct {
	Name      string
	Sensitive bool
}

// An Instance is one deploy record wired to its catalogue descriptor.
type Instance struct {
	// Name is the instance name: the reconciliation key, the flag/env
	// namespace, and the consumer-group leaf.
	Name string

	// App is the catalogue descriptor; [Main] constructs the service
	// via App.Make (the dry-instantiation contract).
	App *application.Descriptor

	// Params are the record's bound parameters.
	Params []Binding
}

// EnvName translates a flag or instance name to its environment-variable
// alias: ASCII-uppercase with '-', '.', and '/' replaced by '_' (the
// ff translation the host uses).
func EnvName(name string) string {
	return strings.ToUpper(strings.Map(func(r rune) rune {
		switch r {
		case '-', '.', '/':
			return '_'
		}
		return r
	}, name))
}

// Needs is the closure of what one selection of instances consumes:
// the symbols their bindings reference, the provisions those bindings
// and the provisions' own parameters reach, and per provision the
// outputs actually consumed.
type Needs struct {
	// Externs and Vars are the reachable symbol names by class.
	Externs map[string]bool
	Vars    map[string]bool

	// Provisions are the reachable provision instance names.
	Provisions map[string]bool

	// Outputs lists, per reachable provision, the consumed outputs.
	Outputs map[string]map[string]bool
}

// Needs computes the reachability closure for the named instances.
// Provision parameters may themselves reference symbols and other
// provisions' outputs; the walk runs to a fixpoint (the compiler
// guarantees the reference graph is a DAG).
func (c *Config) Needs(enabled []string) Needs {
	n := Needs{
		Externs:    make(map[string]bool),
		Vars:       make(map[string]bool),
		Provisions: make(map[string]bool),
		Outputs:    make(map[string]map[string]bool),
	}
	class := make(map[string]string, len(c.Symbols))
	for _, s := range c.Symbols {
		class[s.Name] = s.Class
	}
	provisions := make(map[string]*Provision, len(c.Provisions))
	for i := range c.Provisions {
		provisions[c.Provisions[i].Name] = &c.Provisions[i]
	}

	visit := func(bindings []Binding) (newProvisions []string) {
		for _, b := range bindings {
			switch b.Kind {
			case BindSymbol:
				switch class[b.Value] {
				case ClassExtern:
					n.Externs[b.Value] = true
				case ClassVar:
					n.Vars[b.Value] = true
				}
			case BindOutput:
				if !n.Provisions[b.Instance] {
					n.Provisions[b.Instance] = true
					newProvisions = append(newProvisions, b.Instance)
				}
				out := n.Outputs[b.Instance]
				if out == nil {
					out = make(map[string]bool)
					n.Outputs[b.Instance] = out
				}
				out[b.Output] = true
			}
		}
		return newProvisions
	}

	var frontier []string
	for _, inst := range c.Instances {
		for _, name := range enabled {
			if inst.Name == name {
				frontier = append(frontier, visit(inst.Params)...)
			}
		}
	}
	for len(frontier) > 0 {
		name := frontier[0]
		frontier = frontier[1:]
		if p := provisions[name]; p != nil {
			frontier = append(frontier, visit(p.Params)...)
		}
	}
	return n
}

// CheckSite validates a site's keys against the config: every extern
// line must name an extern symbol, every var line a var symbol, and
// every driver line a provision instance and an output of its scheme.
// The returned fault lines are sorted; an empty result is a valid
// site. Both the generator and the host run this check, so a typo
// fails at whichever stage first sees the file.
func (c *Config) CheckSite(st *site.Site) []string {
	symbols := make(map[string]string, len(c.Symbols))
	for _, s := range c.Symbols {
		symbols[s.Name] = s.Class
	}
	var faults []string
	for _, name := range st.Externs() {
		switch symbols[name] {
		case ClassExtern:
		case ClassVar:
			faults = append(faults, "extern."+name+": "+name+" is a var; rebind it as var."+name)
		default:
			faults = append(faults, "extern."+name+": no such extern in the image")
		}
	}
	for _, name := range st.Vars() {
		switch symbols[name] {
		case ClassVar:
		case ClassExtern:
			faults = append(faults, "var."+name+": "+name+" is an extern; bind it as extern."+name)
		default:
			faults = append(faults, "var."+name+": no such var in the image")
		}
	}
	provisions := make(map[string]*Provision, len(c.Provisions))
	for i := range c.Provisions {
		provisions[c.Provisions[i].Name] = &c.Provisions[i]
	}
	for _, name := range st.Drivers() {
		p, ok := provisions[name]
		if !ok {
			faults = append(faults, "driver."+name+".*: no such provision instance in the image")
			continue
		}
		scheme := make(map[string]bool, len(p.Outputs))
		for _, out := range p.Outputs {
			scheme[out.Name] = true
		}
		section := st.Driver(name)
		for _, out := range sortedKeys(section) {
			if !scheme[out] {
				faults = append(faults, "driver."+name+"."+out+": "+p.Type+" declares no such output")
			}
		}
	}
	return faults
}

// UnboundExterns returns the needed extern symbols the site does not
// bind, in symbol-table order: the required-extern gate's evidence.
func (c *Config) UnboundExterns(st *site.Site, needs Needs) []Symbol {
	var missing []Symbol
	for _, s := range c.Symbols {
		if s.Class != ClassExtern || !needs.Externs[s.Name] {
			continue
		}
		if _, ok := st.Extern(s.Name); !ok {
			missing = append(missing, s)
		}
	}
	return missing
}

// selectInstances applies the host's tri-state enablement
// semantics: any flag set true selects exactly the true ones; any set
// false (none true) selects all but the false ones; none set selects
// all. The result keeps Config order.
func (c *Config) selectInstances(enables map[string]*enableFlag) []string {
	anyTrue, anyFalse := false, false
	for _, e := range enables {
		if e.set && e.value {
			anyTrue = true
		}
		if e.set && !e.value {
			anyFalse = true
		}
	}
	var names []string
	for _, inst := range c.Instances {
		e := enables[inst.Name]
		switch {
		case anyTrue:
			if e != nil && e.set && e.value {
				names = append(names, inst.Name)
			}
		case anyFalse:
			if e == nil || !e.set || e.value {
				names = append(names, inst.Name)
			}
		default:
			names = append(names, inst.Name)
		}
	}
	return names
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
