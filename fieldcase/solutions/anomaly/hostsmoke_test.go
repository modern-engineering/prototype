// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package anomaly

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The dev site's sensitive values: none of these may ever appear in
// host output, in any mode. The list mirrors site/dev.site.
var devSecrets = []string{
	"0abcdef0-dev0-4000-8000-devaccount00", // extern.tenantID
	"0abcdef0-dev0-4000-8000-devorg000000", // extern.organizationID
	"dev-token-cache-pass",
	"dev-anomaly-pass",
	"dev-twin-pass",
	"dev-alerts-pass",
}

// TestHostGateEmptySite is the negative half of the stage-(c) gate:
// with an empty site, the host refuses to start and lists every one of
// the solution's seven externs with its type, before any Runner or
// driver runs.
func TestHostGateEmptySite(t *testing.T) {
	short(t)
	empty := filepath.Join(t.TempDir(), "empty.site")
	if err := os.WriteFile(empty, nil, 0o666); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(hostBin, empty)
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("host did not fail: err=%v\n%s", err, out)
	}
	if code := exit.ExitCode(); code != 2 {
		t.Errorf("exit code = %d, want 2 (configuration fault)\n%s", code, out)
	}
	for _, want := range []string{
		"unbound extern tenantID (substrate.Secret): the site must bind extern.tenantID",
		"unbound extern kafkaCluster (substrate.KafkaCluster)",
		"unbound extern neo4jServer (substrate.Neo4jServer)",
		"unbound extern organizationID (substrate.Secret)",
		"unbound extern postgresServer (substrate.PostgresServer)",
		"unbound extern redisServer (substrate.RedisServer)",
		"unbound extern siemServer (substrate.Endpoint)",
		"7 unbound extern(s); refusing to start",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("gate output is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "driver:") || strings.Contains(string(out), "running") {
		t.Errorf("the gate must fire before drivers and runners:\n%s", out)
	}
}

// TestHostSmoke is the positive lifecycle proof: with the dev site the
// host provisions through the dev drivers, logs the effective value of
// every resolved symbol (sensitive values redacted), reports every
// instance running, and on SIGTERM after a two-second run shuts down
// cleanly in the Shutdown -> Cancel -> Wait order, exit 0.
func TestHostSmoke(t *testing.T) {
	short(t)
	cmd := exec.Command(hostBin, "solutions/anomaly/site/dev.site")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "HEALTH=127.0.0.1:0")
	out := runUntilReady(t, cmd, "; ready")

	// A two-second run: the simplified bodies log and loop, so this is
	// a scenario lifecycle window with no I/O beyond the health listener.
	time.Sleep(2 * time.Second)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, cmd, 15*time.Second); err != nil {
		t.Errorf("host exited uncleanly after SIGTERM: %v\n%s", err, out.String())
	}
	logged := out.String()

	// The A-10 effective-value audit, redactions included.
	for _, want := range []string{
		`audit: extern tenantID = <redacted> (site, substrate.Secret)`,
		`audit: extern kafkaCluster = "localhost:9092" (site, substrate.KafkaCluster)`,
		`audit: extern siemServer = "127.0.0.1:5514" (site, substrate.Endpoint)`,
		`audit: var networkController = "http://127.0.0.1:8081/api" (site override)`,
		`audit: var topicAlerts = "security.alerts" (image default)`,
		`audit: output kafka.brokers = "localhost:9092" (driver, slice substrate.Kafka)`,
		`audit: output identityState.password = <redacted> (driver, slice substrate.Redis)`,
		`audit: identityAnomalyDetector.redisPassword = <redacted> (record output identityState.password, instance)`,
		`audit: identityAnomalyDetector.redisAddress = "127.0.0.1:6379" (record output identityState.address, instance)`,
		`audit: networkObservationCollector.controllerTenantID = <redacted> (record ref tenantID, instance)`,
		`audit: observationTranslator.kafkaBrokers = "localhost:9092" (record output kafka.brokers, default-deploy)`,
		`audit: digitalTwinGraph.pass = <redacted> (record output twinDB.password, instance)`,
		`audit: digitalTwinGraph.receiveTimeout = "0s" (catalogue default)`,
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("audit is missing %q", want)
		}
	}

	// The A-11 driver lifecycle posture and the consumer-group identity
	// derivation (the stable runtime identity).
	for _, want := range []string{
		"driver: slice kafka (substrate.Kafka)",
		"delete protection",
		"identity: digitaltwin consumer group = anomaly/digitaltwin",
		"hosting 7 of 7 instances",
		"running 7 instance(s); ready",
		"health: listening on 127.0.0.1:",
		"received terminated; shutting down",
		"shutdown complete",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("host log is missing %q", want)
		}
	}

	// The instances really started: their (simplified) bodies log their
	// startup lines before any tick.
	for _, want := range []string{
		"network observation collector: account 0abcdef0 polling http://127.0.0.1:8081/api",
		"observation translator: 4 pipelines",
		"digital-twin translator: fetch limit 1",
		`digital-twin graph: twin database "twin-db"`,
		`identity anomaly detector: policy "UnexpectedEndpoint"`,
		"alerts journal:",
		"syslog exporter:",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("instance startup log is missing %q", want)
		}
	}

	assertNoSecrets(t, logged)
}

// TestHostEnableSelection proves the tri-state enable semantics and
// the scoped extern gate: a pod-shaped run selecting one instance via
// its env alias hosts exactly that instance, audits only the symbols
// it reaches, and needs only its externs bound.
func TestHostEnableSelection(t *testing.T) {
	short(t)
	cmd := exec.Command(hostBin, "solutions/anomaly/site/dev.site")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "HEALTH=127.0.0.1:0", "IDENTITYANOMALYDETECTOR=true")
	out := runUntilReady(t, cmd, "; ready")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err := waitFor(t, cmd, 15*time.Second); err != nil {
		t.Errorf("host exited uncleanly after SIGTERM: %v\n%s", err, out.String())
	}
	logged := out.String()
	for _, want := range []string{
		"hosting 1 of 7 instances: [identityAnomalyDetector]",
		"running 1 instance(s); ready",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("host log is missing %q:\n%s", want, logged)
		}
	}
	// The audit is scoped to what this process consumes: the CONTROLLER
	// account secret feeds only the (not hosted) instrument.
	if strings.Contains(logged, "audit: extern tenantID") {
		t.Errorf("audit leaks a symbol no hosted instance reaches:\n%s", logged)
	}
	assertNoSecrets(t, logged)

	// The scoped gate: the detector's pod needs only its own externs.
	empty := filepath.Join(t.TempDir(), "empty.site")
	if err := os.WriteFile(empty, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	gate := exec.Command(hostBin, empty)
	gate.Dir = moduleRoot
	gate.Env = append(os.Environ(), "IDENTITYANOMALYDETECTOR=true")
	gateOut, err := gate.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("scoped gate did not fail with exit 2: %v\n%s", err, gateOut)
	}
	if !strings.Contains(string(gateOut), "2 unbound extern(s)") ||
		!strings.Contains(string(gateOut), "unbound extern kafkaCluster") ||
		!strings.Contains(string(gateOut), "unbound extern redisServer") {
		t.Errorf("scoped gate should demand exactly the detector's externs:\n%s", gateOut)
	}
}

// runUntilReady starts cmd, streams its stderr into the returned
// buffer, and blocks until the marker line appears (or fails the
// test). The buffer keeps filling in the background afterwards.
func runUntilReady(t *testing.T, cmd *exec.Cmd, marker string) *syncBuffer {
	t.Helper()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	buf := &syncBuffer{}
	ready := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderr)
		signalled := false
		for scanner.Scan() {
			buf.append(scanner.Text() + "\n")
			if !signalled && strings.Contains(scanner.Text(), marker) {
				signalled = true
				close(ready)
			}
		}
	}()
	select {
	case <-ready:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		t.Fatalf("host never logged %q:\n%s", marker, buf.String())
	}
	return buf
}

// waitFor waits for the process to exit cleanly within the timeout.
func waitFor(t *testing.T, cmd *exec.Cmd, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		cmd.Process.Kill()
		return io.ErrNoProgress
	}
}

// assertNoSecrets asserts the redaction discipline: no sensitive site
// value ever reaches the output.
func assertNoSecrets(t *testing.T, logged string) {
	t.Helper()
	for _, secret := range devSecrets {
		if strings.Contains(logged, secret) {
			t.Errorf("output leaks the sensitive value %q", secret)
		}
	}
}

// A syncBuffer is a mutex-guarded string buffer shared between the
// scanner goroutine and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) append(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(s)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
