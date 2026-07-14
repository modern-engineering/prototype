// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
	"github.com/modern-engineering/prototype/solution/kubernetes"
)

// testImage composes the shard corpus by struct literal: two chained
// provisions feeding one deploy, a contrast deploy wired to nothing,
// and a symbol table with one spare of each class so selection has
// something to drop. Record order deliberately interleaves the verbs —
// statement order is meaning a shard must keep, not tidy.
func testImage() *image.Image {
	img := &image.Image{
		Format:     image.Format,
		Solution:   "shardcorpus",
		Generation: 7,
		Build: &image.Build{
			Units: []image.UnitDigest{{Name: "main.sdl", SHA256: strings.Repeat("ab", 32)}},
		},
		Catalogue: []image.Package{{
			Path: "example.test/mill",
			Name: "mill",
			Elements: []image.ElementSchema{
				{Name: "App", Kind: image.KindComponent, Params: []image.ParamSchema{
					{Name: "in"}, {Name: "mode"}, {Name: "spin"},
				}},
				{Name: "Feed", Kind: image.KindProvision, Kinds: []string{image.KindAttach},
					Params:  []image.ParamSchema{{Name: "from"}, {Name: "grade"}},
					Outputs: []image.OutputSchema{{Name: "chute"}, {Name: "dust", Sensitive: true}}},
				{Name: "Silo", Kind: image.KindSymbol, Sensitive: true},
				{Name: "Yard", Kind: image.KindSymbol},
			},
		}},
		Symbols: []image.SymbolDef{
			{Name: "siloKey", Class: image.ClassExtern, Type: &image.Ref{Package: "example.test/mill", Name: "Silo"}},
			{Name: "spareGate", Class: image.ClassExtern, Type: &image.Ref{Package: "example.test/mill", Name: "Yard"}},
			{Name: "grade", Class: image.ClassVar, Value: image.String("coarse")},
			{Name: "spin", Class: image.ClassVar, Value: image.Duration(1500 * time.Millisecond)},
			{Name: "spareKnob", Class: image.ClassVar, Value: image.Int(9)},
		},
		Records: []image.Record{
			{Verb: image.VerbProvision, Kind: image.KindAttach, Name: "intake",
				Element: image.Ref{Package: "example.test/mill", Name: "Feed"},
				Params: []image.Binding{
					{Key: "from", Ref: &image.SymbolRef{Symbol: "siloKey"}, Source: image.SourceInstance, Sensitive: true},
				}},
			{Verb: image.VerbDeploy, Name: "Idle",
				Element: image.Ref{Package: "example.test/mill", Name: "App"},
				Params: []image.Binding{
					{Key: "in", Value: image.String("nothing"), Source: image.SourceInstance},
				}},
			{Verb: image.VerbProvision, Kind: image.KindAttach, Name: "hopper",
				Element: image.Ref{Package: "example.test/mill", Name: "Feed"},
				Params: []image.Binding{
					{Key: "from", Ref: &image.SymbolRef{Symbol: "intake", Output: "chute"}, Source: image.SourceInstance},
					{Key: "grade", Ref: &image.SymbolRef{Symbol: "grade"}, Source: image.SourceInstance},
				}},
			{Verb: image.VerbDeploy, Name: "Grinder",
				Element: image.Ref{Package: "example.test/mill", Name: "App"},
				Params: []image.Binding{
					{Key: "in", Ref: &image.SymbolRef{Symbol: "hopper", Output: "chute"}, Source: image.SourceInstance},
					{Key: "spin", Ref: &image.SymbolRef{Symbol: "spin"}, Source: image.SourceInstance},
				},
				Deployment: []image.Binding{{Key: "location", Value: image.Token("west"), Source: image.SourceInstance}},
				Extensions: map[string][]image.Binding{
					"k8s.pod": {
						{Key: "cpu", Value: image.String("300m"), Source: image.SourceInstance},
						{Key: "memory", Value: image.String("250Mi"), Source: image.SourceInstance},
					},
					"k8s.workload": {{Key: "partOf", Value: image.String("milling"), Source: image.SourceInstance}},
					"acme.mesh":    {{Key: "lane", Value: image.Token("gold"), Source: image.SourceInstance}},
				},
				Metadata: []image.Binding{{Key: "team", Value: image.String("ops"), Source: image.SourceInstance}},
			},
		},
	}
	img.Canonicalize()
	return img
}

// names lists a shard's record names in order.
func names(img *image.Image) []string {
	var out []string
	for _, r := range img.Records {
		out = append(out, r.Name)
	}
	return out
}

// TestShardClosure pins the selection: a shard carries its record,
// every provision the record's references reach transitively, and the
// symbols that closure touches — in image record order — while
// records and symbols outside the closure are gone. The full pinned
// catalogue and the provenance block ride verbatim: schemas are the
// shard's dictionary, not its state.
func TestShardClosure(t *testing.T) {
	img := testImage()
	sub, err := kubernetes.Shard(img, "Grinder", nil)
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}

	got := names(sub)
	want := []string{"intake", "hopper", "Grinder"}
	if len(got) != len(want) {
		t.Fatalf("shard records = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shard records = %v, want %v", got, want)
		}
	}

	var symbols []string
	for _, def := range sub.Symbols {
		symbols = append(symbols, def.Name)
	}
	if len(symbols) != 3 || symbols[0] != "grade" || symbols[1] != "siloKey" || symbols[2] != "spin" {
		t.Fatalf("shard symbols = %v, want [grade siloKey spin]", symbols)
	}

	if len(sub.Catalogue) != len(img.Catalogue) {
		t.Fatalf("shard pins %d catalogue packages, want the image's %d", len(sub.Catalogue), len(img.Catalogue))
	}
	if sub.Generation != img.Generation || sub.Build == nil || sub.Solution != img.Solution {
		t.Fatalf("shard header = %s/%d build=%v, want the image's verbatim", sub.Solution, sub.Generation, sub.Build)
	}
}

// TestShardAdvisoryCompartments pins that deployment intent, extension
// stanzas, and metadata ride the shard untouched: transport must not
// consume what only controllers read, and the host that enacts the
// shard keeps ignoring them.
func TestShardAdvisoryCompartments(t *testing.T) {
	sub, err := kubernetes.Shard(testImage(), "Grinder", nil)
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}
	rec := sub.Records[len(sub.Records)-1]
	if len(rec.Deployment) != 1 || rec.Deployment[0].Key != "location" {
		t.Fatalf("deployment compartment = %v, want the location intent", rec.Deployment)
	}
	if len(rec.Extensions) != 3 || rec.Extensions["acme.mesh"] == nil || rec.Extensions["k8s.pod"] == nil {
		t.Fatalf("extensions = %v, want all three stanzas riding", rec.Extensions)
	}
	if len(rec.Metadata) != 1 || rec.Metadata[0].Key != "team" {
		t.Fatalf("metadata = %v, want the carried-through team", rec.Metadata)
	}
}

// TestShardRebind pins the var seam: a reached var rebinds to the
// site's value parsed under its default's kind, a rebind for a var
// outside the closure is ignored, externs never rebind through the
// map, and a value the kind cannot parse is refused naming the var.
func TestShardRebind(t *testing.T) {
	img := testImage()

	sub, err := kubernetes.Shard(img, "Grinder", map[string]string{
		"grade":     "fine",
		"spin":      "2s",
		"spareKnob": "11",   // outside the closure: ignored
		"siloKey":   "shhh", // an extern: never rebound here
	})
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}
	byName := make(map[string]image.SymbolDef)
	for _, def := range sub.Symbols {
		byName[def.Name] = def
	}
	if got := byName["grade"].Value; got == nil || got.Kind != image.KindString || got.Str != "fine" {
		t.Fatalf("grade rebound to %+v, want string fine", got)
	}
	if got := byName["spin"].Value; got == nil || got.Kind != image.KindDuration || got.Dur != 2*time.Second {
		t.Fatalf("spin rebound to %+v, want duration 2s", got)
	}
	if def := byName["siloKey"]; def.Value != nil {
		t.Fatalf("extern siloKey carries value %+v; externs stay late-bound", def.Value)
	}

	if _, err := kubernetes.Shard(img, "Grinder", map[string]string{"spin": "fast"}); err == nil {
		t.Fatal("rebinding a duration var to a non-duration succeeded; want a refusal naming the var")
	} else if !strings.Contains(err.Error(), "spin") {
		t.Fatalf("refusal %q does not name the var", err)
	}
}

// TestShardDeterminism pins that sharding is a pure derivation: the
// same image sharded the same way twice encodes to identical bytes,
// the property a drift-gated committed artifact set stands on.
func TestShardDeterminism(t *testing.T) {
	var a, b bytes.Buffer
	one, err := kubernetes.Shard(testImage(), "Grinder", map[string]string{"grade": "fine"})
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}
	two, err := kubernetes.Shard(testImage(), "Grinder", map[string]string{"grade": "fine"})
	if err != nil {
		t.Fatalf("Shard: %v", err)
	}
	if err := one.Encode(&a); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := two.Encode(&b); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("two identical shards encode differently")
	}
}

// TestShardRefusals pins the boundary: only deploy records shard —
// a provision instance and an unknown instance are refused by name —
// and an image that fails its own validation is refused before any
// walking, with the image's fault relayed rather than rephrased.
func TestShardRefusals(t *testing.T) {
	img := testImage()

	if _, err := kubernetes.Shard(img, "hopper", nil); err == nil || !strings.Contains(err.Error(), "hopper") {
		t.Fatalf("sharding a provision record: err = %v, want a refusal naming hopper", err)
	}
	if _, err := kubernetes.Shard(img, "nobody", nil); err == nil || !strings.Contains(err.Error(), "nobody") {
		t.Fatalf("sharding an unknown instance: err = %v, want a refusal naming nobody", err)
	}

	bad := testImage()
	bad.Solution = ""
	if _, err := kubernetes.Shard(bad, "Grinder", nil); err == nil {
		t.Fatal("sharding an invalid image succeeded; want the validation fault relayed")
	}
}
