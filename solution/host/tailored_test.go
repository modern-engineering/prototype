// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host_test

import (
	"bytes"
	"testing"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
)

// tailoredUnit is the embedded solution of the Mode-T tests: the M1
// shape in SDL, compiled in memory against the live mill catalogue —
// the one path here that runs the real compiler in front of the host.
const tailoredUnit = `solution millrun

import "example.com/acme/mill"

extern endpoint mill.Endpoint

provision mill.Stand attach as standIn {
	params {
		endpoint: endpoint
	}
}

deploy mill.Echo as E1 {
	params {
		count: 1
		nats: standIn.config
		subject: "tailored"
	}
}
`

func tailoredConfig(ctl *standControl, reports chan report, stderr *bytes.Buffer, source string) solution.CompileConfig {
	return solution.CompileConfig{
		Solution:  "millrun",
		Units:     []solution.Unit{{Name: "millrun.sdl", Source: source}},
		Catalogue: millCatalogue(ctl, reports),
		Stderr:    stderr,
	}
}

// The whole Mode-T bridge holds: sources compile in memory, the
// extern flag binds the site value, and the image enacts to
// completion — sources to a ran solution in one call, the dev loop's
// inner shape.
func TestMainTailoredRunsSolution(t *testing.T) {
	ctl := newStandControl()
	reports := make(chan report, 8)
	var stderr bytes.Buffer
	code := host.MainTailored(
		tailoredConfig(ctl, reports, &stderr, tailoredUnit),
		// The stray extern exercises the permissive door: supplied
		// names the image never declares are ignored.
		[]string{"-extern", "endpoint=mill:9", "-extern", "stray=x", "-grace", "1s"},
	)
	if code != 0 {
		t.Fatalf("MainTailored() = %d, want 0; compile stderr:\n%s", code, stderr.String())
	}
	if got := <-ctl.got; got != "mill:9" {
		t.Errorf("driver endpoint = %q, want %q", got, "mill:9")
	}
	want := report{subject: "tailored", nats: "stand://mill", count: 1}
	if got := drain(t, reports, 1)[0]; got != want {
		t.Errorf("report = %+v, want %+v", got, want)
	}
}

// A solution that does not compile is a configuration fault under run
// semantics — exit 2, diagnostics printed positioned exactly as sdl
// build prints them.
func TestMainTailoredCompileFailure(t *testing.T) {
	var stderr bytes.Buffer
	broken := "solution millrun\n\ndeploy mill.Echo as E1\n" // mill is never imported
	code := host.MainTailored(tailoredConfig(newStandControl(), nil, &stderr, broken), nil)
	if code != 2 {
		t.Fatalf("MainTailored() = %d, want 2", code)
	}
	if want := "millrun.sdl:3:8: package mill is not imported\n"; stderr.String() != want {
		t.Errorf("compile stderr = %q, want %q", stderr.String(), want)
	}
}

// The flag surface refuses malformed input as configuration, before
// compiling anything.
func TestMainTailoredBadArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"-bogus"}},
		{"extern without value", []string{"-extern", "endpoint"}},
		{"extern without name", []string{"-extern", "=v"}},
		{"positional leftovers", []string{"leftover"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := host.MainTailored(tailoredConfig(newStandControl(), nil, &stderr, tailoredUnit), tt.args)
			if code != 2 {
				t.Errorf("MainTailored(%q) = %d, want 2", tt.args, code)
			}
			if stderr.Len() != 0 {
				t.Errorf("bad args reached the compiler; stderr:\n%s", stderr.String())
			}
		})
	}
}

// The teaching error closes its loop: the -extern hint the gate
// prints names the very flag this surface parses.
func TestMainTailoredGatesExterns(t *testing.T) {
	var stderr bytes.Buffer
	code := host.MainTailored(tailoredConfig(newStandControl(), nil, &stderr, tailoredUnit), nil)
	if code != 2 {
		t.Fatalf("MainTailored() = %d, want 2", code)
	}
}
