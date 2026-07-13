// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package substrate defines the site-owned services and provisioned accesses
// used by the managed-network security scenario. Output schemes cover the
// connection shapes required by their consumers; for example, Redis exposes
// both host-plus-port fields and a combined address.
package substrate

import (
	"context"
	"errors"
	"flag"

	"github.com/modern-engineering/prototype/solution"
)

// KafkaCluster names the Kafka cluster every bridge component talks
// through. In cloud and dev environments it is the per-namespace
// Strimzi cluster the deployment bundle owns
// (site-kafka-bootstrap:9092); on-prem sites bind an
// external edge broker instead (edge-gateway:23242).
var KafkaCluster = &solution.SymbolType{
	Doc: "a Strimzi-managed Kafka cluster, named by the site (site-kafka-bootstrap in cloud, an edge broker on-prem)",
}

// RedisServer names the shared Redis instance (bitnami chart,
// security-redis:6379) that holds CONTROLLER API tokens and detector
// pairing state, in separate logical databases.
var RedisServer = &solution.SymbolType{
	Doc: "the shared Redis instance of the deployment (security-redis:6379 in the deployment bundle)",
}

// Neo4jServer names the Neo4j server backing the network digital twin
// (bolt://twin-neo4j:7687 in the deployment bundle).
var Neo4jServer = &solution.SymbolType{
	Doc: "the Neo4j server backing the digital twin of a target network (bolt://twin-neo4j:7687 in the deployment bundle)",
}

// PostgresServer names the PostgreSQL server the alerts journal writes
// to and PostgREST reads from (postgresql:5432 in the deployment bundle).
var PostgresServer = &solution.SymbolType{
	Doc: "the PostgreSQL server behind the alerts journal and the PostgREST facade (postgresql:5432)",
}

// Secret is an operator-supplied credential. The scenario deployment delivers
// these as Kubernetes Secret keys mapped into per-component env vars
// (e.g. controller-credentials/controller.tenant.id); every binding wired from
// a Secret-typed extern is tainted in the image.
var Secret = &solution.SymbolType{
	Doc:       "an operator-supplied credential, delivered as a Kubernetes Secret key in the scenario deployment",
	Sensitive: true,
}

// Endpoint is a site-bound network coordinate outside the platform's
// guarantees, such as the customer's SIEM syslog server.
var Endpoint = &solution.SymbolType{
	Doc: "a site-bound network coordinate such as the SIEM syslog server address",
}

// Kafka provisions access to a managed cluster or attaches an externally
// administered broker. Topic lifecycle remains outside this provision type.
var Kafka = &solution.ProvisionType{
	Doc:  "access to a Kafka cluster: sliced from the per-environment Strimzi operator or attached to an external broker",
	Make: makeKafka,
	Outputs: []solution.Output{
		{Name: "brokers", Doc: "comma-separated bootstrap broker addresses (for example site-kafka-bootstrap:9092)"},
	},
	Kinds: solution.Slice | solution.Attach,
}

// A kafkaProvisioner validates configuration without changing infrastructure.
// Adopter-owned drivers implement reconciliation behind the same seam.
type kafkaProvisioner struct {
	flags   *flag.FlagSet
	cluster string
}

func makeKafka() solution.Provisioner {
	p := &kafkaProvisioner{flags: flag.NewFlagSet("kafka", flag.ContinueOnError)}
	p.flags.StringVar(&p.cluster, "cluster", "", "substrate cluster to hold or verify the access on")
	return p
}

func (p *kafkaProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *kafkaProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("Kafka provisioning is not implemented; the type compiles solutions but cannot enact them")
}

// Redis provisions an account on one logical database of a shared server.
// Separate provisions make controller-token and detector-state access explicit.
var Redis = &solution.ProvisionType{
	Doc:  "an account on one logical database of the shared Redis server",
	Make: makeRedis,
	Outputs: []solution.Output{
		{Name: "address", Doc: "host:port coordinate of the server, for consumers taking one address flag"},
		{Name: "host", Doc: "host part of the coordinate, for consumers taking host and port separately"},
		{Name: "port", Doc: "port part of the coordinate, for consumers taking host and port separately"},
		{Name: "db", Doc: "the provisioned logical database number, echoed for consumer flags"},
		{Name: "username", Doc: "account username; a Kubernetes Secret key in the scenario deployment", Sensitive: true},
		{Name: "password", Doc: "account password; a Kubernetes Secret key in the scenario deployment", Sensitive: true},
	},
	Kinds: solution.Slice | solution.Attach,
}

// A redisProvisioner is one dry Redis instance; see kafkaProvisioner
// for why Attach refuses.
type redisProvisioner struct {
	flags  *flag.FlagSet
	server string
	db     int
}

func makeRedis() solution.Provisioner {
	p := &redisProvisioner{flags: flag.NewFlagSet("redis", flag.ContinueOnError)}
	p.flags.StringVar(&p.server, "server", "", "Redis server to carve the database access from")
	p.flags.IntVar(&p.db, "db", 0, "logical database number the access is scoped to")
	return p
}

func (p *redisProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *redisProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("scenario Redis provisioning is not implemented; the type compiles solutions but cannot enact them")
}

// Neo4j provisions a tainted account for the target-network digital twin.
var Neo4j = &solution.ProvisionType{
	Doc:  "access to a Neo4j server for the digital twin of a target network's graph database",
	Make: makeNeo4j,
	Outputs: []solution.Output{
		{Name: "uri", Doc: "bolt URI of the server (for example bolt://twin-neo4j:7687)"},
		{Name: "username", Doc: "basic-auth username; stored in a ConfigMap in the deployment manifest, tainted here", Sensitive: true},
		{Name: "password", Doc: "basic-auth password; stored in a ConfigMap in the deployment manifest, tainted here", Sensitive: true},
	},
	Kinds: solution.Slice | solution.Attach,
}

// A neo4jProvisioner is one dry Neo4j instance; see kafkaProvisioner
// for why Attach refuses.
type neo4jProvisioner struct {
	flags  *flag.FlagSet
	server string
}

func makeNeo4j() solution.Provisioner {
	p := &neo4jProvisioner{flags: flag.NewFlagSet("neo4j", flag.ContinueOnError)}
	p.flags.StringVar(&p.server, "server", "", "Neo4j server to hold or verify the access on")
	return p
}

func (p *neo4jProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *neo4jProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("scenario Neo4j provisioning is not implemented; the type compiles solutions but cannot enact them")
}

// Postgres provisions a tainted account for the alert journal.
var Postgres = &solution.ProvisionType{
	Doc:  "access to a named database on the PostgreSQL server",
	Make: makePostgres,
	Outputs: []solution.Output{
		{Name: "address", Doc: "host:port coordinate of the server (for example postgresql:5432)"},
		{Name: "database", Doc: "the provisioned database name, echoed for consumer flags"},
		{Name: "username", Doc: "account username; postgresql-secret/postgresqlUsername in the scenario deployment", Sensitive: true},
		{Name: "password", Doc: "account password; postgresql-secret/postgresqlPassword in the scenario deployment", Sensitive: true},
	},
	Kinds: solution.Slice | solution.Attach,
}

// A postgresProvisioner is one dry Postgres instance; see
// kafkaProvisioner for why Attach refuses.
type postgresProvisioner struct {
	flags            *flag.FlagSet
	server, database string
}

func makePostgres() solution.Provisioner {
	p := &postgresProvisioner{flags: flag.NewFlagSet("postgres", flag.ContinueOnError)}
	p.flags.StringVar(&p.server, "server", "", "PostgreSQL server to hold or verify the access on")
	p.flags.StringVar(&p.database, "database", "", "database name the access is scoped to (for example \"postgres\")")
	return p
}

func (p *postgresProvisioner) Flags() *flag.FlagSet { return p.flags }

func (p *postgresProvisioner) Attach(context.Context, *solution.OutputWriter) error {
	return errors.New("scenario Postgres provisioning is not implemented; the type compiles solutions but cannot enact them")
}
