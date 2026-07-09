// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package driver

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// capture routes the package logger through a buffer for the duration
// of one test.
func capture(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prev)
		log.SetFlags(prevFlags)
	})
	return &buf
}

func TestSliceEchoesAndLogsDeleteProtection(t *testing.T) {
	buf := capture(t)
	out, err := DevRedis{}.Provision(context.Background(), Record{
		Name: "identityState",
		Type: "substrate.Redis",
		Kind: "slice",
		Site: map[string]string{"address": "localhost:6379", "db": "1", "password": "hunter2"},
		Need: []string{"address", "db"},
	}, map[string]string{"server": "localhost:6379", "db": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if out["address"] != "localhost:6379" || out["password"] != "hunter2" {
		t.Errorf("outputs = %v, want the site section echoed", out)
	}
	logged := buf.String()
	if !strings.Contains(logged, "slice identityState") || !strings.Contains(logged, "delete protection") {
		t.Errorf("slice log misses the delete-protection posture:\n%s", logged)
	}
}

func TestAttachLogsNeverPruned(t *testing.T) {
	buf := capture(t)
	_, err := DevKafka{}.Provision(context.Background(), Record{
		Name: "kafka",
		Type: "substrate.Kafka",
		Kind: "attach",
		Site: map[string]string{"brokers": "edge-gateway:23242"},
		Need: []string{"brokers"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if logged := buf.String(); !strings.Contains(logged, "attach kafka") || !strings.Contains(logged, "never pruned") {
		t.Errorf("attach log misses the never-pruned posture:\n%s", logged)
	}
}

func TestMissingNeededOutputs(t *testing.T) {
	capture(t)
	_, err := DevPostgres{}.Provision(context.Background(), Record{
		Name: "alertsDB",
		Type: "substrate.Postgres",
		Kind: "slice",
		Site: map[string]string{"address": "localhost:5432"},
		Need: []string{"address", "username", "password"},
	}, nil)
	if err == nil {
		t.Fatal("Provision succeeded despite unconfigured needed outputs")
	}
	for _, want := range []string{"driver.alertsDB.username", "driver.alertsDB.password"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q misses %q", err, want)
		}
	}
}
