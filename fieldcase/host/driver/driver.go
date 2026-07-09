// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package driver resolves provision records into runtime outputs.
// The development implementations read outputs from the site file and touch
// no infrastructure; adopter implementations may reconcile real services.
//
// The A-11 semantics stay visible even in dev mode: a slice logs its
// delete-protection posture (an owned partition is destroyed only with
// protection satisfied, and the dev driver never destroys anything),
// and an attachment logs that it verified existing substrate it will
// never prune.
package driver

import (
	"context"
	"fmt"
	"log"
	"sort"
)

// A Record describes one provision record the host asks a driver to
// reconcile: the identity from the image plus the site file's driver
// section for the instance.
type Record struct {
	// Name is the provision instance name (the reconciliation key).
	Name string

	// Type is the provision type's display name, e.g. "substrate.Redis".
	Type string

	// Kind is the record's provision kind: "slice" or "attach".
	Kind string

	// Site is the site file's driver.<name>.* section: the dev-mode
	// source of output values. Nil when the site carries none.
	Site map[string]string

	// Need lists the outputs this run consumes: every output of the
	// type's scheme referenced by an enabled instance or a reachable
	// provision. The driver must resolve at least these.
	Need []string
}

// A Driver provisions one record and returns its resolved outputs.
// Provision must resolve every output named in record.Need or fail;
// it may resolve more (the type's full scheme).
type Driver interface {
	Provision(ctx context.Context, record Record, params map[string]string) (outputs map[string]string, err error)
}

// The dev-mode drivers, one per substrate provision type of the
// anomaly catalogue. All four share the site-echo core; the named
// types keep the door visible: a Kafka driver would manage the
// cluster named by params["cluster"], a scenario Redis driver would carve
// an ACL account on the logical database params["db"], and so on.
type (
	// DevKafka stands in for a Kafka driver (scenario work: verify or own
	// the Strimzi cluster, emit its bootstrap address).
	DevKafka struct{}

	// DevRedis stands in for a Redis driver (scenario work: create an ACL
	// account scoped to one logical database, emit its coordinates and
	// credentials).
	DevRedis struct{}

	// DevNeo4j stands in for a Neo4j driver (scenario work: verify the
	// server, manage the twin database's account, emit the bolt URI).
	DevNeo4j struct{}

	// DevPostgres stands in for a PostgreSQL driver (scenario work: create
	// the database and an owning role, emit address and credentials).
	DevPostgres struct{}
)

func (DevKafka) Provision(ctx context.Context, rec Record, params map[string]string) (map[string]string, error) {
	return echo(rec, params)
}

func (DevRedis) Provision(ctx context.Context, rec Record, params map[string]string) (map[string]string, error) {
	return echo(rec, params)
}

func (DevNeo4j) Provision(ctx context.Context, rec Record, params map[string]string) (map[string]string, error) {
	return echo(rec, params)
}

func (DevPostgres) Provision(ctx context.Context, rec Record, params map[string]string) (map[string]string, error) {
	return echo(rec, params)
}

// echo is the shared dev-mode core: verify the site's driver section
// covers every needed output and return the configured values. The
// params — the record's bound parameters, already resolved by the host
// — are logged as the coordinates a scenario driver would act on.
func echo(rec Record, params map[string]string) (map[string]string, error) {
	var missing []string
	for _, name := range rec.Need {
		if _, ok := rec.Site[name]; !ok {
			missing = append(missing, fmt.Sprintf("driver.%s.%s", rec.Name, name))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("dev driver for %s (%s): site must configure the outputs this run consumes: %v", rec.Name, rec.Type, missing)
	}

	outputs := make(map[string]string, len(rec.Site))
	for name, value := range rec.Site {
		outputs[name] = value
	}

	// The A-11 lifecycle posture, visible in the logs even though the
	// dev driver touches nothing real.
	switch rec.Kind {
	case "slice":
		log.Printf("driver: slice %s (%s) on %s: dev stand-in owns the partition; deletion stays guarded by delete protection (this driver never destroys)",
			rec.Name, rec.Type, coordinates(params))
	case "attach":
		log.Printf("driver: attach %s (%s) on %s: dev stand-in verified existing substrate; attachments are never pruned",
			rec.Name, rec.Type, coordinates(params))
	default:
		return nil, fmt.Errorf("dev driver for %s (%s): unknown provision kind %q", rec.Name, rec.Type, rec.Kind)
	}
	return outputs, nil
}

// coordinates renders the record's bound parameters for the lifecycle
// log line, sorted for determinism.
func coordinates(params map[string]string) string {
	if len(params) == 0 {
		return "(no parameters)"
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := ""
	for i, k := range keys {
		if i > 0 {
			s += " "
		}
		s += fmt.Sprintf("%s=%q", k, params[k])
	}
	return s
}
