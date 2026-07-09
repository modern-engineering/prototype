// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package substrate_test

import (
	"flag"
	"reflect"
	"testing"

	"github.com/modern-engineering/prototype/solution"

	"github.com/modern-engineering/prototype/fieldcase/catalog/substrate"
)

// TestSymbolTypes pins the discovery-visible shape: exported pointer
// vars, documented, with the intended sensitivity.
func TestSymbolTypes(t *testing.T) {
	for name, st := range map[string]*solution.SymbolType{
		"KafkaCluster":   substrate.KafkaCluster,
		"RedisServer":    substrate.RedisServer,
		"Neo4jServer":    substrate.Neo4jServer,
		"PostgresServer": substrate.PostgresServer,
		"Secret":         substrate.Secret,
		"Endpoint":       substrate.Endpoint,
	} {
		if st == nil {
			t.Errorf("%s is nil; discovery needs a live *solution.SymbolType", name)
			continue
		}
		if st.Doc == "" {
			t.Errorf("%s has no Doc; the image pins element documentation", name)
		}
	}
	if !substrate.Secret.Sensitive {
		t.Error("Secret must be sensitive: scenario credentials originate in Kubernetes Secrets")
	}
}

// TestProvisionTypes holds the provision citizens to their contract:
// explicit kinds, documented outputs, and a dry Params hook.
func TestProvisionTypes(t *testing.T) {
	for name, pt := range map[string]*solution.ProvisionType{
		"Kafka":    substrate.Kafka,
		"Redis":    substrate.Redis,
		"Neo4j":    substrate.Neo4j,
		"Postgres": substrate.Postgres,
	} {
		t.Run(name, func(t *testing.T) {
			if pt == nil {
				t.Fatalf("%s is nil; discovery needs a live *solution.ProvisionType", name)
			}
			if pt.Doc == "" {
				t.Error("no Doc; the image pins element documentation")
			}
			if pt.Kinds == 0 {
				t.Error("no Kinds; registration is explicit and zero kinds is a catalogue error")
			}
			for _, out := range pt.Outputs {
				if out.Doc == "" {
					t.Errorf("output %s has no Doc", out.Name)
				}
			}
			if pt.Params == nil {
				return
			}
			if got, want := dryNames(pt), dryNames(pt); !reflect.DeepEqual(got, want) {
				t.Errorf("Params is not dry: two fresh sets declare %v and %v", got, want)
			}
		})
	}
	// Credentials the scenario deployment routes through Kubernetes Secrets must
	// carry the taint so image consumers redact them.
	for _, pt := range []struct {
		name string
		typ  *solution.ProvisionType
	}{
		{"Redis", substrate.Redis},
		{"Neo4j", substrate.Neo4j},
		{"Postgres", substrate.Postgres},
	} {
		for _, out := range pt.typ.Outputs {
			switch out.Name {
			case "username", "password":
				if !out.Sensitive {
					t.Errorf("%s.%s must be sensitive: the field case value lives in a Secret", pt.name, out.Name)
				}
			}
		}
	}
}

// dryNames declares one throwaway flag set and reads back the slot
// names it carries.
func dryNames(pt *solution.ProvisionType) []string {
	fs := flag.NewFlagSet("dry", flag.ContinueOnError)
	pt.Params(fs)
	var names []string
	fs.VisitAll(func(f *flag.Flag) { names = append(names, f.Name) })
	return names
}
