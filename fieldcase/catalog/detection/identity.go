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

// IdentityAnomalyDetector correlates network identities with endpoints.
var IdentityAnomalyDetector = &application.Descriptor{
	Name: "identityAnomalyDetector",
	Doc:  "identityAnomalyDetector consumes digital-twin changes, tracks identity-to-endpoint associations in Redis, and publishes an alert when a known identity appears from an unexpected endpoint.",
	Make: application.MakeFunc(makeIdentityAnomalyDetector),
}

func makeIdentityAnomalyDetector() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("identityAnomalyDetector", flag.ContinueOnError)

	// The provisioned Redis account stores identity-to-endpoint associations.
	redisAddress := fs.String("redisAddress", "localhost:6379", "redis host address")
	parameter.Require(fs, "redisAddress")
	fs.Int("redisDBNumber", 0, "redis db number")
	parameter.Require(fs, "redisDBNumber")
	fs.String("redisUsername", "", "redis user")
	parameter.Require(fs, "redisUsername")
	fs.String("redisPassword", "", "redis password")
	parameter.Require(fs, "redisPassword")

	// Policy and concurrency limits belong to this detector instance.
	policyRuleName := fs.String("policyRuleName", "", "policy rule naming the generated alerts")
	handlersLimit := fs.Int("handlersLimit", 0, "goroutines handling assemblies concurrently; negative means unlimited, zero is not allowed")

	// Message-bus wiring: one interest, one aspect. The alerts topic is
	// a fan-in bus shared with the journal, the syslog exporter, and
	// every other detector.
	consumesGraphChanged := fs.String("consumesGraphChanged", "", "topic of gob-encoded graph-change digests")
	producesAlerts := fs.String("producesAlerts", "", "topic for JSON alerts")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Mirrors detection.IdentityPolicyOptions.Validate and
		// detection.PolicyRule.Validate.
		var err error
		if *policyRuleName == "" {
			err = errors.Join(err, errors.New("policy rule name is required"))
		}
		if *handlersLimit == 0 {
			err = errors.Join(err, errors.New("handlers limit field must be set to a value != 0"))
		}
		if err != nil {
			return fmt.Errorf("validate options: %w", err)
		}

		// The representative loop retains bootstrap probing and shutdown persistence.
		log.Printf("identity anomaly detector: policy %q, state in redis %s: %q -> %q",
			*policyRuleName, *redisAddress, *consumesGraphChanged, *producesAlerts)
		<-ctx.Done()
		log.Printf("identity anomaly detector: would persist pairing state to redis %s", *redisAddress)
		return ctx.Err()
	}), fs
}
