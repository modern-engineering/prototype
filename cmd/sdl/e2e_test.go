// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// End-to-end proof of the toolchain: these tests build the sdl CLI
// once, then drive it as a user would — real solution directories, real
// module contexts, real toolchain runs, images echoed back into fresh
// solutions. They are the slow path; -short skips them all.
package main_test

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

var update = flag.Bool("update", false, "rewrite golden files from the observed output")

var (
	sdlPath  string // the CLI binary under test
	repoRoot string // the prototype module root
)

// locateRepo resolves the prototype module root: the anchor the built
// artifact compiles from and the scripts' consumer modules dir-replace
// against. TestMain lives in script_test.go and resolves it for every
// run.
func locateRepo() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", fmt.Errorf("locating module root: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// buildCLI compiles the sdl command once for the whole suite, from the
// module root so the test does not care which package directory the
// harness runs it in.
func buildCLI() error {
	dir, err := os.MkdirTemp("", "sdl-e2e-")
	if err != nil {
		return err
	}
	sdlPath = filepath.Join(dir, "sdl")
	cmd := exec.Command("go", "build", "-o", sdlPath, "./cmd/sdl")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("building sdl: %v\n%s", err, out)
	}
	return nil
}

// A result is one CLI invocation's observable outcome.
type result struct {
	code   int
	stdout string
	stderr string
}

// runSDL invokes the built CLI in dir.
func runSDL(t *testing.T, dir string, args ...string) result {
	t.Helper()
	cmd := exec.Command(sdlPath, args...)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("sdl %s: %v", strings.Join(args, " "), err)
		}
		code = exit.ExitCode()
	}
	return result{code: code, stdout: outb.String(), stderr: errb.String()}
}

// TestBuildPingpong is (a): the public example compiles to exactly the
// golden image, and the bytes decode as a well-formed image carrying
// the M1 deployable shape — a stand-in provision record whose typed,
// non-sensitive output feeds the deploys, the extern/var symbol pair,
// and the governance block digesting both units.
func TestBuildPingpong(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	out := filepath.Join(t.TempDir(), "out.json")

	start := time.Now()
	res := runSDL(t, repoRoot, "build", "-o", out, "examples/pingpong")
	t.Logf("cold sdl build: %v", time.Since(start))
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "pingpong.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o666); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("image differs from %s\n--- got ---\n%s", golden, got)
	}

	img, err := image.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("emitted image does not decode: %v", err)
	}
	if img.Solution != "pingpong" || len(img.Records) != 4 || len(img.Catalogue) != 2 || len(img.Symbols) != 2 {
		t.Errorf("decoded image: solution %q, %d records, %d packages, %d symbols; want pingpong, 4, 2, 2",
			img.Solution, len(img.Records), len(img.Catalogue), len(img.Symbols))
	}

	// The provision record: an attach of the substrate stand-in,
	// second in image order (extra.sdl's Pong sorts first).
	prov := img.Records[1]
	if prov.Verb != image.VerbProvision || prov.Kind != image.KindAttach || prov.Name != "natsStandIn" {
		t.Errorf("records[1] = %s %s %s, want provision attach natsStandIn", prov.Verb, prov.Kind, prov.Name)
	}

	// The typed output: StandIn's catalogue pin declares config as an
	// explicit, non-sensitive string — the demo shows the value flow.
	var standIn *image.ElementSchema
	for _, pkg := range img.Catalogue {
		for i, el := range pkg.Elements {
			if el.Name == "StandIn" {
				standIn = &pkg.Elements[i]
			}
		}
	}
	if standIn == nil {
		t.Fatalf("catalogue pins no StandIn element: %+v", img.Catalogue)
	}
	if len(standIn.Outputs) != 1 || standIn.Outputs[0].Name != "config" ||
		standIn.Outputs[0].Type != "string" || standIn.Outputs[0].Sensitive {
		t.Errorf("StandIn outputs = %+v, want one non-sensitive string config", standIn.Outputs)
	}

	// The wiring: Ping1's nats param references the stand-in's output,
	// untainted.
	var nats *image.Binding
	for i, b := range img.Records[2].Params {
		if b.Key == "nats" {
			nats = &img.Records[2].Params[i]
		}
	}
	if nats == nil || nats.Ref == nil || nats.Ref.Symbol != "natsStandIn" || nats.Ref.Output != "config" || nats.Sensitive {
		t.Errorf("Ping1 nats = %+v, want an untainted ref to natsStandIn.config", nats)
	}

	// The governance block digests both units; a dir-replaced checkout
	// build records no version settings.
	if img.Build == nil || len(img.Build.Units) != 2 ||
		img.Build.Units[0].Name != "extra.sdl" || img.Build.Units[1].Name != "pingpong.sdl" {
		t.Errorf("governance block = %+v, want extra.sdl and pingpong.sdl digests", img.Build)
	}
}

// TestBuildDeterminism is (b): two builds of the same solution emit
// byte-identical images. The timings here are the warm path — the cold
// one is TestBuildPingpong's log line.
func TestBuildDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := t.TempDir()
	var images [2][]byte
	for i := range images {
		out := filepath.Join(dir, fmt.Sprintf("out%d.json", i))
		start := time.Now()
		res := runSDL(t, repoRoot, "build", "-o", out, "examples/pingpong")
		t.Logf("warm sdl build %d: %v", i+1, time.Since(start))
		if res.code != 0 {
			t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		images[i] = data
	}
	if !bytes.Equal(images[0], images[1]) {
		t.Error("two builds of the same solution emitted different bytes")
	}
}

// TestBuildStdout pins the default output mode: without -o the image
// goes to standard output, byte-identical to what -o would have
// written, with nothing else mixed into either stream.
func TestBuildStdout(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	res := runSDL(t, repoRoot, "build", "examples/pingpong")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "pingpong.json"))
	if err != nil {
		t.Fatal(err)
	}
	if res.stdout != string(want) {
		t.Errorf("stdout differs from the golden image\n--- got ---\n%s", res.stdout)
	}
}

// TestBuildGeneration pins the -generation flag: a positive value
// stamps the image, and a non-positive one is a usage fault (exit 2)
// before any toolchain work starts.
func TestBuildGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	out := filepath.Join(t.TempDir(), "out.json")
	res := runSDL(t, repoRoot, "build", "-generation", "7", "-o", out, "examples/pingpong")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	img, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("emitted image does not decode: %v", err)
	}
	if img.Generation != 7 {
		t.Errorf("Generation = %d, want 7", img.Generation)
	}

	res = runSDL(t, repoRoot, "build", "-generation", "0", "examples/pingpong")
	if res.code != 2 {
		t.Fatalf("exit %d, want 2\n%s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "generation must be positive") {
		t.Errorf("stderr does not name the usage fault:\n%s", res.stderr)
	}
}

// solutionModule lays down a throwaway module holding one solution
// unit, with the prototype module dir-replaced to this checkout — the
// shape of a user module consuming the framework.
func solutionModule(t *testing.T, unitName, unitSource string) string {
	t.Helper()
	return solutionModuleFiles(t, map[string]string{unitName: unitSource})
}

// solutionModuleFiles is solutionModule for multi-unit solutions: one
// throwaway module holding every named unit.
func solutionModuleFiles(t *testing.T, units map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	gomod := fmt.Sprintf(`module example.test/sol

go 1.25.0

require github.com/modern-engineering/prototype v0.0.0

replace github.com/modern-engineering/prototype => %s
`, repoRoot)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o666); err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(repoRoot, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o666); err != nil {
		t.Fatal(err)
	}
	for name, source := range units {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestDiagnostics is (c): solution faults exit 1 with diagnostics
// positioned at the offending source, whether the fault dies before the
// toolchain (an unresolvable import) or deep inside the generated
// compiler (an unknown element).
func TestDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}

	t.Run("UnresolvableImport", func(t *testing.T) {
		dir := solutionModule(t, "broken.sdl", `solution broken

import miss "example.test/nonexistent"

deploy miss.Thing as T
`)
		res := runSDL(t, dir, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, `broken.sdl:3:8: import "example.test/nonexistent"`) {
			t.Errorf("diagnostic not positioned at the import spec:\n%s", res.stderr)
		}
	})

	t.Run("UnknownElement", func(t *testing.T) {
		dir := solutionModule(t, "sol.sdl", `solution broken

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Gone as G
`)
		res := runSDL(t, dir, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, "sol.sdl:5:8: unknown element Gone") {
			t.Errorf("diagnostic not positioned at the element reference:\n%s", res.stderr)
		}
	})

	t.Run("ClauselessUnit", func(t *testing.T) {
		// An empty peer unit is not well-formed: build diagnoses it
		// positioned at exit 1 (as fmt refuses it too), never as an
		// internal exit-2 fault.
		dir := solutionModuleFiles(t, map[string]string{
			"sol.sdl":   "solution broken\n",
			"empty.sdl": "// a stray comment-only unit\n",
		})
		res := runSDL(t, dir, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, "empty.sdl:1:1: unit declares no solution clause") {
			t.Errorf("diagnostic not positioned at the clause-less unit:\n%s", res.stderr)
		}
	})
}

// TestPerFileImportScope proves import scope end to end: units bind
// aliases independently — one alias may name different packages in
// different units, one package may go by different aliases — while the
// driver still generates one import per package path; and a unit
// referencing a package only a peer imports is a positioned fault.
func TestPerFileImportScope(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}

	t.Run("AliasesAreUnitLocal", func(t *testing.T) {
		dir := solutionModuleFiles(t, map[string]string{
			"a.sdl": `solution scoped

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: "pong"
	}
}
`,
			// The same alias names another package here, and pp names
			// the package a.sdl calls ff — both Go-legal, both local.
			"b.sdl": `solution scoped

import (
	ff "github.com/modern-engineering/prototype/examples/substrate"
	pp "github.com/modern-engineering/prototype/examples/ff"
)

provision ff.Postgres attach as legacy

deploy pp.Pong as Pong1 {
	params {
		subject: "ping"
	}
}
`,
		})
		out := filepath.Join(dir, "out.json")
		res := runSDL(t, dir, "build", "-o", out)
		if res.code != 0 {
			t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		img, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("emitted image does not decode: %v", err)
		}
		want := []image.Ref{
			{Package: "github.com/modern-engineering/prototype/examples/ff", Name: "Ping"},
			{Package: "github.com/modern-engineering/prototype/examples/substrate", Name: "Postgres"},
			{Package: "github.com/modern-engineering/prototype/examples/ff", Name: "Pong"},
		}
		if len(img.Records) != len(want) {
			t.Fatalf("records = %+v, want %d records", img.Records, len(want))
		}
		for i, ref := range want {
			if img.Records[i].Element != ref {
				t.Errorf("record %d element = %+v, want %+v", i, img.Records[i].Element, ref)
			}
		}
		// Each package pins once, however many aliases name it.
		if len(img.Catalogue) != 2 {
			t.Errorf("catalogue pins %d packages, want 2 (one per import path)", len(img.Catalogue))
		}
	})

	t.Run("ForgottenImport", func(t *testing.T) {
		dir := solutionModuleFiles(t, map[string]string{
			"a.sdl": `solution scoped

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	sub "github.com/modern-engineering/prototype/examples/substrate"
)

deploy ff.Ping as Ping1
`,
			// sub resolves only through a.sdl's imports; this unit
			// never imports it.
			"b.sdl": `solution scoped

import ff "github.com/modern-engineering/prototype/examples/ff"

extern key sub.Secret

deploy ff.Pong as Pong1 {
	params {
		subject: key
	}
}
`,
		})
		res := runSDL(t, dir, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, "b.sdl:5:12: package sub is not imported") {
			t.Errorf("diagnostic not positioned at the peer-only reference:\n%s", res.stderr)
		}
	})
}

// symbolsUnit wires both symbol classes across both example catalogue
// packages: an extern of a sensitive symbol type bound into one deploy
// and a var bound into another.
const symbolsUnit = `solution symbols

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/substrate"
)

extern apiToken substrate.Secret

var pongSubject: "ping"

deploy ff.Ping as Ping1 {
	params {
		count: 3
		target: apiToken
	}
}

deploy ff.Pong as Pong1 {
	params {
		subject: pongSubject
	}
}
`

// TestSymbolDiagnostics packs the negative symbol material into one
// unit — an extern of an unknown type, a var colliding with an
// instance name, a dangling reference, and a dotted output reference —
// and expects every fault positioned, exit 1.
func TestSymbolDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "broken.sdl", `solution broken

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	sub "github.com/modern-engineering/prototype/examples/substrate"
)

extern wrongType sub.Gone

var Ping1: "taken"

deploy ff.Ping as Ping1 {
	params {
		target: missing
		count: acct.config
	}
}
`)
	res := runSDL(t, dir, "build")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
	}
	for _, want := range []string{
		"broken.sdl:8:18: unknown element Gone",
		"broken.sdl:12:19: duplicate symbol Ping1 (first declared at broken.sdl:10:5)",
		"broken.sdl:14:11: undefined symbol missing",
		"broken.sdl:15:10: undefined symbol acct",
	} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
	}
}

// TestProvisionDiagnostics packs the negative provision material into
// one unit — an ambiguous omitted kind word, a kind the type does not
// register, a self-referencing slice, a two-instance output cycle, and
// dotted references to a missing symbol, a missing output, and a
// deploy instance — and expects every fault positioned, exit 1.
func TestProvisionDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "broken.sdl", `solution broken

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	sub "github.com/modern-engineering/prototype/examples/substrate"
)

provision sub.NATS as amb
provision sub.Postgres slice as wrongKind
provision sub.NATS slice as selfRef {
	params {
		adminAccount: selfRef.config
	}
}
provision sub.NATS slice as loopA {
	params {
		adminAccount: loopB.config
	}
}
provision sub.NATS slice as loopB {
	params {
		adminAccount: loopA.config
	}
}
deploy ff.Pong as Echo
deploy ff.Ping as P {
	params {
		target: natsMissing.config
		count: selfRef.nope
		interval: Echo.config
	}
}
`)
	res := runSDL(t, dir, "build")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
	}
	for _, want := range []string{
		"broken.sdl:8:11: missing provision kind: type sub.NATS registers both slice and attach",
		"broken.sdl:9:24: type sub.Postgres does not register slice",
		"broken.sdl:10:29: provision reference cycle: selfRef -> selfRef",
		"broken.sdl:15:29: provision reference cycle: loopA -> loopB -> loopA",
		"broken.sdl:28:11: undefined symbol natsMissing",
		"broken.sdl:29:18: unknown output nope: provision type substrate.NATS declares no such output",
		"broken.sdl:30:13: instance Echo has no outputs: only provision instances emit outputs",
	} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
	}
}

// TestFactoredForms proves factored spec blocks are pure notation: a
// unit written with factored deploy and provision blocks compiles to
// the same desired state as its single-form twin. The comparison is
// image.Equal, not bytes: the governance block digests each spelling,
// and the spelling is exactly what the two variants vary.
func TestFactoredForms(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	const imports = `
import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	sub "github.com/modern-engineering/prototype/examples/substrate"
)

extern (
	natsCluster sub.NATSCluster
	natsAdmin sub.Secret
)
`
	factored := "solution factored\n" + imports + `
provision (
	sub.NATS slice as natsAccount {
		params {
			cluster: natsCluster
			adminAccount: natsAdmin
		}
	}
	sub.Postgres attach as pgLegacy
)

deploy (
	ff.Ping as Ping1 {
		params {
			count: 1
			target: natsAccount.config
		}
	}
	ff.Pong as Pong1 {
		params {
			subject: "ping"
		}
	}
)
`
	single := "solution factored\n" + imports + `
provision sub.NATS slice as natsAccount {
	params {
		cluster: natsCluster
		adminAccount: natsAdmin
	}
}

provision sub.Postgres attach as pgLegacy

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: natsAccount.config
	}
}

deploy ff.Pong as Pong1 {
	params {
		subject: "ping"
	}
}
`
	var images [2]*image.Image
	for i, source := range []string{factored, single} {
		dir := solutionModule(t, "factored.sdl", source)
		out := filepath.Join(dir, "out.json")
		res := runSDL(t, dir, "build", "-o", out)
		if res.code != 0 {
			t.Fatalf("sdl build of variant %d exited %d\n%s", i, res.code, res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		img, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("image of variant %d does not decode: %v", i, err)
		}
		images[i] = img
	}
	if !image.Equal(images[0], images[1]) {
		t.Errorf("factored and single-form units compiled to different desired states\n--- factored ---\n%+v\n--- single ---\n%+v",
			images[0], images[1])
	}
}

// TestImagePlumbing smokes the sdl image command group over a freshly
// built image: edit amends the generation in place (atomically, the
// content otherwise untouched), and records and symbols print their
// aligned tables.
func TestImagePlumbing(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "symbols.sdl", symbolsUnit)
	out := filepath.Join(dir, "image.json")
	res := runSDL(t, dir, "build", "-o", out)
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}

	res = runSDL(t, dir, "image", "edit", "-generation", "5", out)
	if res.code != 0 {
		t.Fatalf("sdl image edit exited %d\n%s", res.code, res.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	img, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("edited image does not decode: %v", err)
	}
	if img.Generation != 5 {
		t.Errorf("Generation = %d, want 5", img.Generation)
	}

	res = runSDL(t, dir, "image", "records", out)
	if res.code != 0 {
		t.Fatalf("sdl image records exited %d\n%s", res.code, res.stderr)
	}
	t.Logf("records table:\n%s", res.stdout)
	for _, want := range []string{
		"VERB", "ELEMENT",
		"deploy",
		"github.com/modern-engineering/prototype/examples/ff.Ping",
		"Ping1",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("records table is missing %q:\n%s", want, res.stdout)
		}
	}

	res = runSDL(t, dir, "image", "symbols", out)
	if res.code != 0 {
		t.Fatalf("sdl image symbols exited %d\n%s", res.code, res.stderr)
	}
	t.Logf("symbols table:\n%s", res.stdout)
	for _, want := range []string{
		"NAME", "CLASS", "TYPE/VALUE",
		"apiToken", "extern",
		"pongSubject", "var", `"ping"`,
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("symbols table is missing %q:\n%s", want, res.stdout)
		}
	}
}

// TestBuildSample is the CP-C proof: the mockup-7 sample lives in
// examples/sample and compiles to exactly the golden image — extern
// and var symbols, both provision kinds, folded type and verb
// defaults, output references, compartments checked against the
// imported scheme package, and a peer unit — and the image plumbing
// reads the result back.
func TestBuildSample(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	out := filepath.Join(t.TempDir(), "sample.json")
	res := runSDL(t, repoRoot, "build", "-o", out, "examples/sample")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "sample.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o666); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("image differs from %s\n--- got ---\n%s", golden, got)
	}

	img, err := image.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("emitted image does not decode: %v", err)
	}
	if img.Solution != "sample" || len(img.Records) != 6 || len(img.Catalogue) != 3 || len(img.Symbols) != 5 {
		t.Errorf("decoded image: solution %q, %d records, %d packages, %d symbols; want sample, 6, 3, 5",
			img.Solution, len(img.Records), len(img.Catalogue), len(img.Symbols))
	}

	// The scheme pins: importing examples/k8s puts the package in the
	// catalogue with its self-declared qualifiers and key schemas — the
	// record every with k8s.* stanza in the solution was checked
	// against, discovered and emitted by the real generate stage.
	var k8sPkg *image.Package
	for i := range img.Catalogue {
		if img.Catalogue[i].Path == "github.com/modern-engineering/prototype/examples/k8s" {
			k8sPkg = &img.Catalogue[i]
		}
	}
	if k8sPkg == nil {
		t.Fatalf("catalogue pins no k8s package: %+v", img.Catalogue)
	}
	wantSchemes := []image.ElementSchema{
		{
			Name:      "Pod",
			Kind:      image.KindScheme,
			Doc:       "pod-level scheduling conventions",
			Qualifier: "k8s.pod",
			Params: []image.ParamSchema{
				{Name: "priorityClass", Usage: "scheduling priority class"},
				{Name: "replicas", Usage: "desired pod replicas", Default: "1"},
			},
		},
		{
			Name:      "Workload",
			Kind:      image.KindScheme,
			Doc:       "workload grouping conventions",
			Qualifier: "k8s.workload",
			Params: []image.ParamSchema{
				{Name: "partOf", Usage: "umbrella workload this instance joins"},
			},
		},
	}
	if !reflect.DeepEqual(k8sPkg.Elements, wantSchemes) {
		t.Errorf("k8s pins = %+v, want %+v", k8sPkg.Elements, wantSchemes)
	}

	// The opaque ride-along: no imported package claims acme.rollout,
	// so Ping3's stanza lands verbatim — beside the checked k8s.pod
	// one folded from the verb default, both halves of the advisory
	// contract on a single record.
	ping3 := img.Records[5]
	if ping3.Name != "Ping3" {
		t.Fatalf("records[5] = %s, want Ping3", ping3.Name)
	}
	rollout := ping3.Extensions["acme.rollout"]
	if len(ping3.Extensions) != 2 || len(rollout) != 2 ||
		rollout[0].Key != "maxSurge" || rollout[0].Value == nil || rollout[0].Value.Int != 1 ||
		rollout[1].Key != "strategy" || rollout[1].Value == nil || rollout[1].Value.Tok != "canary" {
		t.Errorf("Ping3 extensions = %+v, want the opaque acme.rollout stanza beside the checked k8s.pod one", ping3.Extensions)
	}

	res = runSDL(t, repoRoot, "image", "records", out)
	if res.code != 0 {
		t.Fatalf("sdl image records exited %d\n%s", res.code, res.stderr)
	}
	t.Logf("sample records table:\n%s", res.stdout)
	for _, want := range []string{
		"provision  slice   github.com/modern-engineering/prototype/examples/substrate.NATS",
		"provision  attach  github.com/modern-engineering/prototype/examples/substrate.Postgres",
		"natsAccount", "pgLegacy", "Ping1", "Ping2", "Pong", "Ping3",
		// The compartment columns surface deployment intent and the
		// extension qualifiers per record — the unclaimed qualifier
		// listed beside the checked one.
		"DEPLOYMENT", "EXTENSIONS",
		"location: euCentral1", "acme.rollout, k8s.pod",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("records table is missing %q:\n%s", want, res.stdout)
		}
	}

	// info reports the identity and the governance block: both unit
	// digests, no settings (nothing resolves to a real version in this
	// checkout), both catalogue pins.
	res = runSDL(t, repoRoot, "image", "info", out)
	if res.code != 0 {
		t.Fatalf("sdl image info exited %d\n%s", res.code, res.stderr)
	}
	t.Logf("sample info report:\n%s", res.stdout)
	for _, want := range []string{
		"solution\tsample\n",
		"generation\t1\n",
		"format\tsolution-image/1\n",
		"unit\tsample.sdl\t",
		"unit\tsample_extra.sdl\t",
		"catalogue\tgithub.com/modern-engineering/prototype/examples/ff\tff\t2 elements\n",
		"catalogue\tgithub.com/modern-engineering/prototype/examples/k8s\tk8s\t2 elements\n",
		"catalogue\tgithub.com/modern-engineering/prototype/examples/substrate\tsubstrate\t",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("info report is missing %q:\n%s", want, res.stdout)
		}
	}
	if strings.Contains(res.stdout, "setting\t") {
		t.Errorf("info reported a version setting; dir-replaced and checkout builds must record none:\n%s", res.stdout)
	}
}

// schemeStanza is the deploy statement both scheme subtests share: an
// int key bound to a string literal and a key the scheme never
// declared. Whether that text is two positioned faults or two plain
// bindings is decided by one thing only — whether any unit imports
// the package claiming the qualifier.
const schemeStanza = `
deploy ff.Ping as P {
	with k8s.pod {
		replicas: "three"
		unknownKey: 1
	}
}
`

// TestSchemeChecking drives the discovered-scheme contract end to end
// through the real toolchain: importing examples/k8s turns the shared
// stanza into compile faults with the linker's exact wordings, and
// dropping the import lets the identical text ride the image opaquely
// — same bytes, both advisory halves.
func TestSchemeChecking(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}

	t.Run("ImportedSchemeChecks", func(t *testing.T) {
		dir := solutionModule(t, "bad.sdl", `solution schemes

import (
	ff "github.com/modern-engineering/prototype/examples/ff"
	"github.com/modern-engineering/prototype/examples/k8s"
)
`+schemeStanza)
		res := runSDL(t, dir, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		for _, want := range []string{
			"bad.sdl:10:13: invalid value three for k8s.pod key replicas: parse error",
			"bad.sdl:11:3: unknown key unknownKey in with k8s.pod (scheme k8s.Pod)",
		} {
			if !strings.Contains(res.stderr, want) {
				t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
			}
		}
	})

	t.Run("UnimportedSchemeRidesOpaque", func(t *testing.T) {
		dir := solutionModule(t, "good.sdl", `solution schemes

import ff "github.com/modern-engineering/prototype/examples/ff"
`+schemeStanza)
		out := filepath.Join(dir, "out.json")
		res := runSDL(t, dir, "build", "-o", out)
		if res.code != 0 {
			t.Fatalf("exit %d, want 0\n%s", res.code, res.stderr)
		}
		if res.stderr != "" {
			t.Errorf("stderr = %q, want silence: unresolved qualifiers are tolerated, not warned about", res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		img, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("emitted image does not decode: %v", err)
		}
		if len(img.Records) != 1 {
			t.Fatalf("records = %+v, want the one P deploy", img.Records)
		}
		pod := img.Records[0].Extensions["k8s.pod"]
		if len(pod) != 2 ||
			pod[0].Key != "replicas" || pod[0].Value == nil || pod[0].Value.Str != "three" ||
			pod[1].Key != "unknownKey" || pod[1].Value == nil || pod[1].Value.Int != 1 {
			t.Errorf("extensions = %+v, want the stanza riding verbatim", img.Records[0].Extensions)
		}
	})
}

// TestFmtExamples is (f): the checked-in examples are canonical, so the
// repository holds the form the toolchain prints.
func TestFmtExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	res := runSDL(t, repoRoot, "fmt", "-l", "examples")
	if res.code != 0 || res.stdout != "" {
		t.Errorf("fmt -l exited %d and listed %q; the examples must stay canonical\n%s",
			res.code, res.stdout, res.stderr)
	}
}

// writeFile lays down one file of a test fixture.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		t.Fatal(err)
	}
}

// workspaceCatalogue renders the workspace catalogue module's source
// with one mutable flag default — the marker the test reads back out of
// the built image to prove which source the build resolved.
func workspaceCatalogue(greeting string) string {
	return fmt.Sprintf(`package catalogue

import (
	"context"
	"flag"

	"github.com/modern-engineering/prototype/application"
)

// Hello greets its audience.
var Hello = &application.Descriptor{
	Name: "hello",
	Doc:  "hello greets its audience",
	Make: application.MakeFunc(func() (application.Runner, *flag.FlagSet) {
		fs := flag.NewFlagSet("hello", flag.ContinueOnError)
		fs.String("greeting", %q, "what hello says")
		return application.RunnerFunc(func(context.Context) error { return nil }), fs
	}),
}
`, greeting)
}

// TestBuildWorkspace is the workspace-mode proof: a solution inside one
// module of an active go.work resolves its catalogue imports through
// the workspace's union of modules, exactly as the go CLI would — no
// replace or require directives anywhere, because a workspace resolves
// member imports without them (a member requiring another member still
// needs that version's go.mod fetchable for the module graph, so the
// require-less shape is also the only hermetic one). The catalogue
// module's path is served by no proxy, so only local member source can
// satisfy it; the prototype rides along as one more member, keeping
// the version-skew handshake silent. Editing the catalogue source
// between builds must reach the very next image: the workspace tracks
// member directories, never a fetched copy.
func TestBuildWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	ws := t.TempDir()
	modA := filepath.Join(ws, "modA") // holds the solution
	modB := filepath.Join(ws, "modB") // holds the catalogue
	for _, dir := range []string{modA, modB} {
		if err := os.Mkdir(dir, 0o777); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(modA, "go.mod"), "module example.test/sol\n\ngo 1.25.0\n")
	writeFile(t, filepath.Join(modA, "hello.sdl"), `solution wsdemo

import cat "example.test/catalogue"

deploy cat.Hello as Hello1 {
	params {
		greeting: "hi"
	}
}
`)
	writeFile(t, filepath.Join(modB, "go.mod"), "module example.test/catalogue\n\ngo 1.25.0\n")
	writeFile(t, filepath.Join(modB, "catalogue.go"), workspaceCatalogue("workspace-before"))
	writeFile(t, filepath.Join(ws, "go.work"), fmt.Sprintf("go 1.25.0\n\nuse (\n\t./modA\n\t./modB\n\t%s\n)\n", repoRoot))

	build := func() *image.Image {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.json")
		res := runSDL(t, modA, "build", "-o", out)
		if res.code != 0 {
			t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
		}
		if strings.Contains(res.stderr, "warning") {
			t.Errorf("skew warning despite the prototype being a workspace member:\n%s", res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		img, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("emitted image does not decode: %v", err)
		}
		return img
	}

	img := build()
	if len(img.Catalogue) != 1 || img.Catalogue[0].Path != "example.test/catalogue" {
		t.Fatalf("catalogue = %+v, want the workspace module example.test/catalogue", img.Catalogue)
	}
	if got := helloDefault(t, img); got != "workspace-before" {
		t.Errorf("greeting default = %q, want the member source's %q", got, "workspace-before")
	}
	if len(img.Records) != 1 || img.Records[0].Name != "Hello1" {
		t.Fatalf("records = %+v, want the one Hello1 deploy", img.Records)
	}

	writeFile(t, filepath.Join(modB, "catalogue.go"), workspaceCatalogue("workspace-after"))
	if got := helloDefault(t, build()); got != "workspace-after" {
		t.Errorf("greeting default after the edit = %q, want %q", got, "workspace-after")
	}
}

// helloDefault digs the greeting parameter's pinned default out of the
// image's catalogue schema.
func helloDefault(t *testing.T, img *image.Image) string {
	t.Helper()
	for _, el := range img.Catalogue[0].Elements {
		for _, p := range el.Params {
			if p.Name == "greeting" {
				return p.Default
			}
		}
	}
	t.Fatalf("no greeting parameter in catalogue %+v", img.Catalogue)
	return ""
}

// runSDLEnv invokes the built CLI in dir with extra environment
// variables appended to the inherited environment.
func runSDLEnv(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(sdlPath, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("sdl %s: %v", strings.Join(args, " "), err)
		}
		code = exit.ExitCode()
	}
	return result{code: code, stdout: outb.String(), stderr: errb.String()}
}

// writeProxy lays down a file:// module proxy serving this checkout as
// github.com/modern-engineering/prototype@v0.1.0 in the cmd/go proxy
// layout — @v/list, .info, .mod, and a module zip holding everything
// the generated compiler imports (application, solution, sdl, and the
// examples catalogue) plus the module's own metadata files.
func writeProxy(t *testing.T, dir string) {
	t.Helper()
	const version = "v0.1.0"
	vdir := filepath.Join(dir, "github.com", "modern-engineering", "prototype", "@v")
	if err := os.MkdirAll(vdir, 0o777); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(vdir, "list"), version+"\n")
	writeFile(t, filepath.Join(vdir, version+".info"), fmt.Sprintf("{\"Version\":%q}\n", version))
	gomod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(vdir, version+".mod"), string(gomod))

	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	add := func(rel string) {
		w, err := zw.Create("github.com/modern-engineering/prototype@" + version + "/" + filepath.ToSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"go.mod", "go.sum", "LICENSE"} {
		add(rel)
	}
	for _, tree := range []string{"application", "solution", "sdl", "examples"} {
		err := filepath.WalkDir(filepath.Join(repoRoot, tree), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				rel, err := filepath.Rel(repoRoot, p)
				if err != nil {
					return err
				}
				add(rel)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vdir, version+".zip"), zbuf.Bytes(), 0o666); err != nil {
		t.Fatal(err)
	}
}

// hermeticProxyEnv builds the module-resolution environment for the
// module-less tests: a file:// proxy serving this checkout at v0.1.0,
// falling back (on 404, the comma semantics) to the local module cache
// in its download layout for the x/ dependencies — which the cache
// holds whenever this suite runs at all, since building the CLI needs
// them — and a fresh GOMODCACHE so nothing real is poisoned by the
// fictional version. No route leaves the machine. The fresh cache is
// scrubbed with go clean -modcache before removal: the cache write-
// protects its directories, which os.RemoveAll alone cannot clear.
func hermeticProxyEnv(t *testing.T) []string {
	t.Helper()
	proxy := t.TempDir()
	writeProxy(t, proxy)
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatalf("locating the module cache: %v", err)
	}
	download := filepath.Join(strings.TrimSpace(string(out)), "cache", "download")

	cache := t.TempDir()
	t.Cleanup(func() {
		cmd := exec.Command("go", "clean", "-modcache")
		cmd.Env = append(os.Environ(), "GOMODCACHE="+cache)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("cleaning the test module cache: %v\n%s", err, out)
		}
	})
	return []string{
		"GOPROXY=file://" + filepath.ToSlash(proxy) + ",file://" + filepath.ToSlash(download),
		"GOMODCACHE=" + cache,
		"GOSUMDB=off",
		// Neutralize developer overrides that could route the fictional
		// module past the fixture.
		"GOPRIVATE=", "GONOPROXY=", "GONOSUMDB=", "GOFLAGS=",
	}
}

// TestBuildNoModule is the module-less proof: a solution directory
// outside any go.mod or go.work builds by resolving its imports at
// their latest versions through the ambient proxy configuration, the
// generated program's own go.mod carrying the requirements. The fixture
// proxy serves the prototype at the fictional v0.1.0, so the built CLI
// necessarily reports version skew — the handshake's evidence — and an
// unresolvable import still dies as a diagnostic positioned at its
// import spec, tidy's own cause on stderr beside it.
func TestBuildNoModule(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	env := hermeticProxyEnv(t)

	t.Run("Image", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "nomod.sdl"), `solution nomod

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: "pong"
	}
}
`)
		out := filepath.Join(dir, "out.json")
		start := time.Now()
		res := runSDLEnv(t, dir, env, "build", "-o", out)
		t.Logf("module-less sdl build: %v", time.Since(start))
		if res.code != 0 {
			t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, "sdl: warning: solution resolves github.com/modern-engineering/prototype v0.1.0") {
			t.Errorf("no skew warning against the fetched v0.1.0:\n%s", res.stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		img, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("emitted image does not decode: %v", err)
		}
		if len(img.Catalogue) != 1 || img.Catalogue[0].Path != "github.com/modern-engineering/prototype/examples/ff" {
			t.Errorf("catalogue = %+v, want the proxy-served ff package", img.Catalogue)
		}
		if len(img.Records) != 1 || img.Records[0].Name != "Ping1" {
			t.Errorf("records = %+v, want the one Ping1 deploy", img.Records)
		}
		// The proxy serves a real (if fictional) module version — the
		// one shape in this suite where the governance block records
		// the resolved prototype pin.
		want := image.Setting{Key: "prototype.version", Value: "v0.1.0"}
		if img.Build == nil || len(img.Build.Settings) != 1 || img.Build.Settings[0] != want {
			t.Errorf("build settings = %+v, want the one %v", img.Build, want)
		}
	})

	t.Run("UnresolvableImport", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "broken.sdl"), `solution broken

import bad "example.test/nonexistent"

deploy bad.Thing as T
`)
		res := runSDLEnv(t, dir, env, "build")
		if res.code != 1 {
			t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
		}
		if !strings.Contains(res.stderr, `broken.sdl:3:8: import "example.test/nonexistent"`) {
			t.Errorf("diagnostic not positioned at the import spec:\n%s", res.stderr)
		}
	})
}

// TestBuildVendored proves a vendored solution module builds: the
// driver queries the module graph and loads catalogue packages with an
// explicit -mod=readonly — vendor mode cannot answer module queries —
// and the synthesized build resolves from the module cache and the
// mirrored replaces, never from the solution's vendor/ tree. The
// vendor tree here is deliberately minimal (modules.txt alone flips
// the go tool into vendor mode): if any stage consulted it, the build
// would fail loudly.
func TestBuildVendored(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "sol.sdl", `solution vendored

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: "pong"
	}
}
`)
	if err := os.Mkdir(filepath.Join(dir, "vendor"), 0o777); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "vendor", "modules.txt"),
		"# github.com/modern-engineering/prototype v0.0.0 => "+repoRoot+"\n## explicit; go 1.25.0\n")

	out := filepath.Join(t.TempDir(), "out.json")
	res := runSDL(t, dir, "build", "-o", out)
	if res.code != 0 {
		t.Fatalf("sdl build exited %d in the vendored module\n%s", res.code, res.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	img, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("emitted image does not decode: %v", err)
	}
	if len(img.Records) != 1 || img.Records[0].Name != "Ping1" {
		t.Errorf("records = %+v, want the one Ping1 deploy", img.Records)
	}
}

// TestBuildOutputAtomic pins the -o replacement contract: a failed
// rebuild leaves the previous image byte-identical — whether the fault
// dies early, at parse before the toolchain runs, or late, inside the
// generated compiler — and leaves no temporary litter beside it, while
// a successful rebuild replaces it.
func TestBuildOutputAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	const good = `solution atomic

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1 {
	params {
		count: 1
		target: "pong"
	}
}
`
	dir := solutionModule(t, "sol.sdl", good)
	out := filepath.Join(dir, "out.json")
	res := runSDL(t, dir, "build", "-o", out)
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	prior, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	rebuild := func(name, unit string) {
		t.Helper()
		writeFile(t, filepath.Join(dir, "sol.sdl"), unit)
		res := runSDL(t, dir, "build", "-o", out)
		if res.code != 1 {
			t.Fatalf("%s: exit %d, want 1\n%s", name, res.code, res.stderr)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, prior) {
			t.Errorf("%s: failed rebuild changed the previous image\n--- now ---\n%s", name, got)
		}
	}
	// Early: the unit no longer parses, so the fault dies before the
	// toolchain ever runs.
	rebuild("parse diagnostic", "solution atomic\n\ndeploy 7\n")
	// Late: the unit parses and the generated compiler rejects it.
	rebuild("compiler diagnostic", `solution atomic

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Gone as G
`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("failed rebuilds left temporary litter: %s", e.Name())
		}
	}

	writeFile(t, filepath.Join(dir, "sol.sdl"), strings.ReplaceAll(good, "count: 1", "count: 2"))
	res = runSDL(t, dir, "build", "-o", out)
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, prior) {
		t.Error("successful rebuild left the previous image in place")
	}
	if _, err := image.Decode(bytes.NewReader(got)); err != nil {
		t.Errorf("replaced image does not decode: %v", err)
	}
}

// TestBuildOutputDevice pins the non-regular -o special case: a device
// target is streamed into directly — rename cannot replace it and it
// holds no previous image to protect — and a failed build neither
// warns about nor attempts its removal.
func TestBuildOutputDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "sol.sdl", `solution device

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Ping as Ping1
`)
	res := runSDL(t, dir, "build", "-o", os.DevNull)
	if res.code != 0 {
		t.Fatalf("sdl build -o %s exited %d\n%s", os.DevNull, res.code, res.stderr)
	}
	if res.stderr != "" {
		t.Errorf("stderr = %q, want empty", res.stderr)
	}

	writeFile(t, filepath.Join(dir, "sol.sdl"), `solution device

import ff "github.com/modern-engineering/prototype/examples/ff"

deploy ff.Gone as G
`)
	res = runSDL(t, dir, "build", "-o", os.DevNull)
	if res.code != 1 {
		t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
	}
	if strings.Contains(res.stderr, "removing") {
		t.Errorf("failed device-target build warned about removal:\n%s", res.stderr)
	}
}

// TestWorkFlag is (d): -work announces the work directory and leaves
// the generated compiler behind for inspection.
func TestWorkFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	out := filepath.Join(t.TempDir(), "out.json")
	res := runSDL(t, repoRoot, "build", "-work", "-o", out, "examples/pingpong")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	var workdir string
	for line := range strings.Lines(res.stderr) {
		if rest, ok := strings.CutPrefix(line, "WORK="); ok {
			workdir = strings.TrimSpace(rest)
		}
	}
	if workdir == "" {
		t.Fatalf("no WORK= line on stderr:\n%s", res.stderr)
	}
	defer func() {
		if err := os.RemoveAll(workdir); err != nil {
			t.Logf("cleaning up the kept work directory: %v", err)
		}
	}()
	for _, name := range []string{"solmain.go", "go.mod"} {
		if _, err := os.Stat(filepath.Join(workdir, name)); err != nil {
			t.Errorf("work directory is missing %s: %v", name, err)
		}
	}
}
