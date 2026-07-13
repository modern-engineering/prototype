// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package substrate is the sample substrate catalogue: the classes of
// late-bound values a solution wires through extern symbols, and the
// provisioning types that carve access out of platform-guaranteed
// services. Where package ff shows the component kind of catalogue
// citizen, substrate shows the other two — exported
// *solution.SymbolType vars that extern declarations name, and
// exported *solution.ProvisionType vars whose instances slice or
// attach to shared substrate and emit reconcile-time outputs.
package substrate

import (
	"context"
	"errors"
	"flag"

	"github.com/modern-engineering/prototype/solution"
)

// Secret is an operator-supplied credential. It is sensitive: every
// binding wired from a Secret-typed extern is tainted in the image, so
// deployment backends know what they must not log or expose.
var Secret = &solution.SymbolType{
	Doc:       "an operator-supplied credential, bound by the site and redacted wherever it surfaces",
	Sensitive: true,
}

// Endpoint is a network coordinate — an address, URL, or subject — the
// deploying site routes traffic through.
var Endpoint = &solution.SymbolType{
	Doc: "a site-bound network coordinate such as an address or subject",
}

// NATSCluster names a NATS cluster of the guaranteed substrate. The
// site binds it at reification; slices carve their accounts out of the
// cluster it names.
var NATSCluster = &solution.SymbolType{
	Doc: "a substrate NATS cluster, guaranteed by the platform and named by the site",
}

// PostgresServer names a PostgreSQL server instance that exists
// outside the platform's guarantees — the legacy per-customer
// substrate attachments plug into without owning.
var PostgresServer = &solution.SymbolType{
	Doc: "a PostgreSQL server instance provided by the site, typically legacy per-customer substrate",
}

// NATS provisions an account on a substrate NATS cluster. As a slice
// it owns the account: the driver creates it, mutates it on definition
// changes, and — delete protection satisfied — destroys it. As an
// attachment it verifies an account someone else administers. Both
// kinds emit the same output scheme, so downstream wiring cannot tell
// them apart and migrating between them is a one-word change (A-11).
var NATS = &solution.ProvisionType{
	Doc:  "an account carved out of a substrate NATS cluster, or a verified attachment to one",
	Make: makeNATS,
	Outputs: []solution.Output{
		{Name: "config", Doc: "account configuration granting access to the provisioned account", Sensitive: true},
	},
	Kinds: solution.Slice | solution.Attach,
}

// A natsProvisioner is one dry NATS instance: the parameter surface
// the linker validates statements against and a host wet-binds before
// the driver runs.
type natsProvisioner struct {
	flags                 *flag.FlagSet
	cluster, adminAccount string
}

func makeNATS() solution.Provisioner {
	p := &natsProvisioner{flags: flag.NewFlagSet("nats", flag.ContinueOnError)}
	p.flags.StringVar(&p.cluster, "cluster", "", "substrate cluster to hold the account on")
	p.flags.StringVar(&p.adminAccount, "adminAccount", "", "administrative credential the driver provisions with")
	return p
}

func (p *natsProvisioner) Flags() *flag.FlagSet { return p.flags }

// Attach refuses: NATS is a compile-only citizen until the real
// account driver lands (its acceptance spec — the admin credential
// creates the solution's account if absent — is on the ledger), so
// solutions provisioning it build fine and fail only at enactment.
func (p *natsProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("real NATS provisioning is not implemented; the type compiles solutions but cannot enact them")
}

// Postgres attaches to a PostgreSQL server the site provides. The type
// registers only the attach kind: the legacy per-customer servers
// exist today and are administered elsewhere, so a solution plugs in
// verify-only and nothing is ever pruned. A slice kind is the
// migration path off that substrate, registered the day the platform
// guarantees a shared server.
var Postgres = &solution.ProvisionType{
	Doc:  "a verified attachment to a site-provided PostgreSQL server",
	Make: makePostgres,
	Outputs: []solution.Output{
		{Name: "dsn", Doc: "connection string for the attached server, credentials included", Sensitive: true},
	},
	Kinds: solution.Attach,
}

// A postgresProvisioner is one dry Postgres instance, the same shape
// as natsProvisioner over a one-slot surface.
type postgresProvisioner struct {
	flags  *flag.FlagSet
	server string
}

func makePostgres() solution.Provisioner {
	p := &postgresProvisioner{flags: flag.NewFlagSet("postgres", flag.ContinueOnError)}
	p.flags.StringVar(&p.server, "server", "", "server instance to verify and attach to")
	return p
}

func (p *postgresProvisioner) Flags() *flag.FlagSet { return p.flags }

// Attach refuses like NATS's: the legacy servers this type attaches
// to exist, but no driver verifies them yet, so the type compiles
// solutions and stops their enactment.
func (p *postgresProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("real Postgres provisioning is not implemented; the type compiles solutions but cannot enact them")
}
