// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package controller provides the network-observation collector for the field case.
// It polls a network controller for asset, session, link, gateway, and
// traffic observations, caches API tokens in Redis, and publishes raw
// observations to Kafka.
package controller

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/parameter"
)

// Collector polls the network controller and publishes raw observations.
var Collector = &application.Descriptor{
	Name: "networkObservationCollector",
	Doc:  "networkObservationCollector polls a network controller for asset, session, link, gateway, and traffic observations. It caches API tokens in Redis and publishes the observations to Kafka.",
	URL:  "https://controller.example",
	Make: application.MakeFunc(makeCollector),
}

// networks is a comma-separated list of controller scopes.
// Each Set replaces the previous selection.
type networks []string

func (l *networks) String() string { return strings.Join(*l, ",") }

func (l *networks) Set(s string) error {
	*l = strings.Split(s, ",")
	return nil
}

func makeCollector() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("networkObservationCollector", flag.ContinueOnError)

	// Controller and token-cache credentials are required operator inputs.
	tenantID := fs.String("controllerTenantID", "", "controller tenant identifier")
	parameter.Require(fs, "controllerTenantID")
	fs.String("controllerOrganizationID", "", "controller organization identifier")
	fs.String("redisUser", "", "token-cache Redis username")
	parameter.Require(fs, "redisUser")
	fs.String("redisPassword", "", "token-cache Redis password")
	parameter.Require(fs, "redisPassword")

	// Polling, retry, controller, token-cache, and normalization options are
	// explicit so a site can reproduce the collector's behavior.
	pollingInterval := fs.Duration("pollingInterval", 0, "period between controller polls")
	fs.Int("retryMax", 0, "max retries per controller HTTP request")
	fs.Duration("backoff", 0, "constant backoff between HTTP retries")
	baseURL := fs.String("baseURL", "", "network controller base URL")
	redisConnHost := fs.String("redisConnHost", "", "token-cache Redis host")
	fs.Int("redisConnPort", 0, "token-cache Redis port")
	fs.Int("redisConnDB", 0, "token-cache Redis logical database")
	fs.String("fileTokensPath", "", "directory of bootstrap token files, a mounted Secret volume")
	fs.Bool("fileTokensReadOnly", false, "whether the bootstrap token files are read-only")
	var focused networks
	fs.Var(&focused, "focusedNetworks", "comma-separated controller networks to poll")
	maxWorkers := fs.Int("maxWorkers", 0, "concurrent console fetch workers, at least 1")
	apiUsername := fs.String("apiUsername", "", "API username for controller login")
	apiPassword := fs.String("apiPassword", "", "basic-auth password for controller login")
	fs.Bool("normalizeHardwareAddress", false, "canonicalize hardware addresses in asset observations")

	// Each observation family has an explicit output channel.
	producesAssetObservation := fs.String("producesAssetObservation", "", "topic for raw asset observations")
	fs.String("producesSession", "", "topic for raw session reports")
	fs.String("producesLink", "", "topic for raw connectivity reports")
	fs.String("producesGateway", "", "topic for raw gateway reports")
	fs.String("producesTrafficStatus", "", "topic for traffic status samples")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Mirrors visibility.Options.validate: BaseURL, RedisConn.Host,
		// FocusedNetworks, and MaxWorkers are required; partial basic
		// auth (email without password) is rejected.
		var err error
		if *baseURL == "" {
			err = errors.Join(err, errors.New("required: baseURL"))
		}
		if *redisConnHost == "" {
			err = errors.Join(err, errors.New("required: redisConnHost"))
		}
		if len(focused) == 0 {
			err = errors.Join(err, errors.New("required: focusedNetworks"))
		}
		if *maxWorkers < 1 {
			err = errors.Join(err, errors.New("required: maxWorkers"))
		}
		if *apiUsername != "" && *apiPassword == "" {
			err = errors.Join(err, errors.New("required: apiPassword"))
		}
		if err != nil {
			return fmt.Errorf("options: %w", err)
		}
		// Reject an invalid interval before constructing the ticker.
		if *pollingInterval <= 0 {
			return errors.New("options: pollingInterval must be positive")
		}

		// The representative loop preserves the collector's polling cadence and fan-out.
		log.Printf("network observation collector: account %.8s polling %s (networks %v)", *tenantID, *baseURL, focused)
		tick := time.NewTicker(*pollingInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-tick.C:
				log.Printf("network observation collector: poll cycle: would publish asset observations to %q", *producesAssetObservation)
			}
		}
	}), fs
}
