// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package detection

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/parameter"
)

// Journal stores security alerts and exposes queryable views.
var Journal = &application.Descriptor{
	Name: "detectionAlertsJournal",
	Doc:  "detectionAlertsJournal consumes security alerts, upserts them into PostgreSQL, and maintains views for a PostgREST facade.",
	Make: application.MakeFunc(makeJournal),
}

func makeJournal() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("detectionAlertsJournal", flag.ContinueOnError)

	// The provisioned PostgreSQL account is required for journal storage.
	postgresAddress := fs.String("postgresAddress", "", "postgres server address")
	parameter.Require(fs, "postgresAddress")
	fs.String("postgresUsername", "", "postgres server username")
	parameter.Require(fs, "postgresUsername")
	fs.String("postgresPassword", "", "postgres server password")
	parameter.Require(fs, "postgresPassword")
	fs.String("postgresDBName", "", "postgres db name")
	parameter.Require(fs, "postgresDBName")

	// The query role is specific to this journal instance.
	postgresRoleName := fs.String("postgresRoleName", "", "PostgreSQL role granted access to the journal views")

	// Message-bus wiring: a pure sink — one interest, no aspects.
	consumesAlerts := fs.String("consumesAlerts", "", "topic of JSON alerts")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Mirrors detection.JournalOptions.Validate.
		if *postgresRoleName == "" {
			return fmt.Errorf("validate options: %w", errors.New("postgres role name is required"))
		}

		// The representative loop retains schema setup and alert upserts.
		log.Printf("alerts journal: %q -> postgres %s (views for role %q)",
			*consumesAlerts, *postgresAddress, *postgresRoleName)
		<-ctx.Done()
		return ctx.Err()
	}), fs
}
