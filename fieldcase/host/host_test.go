// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/application/parameter"
)

// TestNeeds pins the reachability closure: an enabled instance pulls
// in the symbols it references, the provisions behind its output
// references, and — transitively — what those provisions' parameters
// reference.
func TestNeeds(t *testing.T) {
	cfg := &Config{
		Symbols: []Symbol{
			{Name: "cluster", Class: ClassExtern, Type: "sub.Cluster"},
			{Name: "admin", Class: ClassExtern, Type: "sub.Secret", Sensitive: true},
			{Name: "topic", Class: ClassVar, Default: "t"},
			{Name: "unrelated", Class: ClassExtern, Type: "sub.Cluster"},
		},
		Provisions: []Provision{
			{
				Name: "acct", Type: "sub.NATS", Kind: "slice",
				Params: []Binding{
					{Key: "cluster", Kind: BindSymbol, Value: "cluster"},
					{Key: "admin", Kind: BindOutput, Instance: "adminAcct", Output: "config"},
				},
				Outputs: []Output{{Name: "config", Sensitive: true}},
			},
			{
				Name: "adminAcct", Type: "sub.NATS", Kind: "slice",
				Params: []Binding{
					{Key: "cluster", Kind: BindSymbol, Value: "admin"},
				},
				Outputs: []Output{{Name: "config", Sensitive: true}},
			},
			{
				Name: "idle", Type: "sub.NATS", Kind: "slice",
				Outputs: []Output{{Name: "config"}},
			},
		},
		Instances: []Instance{
			{Name: "one", Params: []Binding{
				{Key: "t", Kind: BindSymbol, Value: "topic"},
				{Key: "nats", Kind: BindOutput, Instance: "acct", Output: "config"},
			}},
			{Name: "two", Params: []Binding{
				{Key: "x", Kind: BindSymbol, Value: "unrelated"},
			}},
		},
	}
	n := cfg.Needs([]string{"one"})
	if !n.Vars["topic"] || !n.Externs["cluster"] || !n.Externs["admin"] {
		t.Errorf("closure misses transitive symbols: %+v", n)
	}
	if n.Externs["unrelated"] {
		t.Errorf("closure leaks a disabled instance's extern: %+v", n)
	}
	if !n.Provisions["acct"] || !n.Provisions["adminAcct"] || n.Provisions["idle"] {
		t.Errorf("provision closure wrong: %+v", n.Provisions)
	}
	if !n.Outputs["acct"]["config"] || !n.Outputs["adminAcct"]["config"] {
		t.Errorf("output needs wrong: %+v", n.Outputs)
	}
}

// gateApp is a minimal catalogue citizen whose one flag carries a
// parameter.Require mark and no record binding, so only the host's
// required-parameter gate stands between it and running.
var gateApp = &application.Descriptor{
	Name: "gateApp",
	Doc:  "gateApp exists to trip the required-parameter gate",
	Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		fs := flag.NewFlagSet("gateApp", flag.ContinueOnError)
		fs.String("token", "", "a required credential nothing binds")
		parameter.Require(fs, "token")
		return application.RunnerFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}), fs
	}),
}

// TestRequiredParamGate drives Main in-process against a config whose
// image binds nothing: the parameter.Require mark alone must refuse
// startup, after flag parsing but before any listener or Runner.
func TestRequiredParamGate(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.site")
	if err := os.WriteFile(empty, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	logged := captureMain(t, []string{"host.test", empty}, &Config{
		Solution:  "gatetest",
		Instances: []Instance{{Name: "inst", App: gateApp}},
	}, 2)
	for _, want := range []string{
		"inst.token is required and unbound",
		"1 required parameter(s) unbound; refusing to start",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("gate log is missing %q:\n%s", want, logged)
		}
	}
	if strings.Contains(logged, "health: listening") {
		t.Errorf("the gate must fire before the health listener binds:\n%s", logged)
	}
}

// TestBindingDriftGate pins the image-vs-binary drift check: a baked
// binding whose key the descriptor does not declare refuses startup.
func TestBindingDriftGate(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.site")
	if err := os.WriteFile(empty, nil, 0o666); err != nil {
		t.Fatal(err)
	}
	logged := captureMain(t, []string{"host.test", empty}, &Config{
		Solution: "drifttest",
		Instances: []Instance{{Name: "inst", App: gateApp, Params: []Binding{
			{Key: "token", Kind: BindLiteral, Value: "x", Source: "instance"},
			{Key: "gone", Kind: BindLiteral, Value: "y", Source: "instance"},
		}}},
	}, 2)
	if !strings.Contains(logged, `image binds "gone" but descriptor gateApp declares no such flag`) {
		t.Errorf("drift gate did not name the stray binding:\n%s", logged)
	}
}

// captureMain runs Main with a substituted os.Args and logger,
// asserting the exit code and returning the log.
func captureMain(t *testing.T, args []string, cfg *Config, wantCode int) string {
	t.Helper()
	prevArgs := os.Args
	prevOut := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()
	defer func() {
		os.Args = prevArgs
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	os.Args = args
	if code := Main(cfg); code != wantCode {
		t.Errorf("Main = %d, want %d\n%s", code, wantCode, buf.String())
	}
	return buf.String()
}
