// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"slices"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution"
)

// schemeOutputs declares one output of every scalar type plus an
// untyped one, proving the empty type is string's second spelling.
func schemeOutputs() []solution.Output {
	return []solution.Output{
		{Name: "config", Doc: "untyped, so string by default"},
		{Name: "url", Doc: "explicitly string", Type: solution.OutputString},
		{Name: "port", Doc: "an int", Type: solution.OutputInt},
		{Name: "ready", Doc: "a bool", Type: solution.OutputBool},
		{Name: "ttl", Doc: "a duration", Type: solution.OutputDuration},
	}
}

// TestOutputWriterRoundTrip drives every output through its matching
// setter and reads each back typed and rendered. The rendered strings
// must be the spellings flag.Value.Set accepts — base-10, true/false,
// canonical Go durations — since wet binding feeds them to the very
// same slots inline literals bind through.
func TestOutputWriterRoundTrip(t *testing.T) {
	w := solution.NewOutputWriter(schemeOutputs())

	if err := w.SetString("config", "accounts/demo.conf"); err != nil {
		t.Fatalf("SetString(config) = %v", err)
	}
	if err := w.SetString("url", "nats://localhost:4222"); err != nil {
		t.Fatalf("SetString(url) = %v", err)
	}
	if err := w.SetInt("port", 4222); err != nil {
		t.Fatalf("SetInt(port) = %v", err)
	}
	if err := w.SetBool("ready", true); err != nil {
		t.Fatalf("SetBool(ready) = %v", err)
	}
	if err := w.SetDuration("ttl", 90*time.Second); err != nil {
		t.Fatalf("SetDuration(ttl) = %v", err)
	}

	if got, err := w.GetString("config"); err != nil || got != "accounts/demo.conf" {
		t.Errorf("GetString(config) = %q, %v", got, err)
	}
	if got, err := w.GetInt("port"); err != nil || got != 4222 {
		t.Errorf("GetInt(port) = %d, %v", got, err)
	}
	if got, err := w.GetBool("ready"); err != nil || !got {
		t.Errorf("GetBool(ready) = %t, %v", got, err)
	}
	if got, err := w.GetDuration("ttl"); err != nil || got != 90*time.Second {
		t.Errorf("GetDuration(ttl) = %v, %v", got, err)
	}

	renders := map[string]string{
		"config": "accounts/demo.conf",
		"url":    "nats://localhost:4222",
		"port":   "4222",
		"ready":  "true",
		"ttl":    "1m30s",
	}
	for name, want := range renders {
		if got, err := w.Render(name); err != nil || got != want {
			t.Errorf("Render(%s) = %q, %v; want %q", name, got, err, want)
		}
	}

	if missing := w.Missing(); len(missing) != 0 {
		t.Errorf("Missing() = %v after a complete write, want none", missing)
	}
}

// TestOutputWriterBoundary pins the boundary error vocabulary: faults
// surface at the call that crosses the contract, with the message
// naming the output and the declared type.
func TestOutputWriterBoundary(t *testing.T) {
	tests := []struct {
		name string
		call func(w *solution.OutputWriter) error
		want string
	}{
		{"set undeclared", func(w *solution.OutputWriter) error {
			return w.SetString("nope", "x")
		}, "undeclared output nope"},
		{"set through the wrong type", func(w *solution.OutputWriter) error {
			return w.SetInt("config", 1)
		}, "output config is string, not int"},
		{"set the untyped output as non-string", func(w *solution.OutputWriter) error {
			return w.SetBool("config", true)
		}, "output config is string, not bool"},
		{"set twice", func(w *solution.OutputWriter) error {
			if err := w.SetInt("port", 1); err != nil {
				return err
			}
			return w.SetInt("port", 2)
		}, "output port written twice"},
		{"get undeclared", func(w *solution.OutputWriter) error {
			_, err := w.GetString("nope")
			return err
		}, "undeclared output nope"},
		{"get through the wrong type", func(w *solution.OutputWriter) error {
			if err := w.SetInt("port", 1); err != nil {
				return err
			}
			_, err := w.GetBool("port")
			return err
		}, "output port is int, not bool"},
		{"get unwritten", func(w *solution.OutputWriter) error {
			_, err := w.GetInt("port")
			return err
		}, "output port is unwritten"},
		{"render undeclared", func(w *solution.OutputWriter) error {
			_, err := w.Render("nope")
			return err
		}, "undeclared output nope"},
		{"render unwritten", func(w *solution.OutputWriter) error {
			_, err := w.Render("ttl")
			return err
		}, "output ttl is unwritten"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(solution.NewOutputWriter(schemeOutputs()))
			if err == nil || err.Error() != tt.want {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// TestOutputWriterMissing holds Missing to declaration order and to
// shrinking as writes land — the host's completeness gate reads it
// verbatim into its per-output faults.
func TestOutputWriterMissing(t *testing.T) {
	w := solution.NewOutputWriter(schemeOutputs())
	want := []string{"config", "url", "port", "ready", "ttl"}
	if got := w.Missing(); !slices.Equal(got, want) {
		t.Fatalf("Missing() = %v, want declaration order %v", got, want)
	}
	if err := w.SetString("url", "u"); err != nil {
		t.Fatal(err)
	}
	if err := w.SetBool("ready", false); err != nil {
		t.Fatal(err)
	}
	want = []string{"config", "port", "ttl"}
	if got := w.Missing(); !slices.Equal(got, want) {
		t.Errorf("Missing() = %v, want %v", got, want)
	}
}
