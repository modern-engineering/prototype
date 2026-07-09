// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package translator provides the raw-to-canonical observation stage.
// Four input streams produce five canonical event streams for the digital twin.
package translator

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/parameter"
)

// Component translates raw controller observations into canonical events.
var Component = &application.Descriptor{
	Name: "observationTranslator",
	Doc:  "observationTranslator consumes raw controller observations and publishes canonical asset, identity, session, route, and link events for the digital twin.",
	Make: application.MakeFunc(makeTranslator),
}

func makeTranslator() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("observationTranslator", flag.ContinueOnError)

	// The fetch limit bounds concurrent translation work.
	fetchLimit := fs.Int("fetchLimit", 0, "max messages fetched concurrently for dependency analysis")

	// Message-bus wiring: the four interests and five aspects of the
	// descriptor, each bound to a topic by the solution.
	consumesAssetObservation := fs.String("consumesAssetObservation", "", "topic of raw asset observations")
	fs.String("consumesLink", "", "topic of raw connectivity reports")
	fs.String("consumesSession", "", "topic of raw session reports")
	fs.String("consumesGateway", "", "topic of raw gateway reports")
	producesAssetObserved := fs.String("producesAssetObserved", "", "topic for canonical asset observations")
	fs.String("producesLinkChanged", "", "topic for canonical connectivity changes")
	fs.String("producesSessionEstablished", "", "topic for canonical network sessions")
	fs.String("producesRouteChanged", "", "topic for canonical route changes")
	fs.String("producesIdentityObserved", "", "topic for canonical identity observations")

	// Kafka connection and batching settings are explicit runtime inputs.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")
	fs.String("kafkaTopicOptions", "", "gocloud topic batcher options as ;-separated key=value pairs (shared env KAFKA_TOPIC_OPTIONS)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Keep a conservative default while sites move the limit into explicit configuration.
		limit := *fetchLimit
		if limit <= 0 {
			limit = 1
		}
		// Mirrors translator.TranslatorOptions.Validate.
		if limit <= 0 {
			return fmt.Errorf("options: %w", errors.New("fetchlimit must be positive"))
		}

		// The representative loop retains the four-way input fan-out.
		log.Printf("observation translator: %d pipelines, fetch limit %d: %q -> %q, ...",
			4, limit, *consumesAssetObservation, *producesAssetObserved)
		<-ctx.Done()
		return ctx.Err()
	}), fs
}
