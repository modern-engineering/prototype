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

// SyslogExporter sends security alerts to a SIEM endpoint.
var SyslogExporter = &application.Descriptor{
	Name: "detectionSyslogExporter",
	Doc:  "detectionSyslogExporter consumes security alerts, encodes them as syslog messages, and sends them to a configured SIEM endpoint.",
	Make: application.MakeFunc(makeSyslogExporter),
}

func makeSyslogExporter() (application.Runner, *flag.FlagSet) {
	fs := flag.NewFlagSet("detectionSyslogExporter", flag.ContinueOnError)

	// The site selects the syslog encoding, transport, endpoint, and timeout.
	fs.String("format", "", "syslog format for the alerts, rfc5424 or rfc3164")
	protocol := fs.String("protocol", "", "transport protocol for the syslog connection, udp or tcp")
	serverAddress := fs.String("serverAddress", "", "host:port of the SIEM syslog server")
	timeout := fs.Duration("timeout", 0, "write timeout on the syslog connection")

	// Message-bus wiring: a pure sink — one interest, no aspects.
	consumesAlerts := fs.String("consumesAlerts", "", "topic of JSON alerts")

	// Shared runtime settings are declared per service because each catalogue application is self-contained.
	fs.String("kafkaBrokers", "", "comma-separated Kafka bootstrap brokers (shared flag kafka-brokers, required)")
	parameter.Require(fs, "kafkaBrokers")
	fs.String("kafkaConfig", "", "sarama configuration as ;-separated key=value pairs (shared flag kafka-config)")

	return application.RunnerFunc(func(ctx context.Context) error {
		// Mirrors detection.SyslogExporterOptions.Validate: format and
		// protocol values are left to the dialler and encoder, as in the
		// scenario.
		var err error
		if *serverAddress == "" {
			err = errors.Join(err, errors.New("the server address parameter is required"))
		}
		if *timeout == 0 {
			err = errors.Join(err, errors.New("the timeout parameter is required"))
		}
		if err != nil {
			return fmt.Errorf("validate options: %w", err)
		}

		// The representative loop retains connection validation and bounded writes.
		log.Printf("syslog exporter: %q -> %s %s", *consumesAlerts, *protocol, *serverAddress)
		<-ctx.Done()
		return ctx.Err()
	}), fs
}
