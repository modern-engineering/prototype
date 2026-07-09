// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	path := write(t, "dev.site", `
# a comment
extern.kafkaCluster = localhost:9092
var.networkController = http://127.0.0.1:8081/api
driver.kafka.brokers = localhost:9092
driver.twinDB.password = s3cr#t = with = equals
`)
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := s.Extern("kafkaCluster"); !ok || v != "localhost:9092" {
		t.Errorf("Extern(kafkaCluster) = %q, %v", v, ok)
	}
	if v, ok := s.Var("networkController"); !ok || v != "http://127.0.0.1:8081/api" {
		t.Errorf("Var(networkController) = %q, %v", v, ok)
	}
	if v := s.Driver("kafka")["brokers"]; v != "localhost:9092" {
		t.Errorf("Driver(kafka)[brokers] = %q", v)
	}
	// Values keep everything past the first '=': later equals signs and
	// '#' are value bytes, not syntax.
	if v := s.Driver("twinDB")["password"]; v != "s3cr#t = with = equals" {
		t.Errorf("Driver(twinDB)[password] = %q", v)
	}
	if got := s.Externs(); len(got) != 1 || got[0] != "kafkaCluster" {
		t.Errorf("Externs() = %v", got)
	}
}

func TestLoadEmpty(t *testing.T) {
	s, err := Load(write(t, "empty.site", ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Externs())+len(s.Vars())+len(s.Drivers()) != 0 {
		t.Errorf("empty site is not empty: %v %v %v", s.Externs(), s.Vars(), s.Drivers())
	}
}

func TestLoadFaults(t *testing.T) {
	tests := []struct {
		name, content, want string
	}{
		{"NotAPair", "extern.kafkaCluster localhost", "not a key = value line"},
		{"EmptyKey", " = value", "empty key"},
		{"UnknownNamespace", "secret.foo = bar", `unknown key namespace "secret"`},
		{"BareExtern", "extern = x", "extern key names no symbol"},
		{"DriverShape", "driver.kafka = x", "driver key must be driver.<instance>.<output>"},
		{"Duplicate", "extern.a = 1\nextern.a = 2", "duplicate key extern.a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(write(t, "bad.site", tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

// TestLoadDuplicateAcrossFiles pins the merge rule the generator's
// fragment split relies on: fragments are disjoint, so a key bound by
// two files of one load is an error naming both origins.
func TestLoadDuplicateAcrossFiles(t *testing.T) {
	a := write(t, "a.site", "extern.x = 1")
	b := write(t, "b.site", "extern.x = 2")
	_, err := Load(a, b)
	if err == nil || !strings.Contains(err.Error(), "duplicate key extern.x") {
		t.Errorf("Load() error = %v, want a duplicate-key error", err)
	}
}
