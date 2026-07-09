// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package digitaltwin provides the event compiler and graph writer that maintain
// a Neo4j-backed digital twin of the target IP network.
package digitaltwin

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/parameter"
)

// TranslatorComponent compiles canonical observations into graph mutations.
// Its descriptor name is also a compatibility-sensitive runtime identity used
// to derive the Kafka consumer group across deployments.
var TranslatorComponent = &application.Descriptor{
	Name: "digitaltwin",
	Doc:  "digitaltwin consumes canonical network observations and publishes graph mutation batches for the digital-twin graph.",
	Make: application.MakeFunc(makeTranslator),
}

func makeTranslator() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("digitaltwin", flag.ContinueOnError)

	// Queue inputs are optional; setting a region enables the SQS bridge and
	// requires both queue URLs. The fetch limit bounds concurrent analysis.
	region := fs.String("region", "", "AWS region for SQS queues; empty disables SQS")
	assetObservedURL := fs.String("assetObservedURL", "", "SQS queue URL for asset observations")
	assetsBehindGatewayStatusURL := fs.String("assetsBehindGatewayStatusURL", "", "SQS queue URL for assets behind a gateway status")
	fetchLimit := fs.Int("fetchLimit", 0, "max messages fetched concurrently for dependency analysis")

	// Message-bus wiring: one aspect, eleven interests. The six streams
	// past the first five are produced by components outside this flow.
	producesRemoteCompilation := fs.String("producesRemoteCompilation", "", "topic for graph mutation batches")
	consumesAssetObserved := fs.String("consumesAssetObserved", "", "topic of canonical asset observations")
	fs.String("consumesLinkChanged", "", "topic of canonical connectivity changes")
	fs.String("consumesSessionEstablished", "", "topic of canonical network sessions")
	fs.String("consumesRouteChanged", "", "topic of canonical route changes")
	fs.String("consumesIdentityObserved", "", "topic of canonical identity observations")
	fs.String("consumesUnmanagedAssetObserved", "", "topic of unmanaged asset observations")
	fs.String("consumesAssetsBehindGateway", "", "topic of assets behind a gateway statuses")
	fs.String("consumesAssetLocationData", "", "topic of asset location observations")
	fs.String("consumesAssetTelemetry", "", "topic of asset telemetry observations")
	fs.String("consumesNetworkDomain", "", "topic of network-domain changes")
	fs.String("consumesNetworkStates", "", "topic of datapath network states")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Keep a conservative default while sites move the limit into explicit configuration.
		limit := *fetchLimit
		if limit <= 0 {
			limit = 1
		}
		// Mirrors digitaltwin.TranslatorOptions.Validate: providing an
		// SQS region makes both queue URLs required.
		var err error
		if *region != "" {
			if *assetObservedURL == "" {
				err = errors.Join(err, errors.New("assetobservedurl is required when region is provided"))
			}
			if *assetsBehindGatewayStatusURL == "" {
				err = errors.Join(err, errors.New("assetsbehindgatewaystatusurl is required when region is provided"))
			}
		}
		if err != nil {
			return fmt.Errorf("options: %w", err)
		}

		// The representative loop retains dependency-ordered compilation across eleven input streams.
		log.Printf("digital-twin translator: fetch limit %d: %q and 10 peers -> %q",
			limit, *consumesAssetObserved, *producesRemoteCompilation)
		<-ctx.Done()
		return ctx.Err()
	}), fs
}

// GraphComponent applies mutation batches to the target network's digital twin.
var GraphComponent = &application.Descriptor{
	Name: "digitalTwinGraph",
	Doc:  "digitalTwinGraph applies mutation batches to the Neo4j-backed digital twin and publishes a digest of each completed change set.",
	Make: application.MakeFunc(makeGraph),
}

func makeGraph() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("digitalTwinGraph", flag.ContinueOnError)

	// The provisioned Neo4j account supplies the graph connection.
	neo4jURI := fs.String("neo4j", "", "URL of the backing neo4j server")
	parameter.Require(fs, "neo4j")
	fs.String("user", "neo4j", "neo4j basic auth username")
	parameter.Require(fs, "user")
	fs.String("pass", "neo4j", "neo4j basic auth password")
	parameter.Require(fs, "pass")

	// Graph batching and timeout options are validated together at startup.
	cycle := fs.Duration("cycle", 0, "changeset flush frequency")
	database := fs.String("database", "", "name of the neo4j database to use")
	fetchLimit := fs.Int("fetchLimit", 0, "max compilations fetched concurrently")
	applyLimit := fs.Int("applyLimit", 0, "max compilations applied per batch, at most fetchLimit")
	batchTTL := fs.Duration("batchTTL", 0, "max wait before flushing a compilation batch")
	apocBatchSize := fs.Int("apocBatchSize", 0, "APOC write-pipelining batch size")
	fs.Bool("doAPOCParallel", false, "use APOC parallel processing for write pipelining")
	readTimeout := fs.Duration("readTimeout", 0, "max duration for reading assemblies")
	writeTimeout := fs.Duration("writeTimeout", 0, "max duration for writing assemblies")
	initialSnapshotTimeout := fs.Duration("initialSnapshotTimeout", 0, "max wait for the initial snapshot before processing compilations")
	receiveTimeout := fs.Duration("receiveTimeout", 0, "watchdog: shut down if no message arrives in time; zero disables")

	// Message-bus wiring: one interest, one aspect.
	fs.String("consumesRemoteCompilation", "", "topic of graph mutation batches")
	producesGraphChanged := fs.String("producesGraphChanged", "", "topic for gob-encoded graph-change digests")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Mirrors digitaltwin.GraphOptions.Validate, including the
		// cross-parameter rule the IR must not preclude: fetchLimit
		// is meaningless below applyLimit.
		var err error
		if *cycle == 0 {
			err = errors.Join(err, errors.New("cycle is required"))
		}
		if *database == "" {
			err = errors.Join(err, errors.New("database is required"))
		}
		if *fetchLimit == 0 {
			err = errors.Join(err, errors.New("fetchlimit is required"))
		}
		if *applyLimit == 0 {
			err = errors.Join(err, errors.New("applylimit is required"))
		}
		if *fetchLimit < *applyLimit {
			err = errors.Join(err, fmt.Errorf("fetchlimit (%d) is meaningless when less than applylimit (%d)", *fetchLimit, *applyLimit))
		}
		if *batchTTL <= 0 {
			err = errors.Join(err, errors.New("batchttl must be non-negative"))
		}
		if *apocBatchSize <= 0 {
			err = errors.Join(err, errors.New("apocbatchsize must be non-negative"))
		}
		if *readTimeout <= 0 {
			err = errors.Join(err, errors.New("readtimeout must be non-negative"))
		}
		if *writeTimeout <= 0 {
			err = errors.Join(err, errors.New("writetimeout must be non-negative"))
		}
		if *initialSnapshotTimeout <= 0 {
			err = errors.Join(err, errors.New("initial-snapshot-timeout must be non-negative"))
		}
		if *receiveTimeout < 0 {
			err = errors.Join(err, errors.New("receive-timeout must be non-negative"))
		}
		if err != nil {
			return fmt.Errorf("options: %w", err)
		}

		// The representative loop retains batched graph updates and change notifications.
		log.Printf("digital-twin graph: twin database %q on %s", *database, *neo4jURI)
		tick := time.NewTicker(*cycle)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick.C:
				log.Printf("digital-twin graph: cycle: would publish GraphChanged to %q", *producesGraphChanged)
			}
		}
	}), fs
}
