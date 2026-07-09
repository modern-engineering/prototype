// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package detection provides the field case's security-alerting tail.
// The identity-anomaly detector publishes to a fan-in alert stream consumed by
// a PostgreSQL journal and a SIEM syslog exporter.
package detection
