// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/substrate"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/host"
	"github.com/modern-engineering/prototype/solution/image"
	"github.com/modern-engineering/prototype/solution/kubernetes"
)

var update = flag.Bool("update", false, "rewrite the golden manifests")

// millSite binds everything the shard corpus reaches, plus one var
// override so the rebind seam shows in the goldens.
func millSite() kubernetes.Site {
	return kubernetes.Site{
		Externs: map[string]string{"siloKey": "hunter2"},
		Vars:    map[string]string{"grade": "fine"},
	}
}

// TestRenderGoldens pins the whole basic contract byte-for-byte: one
// file per deploy record; a ConfigMap carrying the instance's shard
// and its plain externs; a Secret exactly where something tainted
// flows (Grinder's siloKey; Idle reaches nothing and gets none); a
// Deployment mounting both and honoring the k8s stanzas. The goldens
// double as the determinism check: a second render must equal the
// first, and -update rewrites the files when the contract moves.
func TestRenderGoldens(t *testing.T) {
	files, err := kubernetes.Render(testImage(), millSite(), kubernetes.Profile{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	again, err := kubernetes.Render(testImage(), millSite(), kubernetes.Profile{})
	if err != nil {
		t.Fatalf("Render (second): %v", err)
	}

	want := []string{"grinder.yaml", "idle.yaml"}
	if len(files) != len(want) {
		t.Fatalf("rendered %d files, want %v", len(files), want)
	}
	for _, name := range want {
		doc, ok := files[name]
		if !ok {
			t.Fatalf("rendered set misses %s", name)
		}
		if !bytes.Equal(doc, again[name]) {
			t.Errorf("%s renders differently twice", name)
		}
		golden := filepath.Join("testdata", name)
		if *update {
			if err := os.WriteFile(golden, doc, 0o666); err != nil {
				t.Fatalf("update %s: %v", golden, err)
			}
		}
		wantDoc, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("read %s (run with -update to create): %v", golden, err)
		}
		if !bytes.Equal(doc, wantDoc) {
			t.Errorf("%s drifts from its golden; run with -update and review the diff", name)
		}
	}
}

// TestRenderCustody pins the sensitivity split without the goldens'
// help: the tainted extern value appears in the Secret document and
// nowhere else, the plain half never grows a Secret, and only the pod
// with a Secret mounts one.
func TestRenderCustody(t *testing.T) {
	files, err := kubernetes.Render(testImage(), millSite(), kubernetes.Profile{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	grinder := string(files["grinder.yaml"])
	docs := strings.Split(grinder, "\n---\n")
	if len(docs) != 3 {
		t.Fatalf("grinder.yaml carries %d documents, want ConfigMap, Secret, Deployment", len(docs))
	}
	if !strings.Contains(docs[1], "kind: Secret") || !strings.Contains(docs[1], "siloKey = hunter2") {
		t.Errorf("the Secret document does not carry the tainted binding:\n%s", docs[1])
	}
	if strings.Contains(docs[0], "hunter2") || strings.Contains(docs[2], "hunter2") {
		t.Error("the tainted value leaked outside the Secret document")
	}
	if !strings.Contains(docs[2], "-externs="+kubernetes.SecretMount+"/externs.conf") {
		t.Error("the Deployment does not feed the host the secret externs file")
	}

	idle := string(files["idle.yaml"])
	if strings.Contains(idle, "kind: Secret") {
		t.Error("idle reaches nothing tainted yet renders a Secret")
	}
	if !strings.Contains(idle, "# no plain externs reach this instance") {
		t.Error("idle's extern file placeholder is missing; argument stability depends on it")
	}
}

// TestRenderGates pins the render-time refusals, all reported at
// once: a site key naming no symbol of its class (the typo gate), and
// an instance whose closure reaches an extern the site leaves
// unbound — the MUST-bind rule enforced first where artifacts are
// made, before any pod ever crash-loops on it.
func TestRenderGates(t *testing.T) {
	img := testImage()

	_, err := kubernetes.Render(img, kubernetes.Site{
		Externs: map[string]string{"siloKey": "x", "grade": "oops", "ghost": "y"},
		Vars:    map[string]string{"siloKey": "z"},
	}, kubernetes.Profile{})
	if err == nil {
		t.Fatal("a site full of typos rendered; want the typo gate")
	}
	for _, wants := range []string{"grade", "ghost", "siloKey"} {
		if !strings.Contains(err.Error(), wants) {
			t.Errorf("typo gate %q does not name %s", err, wants)
		}
	}

	_, err = kubernetes.Render(img, kubernetes.Site{}, kubernetes.Profile{})
	if err == nil {
		t.Fatal("an empty site rendered; want the MUST-bind gate")
	}
	if !strings.Contains(err.Error(), "Grinder") || !strings.Contains(err.Error(), "siloKey") {
		t.Errorf("gate %q does not name the instance and its unbound extern", err)
	}

	// The extern file is line-based; a value spanning lines would
	// corrupt it silently, so the basic payload refuses it by name.
	_, err = kubernetes.Render(img, kubernetes.Site{
		Externs: map[string]string{"siloKey": "-----BEGIN KEY-----\nabc"},
	}, kubernetes.Profile{})
	if err == nil || !strings.Contains(err.Error(), "siloKey") {
		t.Errorf("a multiline extern value rendered; err = %v, want a refusal naming it", err)
	}
}

// TestRenderProfile pins the profile seam end to end: a house shape
// replaces the payload (a site fragment plus an unmounted enable
// key), the container (its own image, an enable env fed from the
// ConfigMap key, probes, a port), and the object dressing (namespace,
// fleet labels) — all without touching the renderer's own duties
// (checksum, custody split, stanza resources, metadata annotations).
func TestRenderProfile(t *testing.T) {
	prof := kubernetes.Profile{
		Namespace: "milling-prod",
		Labels:    map[string]string{"environment": "production"},
		Payload: func(in kubernetes.Instance) (kubernetes.Payload, error) {
			var conf, secret []string
			for _, b := range in.Externs {
				line := "extern." + b.Name + " = " + b.Value
				if b.Sensitive {
					secret = append(secret, line)
				} else {
					conf = append(conf, line)
				}
			}
			if len(conf) == 0 {
				conf = []string{"# nothing plain"}
			}
			p := kubernetes.Payload{
				Config: map[string]string{"site.conf": strings.Join(conf, "\n") + "\n"},
				Extra:  map[string]string{in.Record.Name: "true"},
			}
			if len(secret) > 0 {
				p.Secret = map[string]string{"secret.conf": strings.Join(secret, "\n") + "\n"}
			}
			return p, nil
		},
		Container: func(in kubernetes.Instance) (kubernetes.Container, error) {
			return kubernetes.Container{
				Image: "registry.example/fleet:v3",
				Args:  []string{kubernetes.ConfigMount + "/site.conf"},
				Env:   []kubernetes.EnvVar{{Name: "ENABLE", ConfigMapKey: in.Record.Name}},
				Ports: []kubernetes.Port{{Name: "health", ContainerPort: 3000}},
				Readiness: &kubernetes.HTTPProbe{
					Path: "/readyz", Port: "health", PeriodSeconds: 5, FailureThreshold: 3,
				},
			}, nil
		},
	}
	files, err := kubernetes.Render(testImage(), millSite(), prof)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	grinder := string(files["grinder.yaml"])
	for _, wants := range []string{
		"namespace: milling-prod",
		"environment: \"production\"",
		"app.kubernetes.io/part-of: \"milling\"",
		"Grinder: \"true\"",
		"site.conf: |",
		"secret.conf: |",
		"extern.siloKey = hunter2",
		"image: registry.example/fleet:v3",
		"key: Grinder",
		"path: /readyz",
		"containerPort: 3000",
		"cpu: \"300m\"",
		"solution.metadata/team: \"ops\"",
		"checksum/config: ",
	} {
		if !strings.Contains(grinder, wants) {
			t.Errorf("profile render misses %q", wants)
		}
	}
	if strings.Contains(grinder, "image.json") {
		t.Error("the house payload still carries the basic contract's shard file")
	}
}

// TestRenderedShardEnacts is the wet proof behind the whole contract:
// a shard rendered for one pod plans and runs on the stock host,
// against the live example catalogue, exactly as the pod would run it
// — the stand-in provision attaches, its output feeds the deploy, and
// the run completes. Completion is itself the isolation pin: the
// corpus image also deploys a forever-runner whose absence from this
// shard is the only reason Run returns.
func TestRenderedShardEnacts(t *testing.T) {
	cat := []solution.Package{
		{Path: "github.com/modern-engineering/prototype/examples/ff", Name: "ff", Elements: []solution.Element{
			solution.App("Ping", ff.Ping),
		}},
		{Path: "github.com/modern-engineering/prototype/examples/substrate", Name: "substrate", Elements: []solution.Element{
			solution.Provision("StandIn", substrate.StandIn),
			solution.Symbol("Endpoint", substrate.Endpoint),
		}},
	}
	img := &image.Image{
		Format:     image.Format,
		Solution:   "wetproof",
		Generation: 1,
		Catalogue: []image.Package{
			{Path: "github.com/modern-engineering/prototype/examples/ff", Name: "ff", Elements: []image.ElementSchema{
				{Name: "Ping", Kind: image.KindComponent, Params: []image.ParamSchema{
					{Name: "count", Default: "-1"}, {Name: "interval", Default: "1s"}, {Name: "nats"}, {Name: "target"},
				}},
			}},
			{Path: "github.com/modern-engineering/prototype/examples/substrate", Name: "substrate", Elements: []image.ElementSchema{
				{Name: "StandIn", Kind: image.KindProvision, Kinds: []string{image.KindAttach},
					Params:  []image.ParamSchema{{Name: "endpoint"}},
					Outputs: []image.OutputSchema{{Name: "config"}}},
				{Name: "Endpoint", Kind: image.KindSymbol},
			}},
		},
		Symbols: []image.SymbolDef{
			{Name: "natsEndpoint", Class: image.ClassExtern, Type: &image.Ref{Package: "github.com/modern-engineering/prototype/examples/substrate", Name: "Endpoint"}},
		},
		Records: []image.Record{
			{Verb: image.VerbProvision, Kind: image.KindAttach, Name: "natsStandIn",
				Element: image.Ref{Package: "github.com/modern-engineering/prototype/examples/substrate", Name: "StandIn"},
				Params: []image.Binding{
					{Key: "endpoint", Ref: &image.SymbolRef{Symbol: "natsEndpoint"}, Source: image.SourceInstance},
				}},
			{Verb: image.VerbDeploy, Name: "Pinger",
				Element: image.Ref{Package: "github.com/modern-engineering/prototype/examples/ff", Name: "Ping"},
				Params: []image.Binding{
					{Key: "count", Value: image.Int(2), Source: image.SourceInstance},
					{Key: "interval", Value: image.Duration(5 * time.Millisecond), Source: image.SourceInstance},
					{Key: "nats", Ref: &image.SymbolRef{Symbol: "natsStandIn", Output: "config"}, Source: image.SourceInstance},
					{Key: "target", Value: image.String("pong"), Source: image.SourceInstance},
				}},
			{Verb: image.VerbDeploy, Name: "Forever",
				Element: image.Ref{Package: "github.com/modern-engineering/prototype/examples/ff", Name: "Ping"},
				Params: []image.Binding{
					{Key: "target", Value: image.String("void"), Source: image.SourceInstance},
				}},
		},
	}
	img.Canonicalize()

	shard, err := kubernetes.Shard(img, "Pinger", nil)
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var audit bytes.Buffer
	err = host.Run(ctx, host.Config{
		Image:     shard,
		Catalogue: cat,
		Externs:   map[string]string{"natsEndpoint": "nats://stand.in:4222"},
		Log:       &audit,
	})
	if err != nil {
		t.Fatalf("host.Run(shard): %v\naudit:\n%s", err, audit.String())
	}
	for _, wants := range []string{"provision natsStandIn", "deploy Pinger"} {
		if !strings.Contains(audit.String(), wants) {
			t.Errorf("audit misses %q:\n%s", wants, audit.String())
		}
	}
}
