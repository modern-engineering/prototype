// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution/image"
)

// A roundTripCase names one vertical of the echo loop: the solution
// that seeds it, the shape the original image must carry, and the
// spellings echo must render back.
type roundTripCase struct {
	name    string
	example string // repo-relative solution dir; empty compiles unit instead
	unit    string // inline unit source for a fresh consumer module

	checkImage   func(t *testing.T, img *image.Image) // the original image's vertical-specific shape
	wantEchoed   []string                             // substrings the echoed unit must render
	checkRebuilt func(t *testing.T, img *image.Image) // the rebuilt image's provenance moves
}

// The image survives being made visible, whatever the vertical wired
// into it: building a solution, echoing its image into a fresh
// consumer module, and rebuilding the echoed unit yields the same
// desired state — image.Equal — while the bytes must differ, because
// the governance block digests the echoed unit (and folded defaults
// move their Sources), which is exactly the provenance Equal masks.
// The echoed unit itself is canonical: sdl fmt has nothing to say
// about it. Each case seeds the loop with one vertical; the verticals'
// compile semantics are solution's pins, the loop closure is this
// suite's.
func TestEchoRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	cases := []roundTripCase{
		{
			// The public example, the M1 deployable shape.
			name:    "Pingpong",
			example: "examples/pingpong",
		},
		{
			// Both symbol classes: the extern's sensitivity taints the
			// binding that references it, and echo renders declarations
			// and bare-identifier references back.
			name: "Symbols",
			unit: symbolsUnit,
			checkImage: func(t *testing.T, img *image.Image) {
				if len(img.Symbols) != 2 ||
					img.Symbols[0].Name != "apiToken" || img.Symbols[0].Class != image.ClassExtern ||
					img.Symbols[1].Name != "pongSubject" || img.Symbols[1].Class != image.ClassVar {
					t.Errorf("symbol table = %+v, want extern apiToken then var pongSubject", img.Symbols)
				}
				if len(img.Records) != 2 || len(img.Records[0].Params) != 2 {
					t.Fatalf("records = %+v, want Ping1 (2 params) and Pong1", img.Records)
				}
				target := img.Records[0].Params[1]
				if target.Key != "target" || target.Ref == nil || target.Ref.Symbol != "apiToken" || !target.Sensitive {
					t.Errorf("Ping1 target = %+v, want a sensitive ref to apiToken", target)
				}
			},
			wantEchoed: []string{
				"extern apiToken substrate.Secret",
				`var pongSubject: "ping"`,
				"target: apiToken",
			},
		},
		{
			// Both provision kinds, with the slice's sensitive output
			// referenced by a deploy; echo routes references through the
			// registered package name, not the unit's original alias.
			name: "Provisions",
			unit: provisionsUnit,
			checkImage: func(t *testing.T, img *image.Image) {
				if len(img.Records) != 3 ||
					img.Records[0].Verb != image.VerbProvision || img.Records[0].Kind != image.KindSlice || img.Records[0].Name != "natsAccount" ||
					img.Records[1].Verb != image.VerbProvision || img.Records[1].Kind != image.KindAttach || img.Records[1].Name != "pgLegacy" ||
					img.Records[2].Verb != image.VerbDeploy || img.Records[2].Kind != "" {
					t.Fatalf("records = %+v, want slice natsAccount, attach pgLegacy, deploy Ping1", img.Records)
				}
				target := img.Records[2].Params[1]
				if target.Key != "target" || target.Ref == nil ||
					target.Ref.Symbol != "natsAccount" || target.Ref.Output != "config" || !target.Sensitive {
					t.Errorf("Ping1 target = %+v, want a sensitive ref to natsAccount.config", target)
				}
			},
			wantEchoed: []string{
				"provision substrate.NATS slice as natsAccount {",
				"provision substrate.Postgres attach as pgLegacy {",
				"target: natsAccount.config",
			},
		},
		{
			// The defaults-bearing trip the design gate demanded: a type
			// default folds into a record that never wrote the parameter,
			// echo renders the folded binding explicitly, and the rebuilt
			// binding's Source moves to instance — the one field Equal
			// masks here, so the trip still closes.
			name: "Defaults",
			unit: defaultsUnit,
			checkImage: func(t *testing.T, img *image.Image) {
				if len(img.Records) != 1 || len(img.Records[0].Params) != 2 {
					t.Fatalf("records = %+v, want Ping1 with count and target", img.Records)
				}
				count := img.Records[0].Params[0]
				if count.Key != "count" || count.Value == nil || count.Value.Int != -1 || count.Source != image.SourceDefaultType {
					t.Fatalf("count binding = %+v, want -1 from %q", count, image.SourceDefaultType)
				}
			},
			wantEchoed: []string{"count: -1"},
			checkRebuilt: func(t *testing.T, img *image.Image) {
				if got := img.Records[0].Params[0].Source; got != image.SourceInstance {
					t.Errorf("rebuilt count Source = %q, want %q (echo folds defaults into instance text)", got, image.SourceInstance)
				}
			},
		},
		{
			// The opaque compartments: the verb default's field folds
			// into every deploy (tokens staying opaque) with the
			// instance's own field winning its key, with-stanzas merge
			// per qualifier, and metadata rides along.
			name: "Compartments",
			unit: compartmentsUnit,
			checkImage: func(t *testing.T, img *image.Image) {
				if len(img.Records) != 2 {
					t.Fatalf("records = %+v, want Ping1 and Ping2", img.Records)
				}
				one, two := img.Records[0], img.Records[1]
				if len(one.Deployment) != 1 || one.Deployment[0].Key != "location" ||
					one.Deployment[0].Value == nil || one.Deployment[0].Value.Kind != image.KindToken ||
					one.Deployment[0].Value.Tok != "awsUsEast1" || one.Deployment[0].Source != image.SourceDefaultDeploy {
					t.Errorf("Ping1 deployment = %+v, want the folded default-deploy token awsUsEast1", one.Deployment)
				}
				if len(two.Deployment) != 1 || two.Deployment[0].Value == nil || two.Deployment[0].Value.Tok != "euCentral1" ||
					two.Deployment[0].Source != image.SourceInstance {
					t.Errorf("Ping2 deployment = %+v, want the instance token euCentral1", two.Deployment)
				}
				pod := one.Extensions["k8s.pod"]
				if len(one.Extensions) != 1 || len(pod) != 1 || pod[0].Key != "priorityClass" ||
					pod[0].Value == nil || pod[0].Value.Tok != "standard" || pod[0].Source != image.SourceDefaultDeploy {
					t.Errorf("Ping1 extensions = %+v, want the folded default-deploy k8s.pod stanza", one.Extensions)
				}
				pod = two.Extensions["k8s.pod"]
				if len(two.Extensions) != 1 || len(pod) != 2 ||
					pod[0].Key != "priorityClass" || pod[0].Source != image.SourceDefaultDeploy ||
					pod[1].Key != "replicas" || pod[1].Value == nil || pod[1].Value.Int != 3 ||
					pod[1].Source != image.SourceInstance {
					t.Errorf("Ping2 extensions = %+v, want k8s.pod = {priorityClass: standard, replicas: 3}", two.Extensions)
				}
				if len(two.Metadata) != 1 || two.Metadata[0].Key != "team" ||
					two.Metadata[0].Value == nil || two.Metadata[0].Value.Str != "search" {
					t.Errorf("Ping2 metadata = %+v, want team: search", two.Metadata)
				}
			},
			wantEchoed: []string{
				"location: awsUsEast1",
				"location: euCentral1",
				"with k8s.pod {",
				"priorityClass: standard",
				"replicas: 3",
				`team: "search"`,
			},
		},
		{
			// The living design sample: every concept at once, its
			// decoded shape pinned by TestBuildSample.
			name:    "Sample",
			example: "examples/sample",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { roundTrip(t, tc) })
	}
}

// roundTrip drives one case around the loop: build, decode, echo,
// prove the echoed unit canonical, rebuild it in a fresh module, and
// close with Equal-but-different-bytes.
func roundTrip(t *testing.T, tc roundTripCase) {
	dir := repoRoot
	outA := filepath.Join(t.TempDir(), "a.json")
	args := []string{"build", "-o", outA}
	if tc.example != "" {
		args = append(args, tc.example)
	} else {
		dir = solutionModule(t, "trip.sdl", tc.unit)
	}
	res := runSDL(t, dir, args...)
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	bytesA, err := os.ReadFile(outA)
	if err != nil {
		t.Fatal(err)
	}
	imgA, err := image.Decode(bytes.NewReader(bytesA))
	if err != nil {
		t.Fatal(err)
	}
	if tc.checkImage != nil {
		tc.checkImage(t, imgA)
	}

	res = runSDL(t, dir, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)
	for _, want := range tc.wantEchoed {
		if !strings.Contains(echoed, want) {
			t.Errorf("echoed unit is missing %q", want)
		}
	}

	dir2 := solutionModule(t, "echoed.sdl", echoed)
	if res := runSDL(t, dir2, "fmt", "-l", "."); res.code != 0 || res.stdout != "" {
		t.Errorf("echoed unit is not canonical: fmt -l exited %d, listed %q\n%s",
			res.code, res.stdout, res.stderr)
	}
	outB := filepath.Join(dir2, "b.json")
	res = runSDL(t, dir2, "build", "-o", outB)
	if res.code != 0 {
		t.Fatalf("sdl build of the echoed unit exited %d\n%s", res.code, res.stderr)
	}
	bytesB, err := os.ReadFile(outB)
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := image.Decode(bytes.NewReader(bytesB))
	if err != nil {
		t.Fatal(err)
	}
	if tc.checkRebuilt != nil {
		tc.checkRebuilt(t, imgB)
	}
	if !image.Equal(imgA, imgB) {
		t.Errorf("round-tripped image is not Equal to the original\n--- rebuilt ---\n%s", bytesB)
	}
	if bytes.Equal(bytesA, bytesB) {
		t.Error("images are byte-identical; the governance block should have digested the echoed unit differently")
	}
}

// provisionsUnit wires both provision kinds against the substrate
// catalogue: a slice carving a NATS account out of site-bound
// substrate, an attachment onto a legacy server, and a deploy
// consuming the slice's sensitive output at reconcile time.
const provisionsUnit = `solution provisions

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	sub "github.com/modern-engineering/prototype/examples/substrate"
)

extern (
	natsCluster sub.NATSCluster
	natsAdmin sub.Secret
	pgServer sub.PostgresServer
)

provision sub.NATS slice as natsAccount {
	params {
		cluster: natsCluster
		adminAccount: natsAdmin
	}
}

provision sub.Postgres attach as pgLegacy {
	params {
		server: pgServer
	}
}

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: natsAccount.config
	}
}
`

// defaultsUnit leaves Ping's count to its type default, declared in
// the unit itself.
const defaultsUnit = `solution defaults

import ff "github.com/modern-engineering/prototype/examples/ff"

default ff.Ping {
	params {
		count: -1
	}
}

deploy ff.Ping as Ping1 {
	params {
		target: "pong"
	}
}
`

// compartmentsUnit is the mockup's compartment material: a verb
// default carrying deployment intent and an advisory stanza, an
// instance overriding the top-level field and extending the stanza,
// and opaque metadata riding along.
const compartmentsUnit = `solution compartments

import ff "github.com/modern-engineering/prototype/examples/ff"

default deploy {
	location: awsUsEast1

	with k8s.pod {
		priorityClass: standard
	}
}

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: "pong"
	}
}

deploy ff.Ping as Ping2 {
	location: euCentral1

	params {
		count: 2
		target: "pong"
	}

	with k8s.pod {
		replicas: 3
	}

	metadata {
		team: "search"
	}
}
`
