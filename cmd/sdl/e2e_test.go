// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// End-to-end proof of the toolchain: these tests build the sdl CLI
// once, then drive it as a user would — real solution directories, real
// module contexts, real toolchain runs, images echoed back into fresh
// solutions. They are the slow path; -short skips them all.
package main_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		if err := buildCLI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	code := m.Run()
	if sdlPath != "" {
		if err := os.RemoveAll(filepath.Dir(sdlPath)); err != nil {
			fmt.Fprintln(os.Stderr, "cleaning up the built CLI:", err)
		}
	}
	os.Exit(code)
}

// buildCLI compiles the sdl command once for the whole suite, from the
// module root so the test does not care which package directory the
// harness runs it in.
func buildCLI() error {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return fmt.Errorf("locating module root: %v", err)
	}
	repoRoot = strings.TrimSpace(string(out))

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
// golden image, and the bytes decode as a well-formed image.
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
	if img.Solution != "pingpong" || len(img.Records) != 3 || len(img.Catalogue) != 1 {
		t.Errorf("decoded image: solution %q, %d records, %d packages; want pingpong, 3, 1",
			img.Solution, len(img.Records), len(img.Catalogue))
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

// solutionModule lays down a throwaway module holding one solution
// unit, with the prototype module dir-replaced to this checkout — the
// shape of a user module consuming the framework.
func solutionModule(t *testing.T, unitName, unitSource string) string {
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
	if err := os.WriteFile(filepath.Join(dir, unitName), []byte(unitSource), 0o666); err != nil {
		t.Fatal(err)
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
}

// TestRoundTripEcho is (e): the image survives being made visible.
// Building the public example, echoing its image into a fresh solution
// module, and building the echoed unit yields the same desired state —
// image.Equal and, since generation and catalogue coincide here, the
// same bytes. The echoed unit itself is canonical, so sdl fmt has
// nothing to say about it.
func TestRoundTripEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	outA := filepath.Join(t.TempDir(), "a.json")
	res := runSDL(t, repoRoot, "build", "-o", outA, "examples/pingpong")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}

	res = runSDL(t, repoRoot, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)

	dir := solutionModule(t, "pingpong.sdl", echoed)
	if res := runSDL(t, dir, "fmt", "-l", "."); res.code != 0 || res.stdout != "" {
		t.Errorf("echoed unit is not canonical: fmt -l exited %d, listed %q\n%s",
			res.code, res.stdout, res.stderr)
	}

	outB := filepath.Join(dir, "b.json")
	start := time.Now()
	res = runSDL(t, dir, "build", "-o", outB)
	t.Logf("round-trip rebuild: %v", time.Since(start))
	if res.code != 0 {
		t.Fatalf("sdl build of the echoed unit exited %d\n%s", res.code, res.stderr)
	}

	bytesA, err := os.ReadFile(outA)
	if err != nil {
		t.Fatal(err)
	}
	bytesB, err := os.ReadFile(outB)
	if err != nil {
		t.Fatal(err)
	}
	imgA, err := image.Decode(bytes.NewReader(bytesA))
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := image.Decode(bytes.NewReader(bytesB))
	if err != nil {
		t.Fatal(err)
	}
	if !image.Equal(imgA, imgB) {
		t.Errorf("round-tripped image is not Equal to the original\n--- rebuilt ---\n%s", bytesB)
	}
	if !bytes.Equal(bytesA, bytesB) {
		t.Errorf("round-tripped image differs byte-wise\n--- original ---\n%s--- rebuilt ---\n%s", bytesA, bytesB)
	}
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
	count: 3
	target: apiToken
}

deploy ff.Pong as Pong1 {
	subject: pongSubject
}
`

// TestSymbolsRoundTrip drives the symbols vertical end to end: the
// solution's extern and var survive into the image (with the extern's
// sensitivity tainting the binding that references it), echo renders
// them back as declarations and bare-identifier references, and the
// echoed unit rebuilds to an Equal image.
func TestSymbolsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "symbols.sdl", symbolsUnit)
	outA := filepath.Join(dir, "a.json")
	res := runSDL(t, dir, "build", "-o", outA)
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
	if len(imgA.Symbols) != 2 ||
		imgA.Symbols[0].Name != "apiToken" || imgA.Symbols[0].Class != image.ClassExtern ||
		imgA.Symbols[1].Name != "pongSubject" || imgA.Symbols[1].Class != image.ClassVar {
		t.Errorf("symbol table = %+v, want extern apiToken then var pongSubject", imgA.Symbols)
	}
	if len(imgA.Records) != 2 || len(imgA.Records[0].Params) != 2 {
		t.Fatalf("records = %+v, want Ping1 (2 params) and Pong1", imgA.Records)
	}
	target := imgA.Records[0].Params[1]
	if target.Key != "target" || target.Ref == nil || target.Ref.Symbol != "apiToken" || !target.Sensitive {
		t.Errorf("Ping1 target = %+v, want a sensitive ref to apiToken", target)
	}

	res = runSDL(t, dir, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)
	if !strings.Contains(echoed, "extern apiToken substrate.Secret") ||
		!strings.Contains(echoed, `var pongSubject: "ping"`) ||
		!strings.Contains(echoed, "target: apiToken") {
		t.Errorf("echoed unit is missing the symbol blocks or the reference params:\n%s", echoed)
	}

	dir2 := solutionModule(t, "symbols.sdl", echoed)
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
	if !image.Equal(imgA, imgB) {
		t.Errorf("round-tripped image is not Equal to the original\n--- rebuilt ---\n%s", bytesB)
	}
	if !bytes.Equal(bytesA, bytesB) {
		t.Errorf("round-tripped image differs byte-wise\n--- original ---\n%s--- rebuilt ---\n%s", bytesA, bytesB)
	}
}

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
	target: missing
	count: acct.config
}
`)
	res := runSDL(t, dir, "build")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1\n%s", res.code, res.stderr)
	}
	for _, want := range []string{
		"broken.sdl:8:18: unknown element Gone",
		"broken.sdl:12:19: duplicate symbol Ping1 (first declared at broken.sdl:10:5)",
		"broken.sdl:13:10: undefined symbol missing",
		"broken.sdl:14:9: undefined symbol acct",
	} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
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
	cluster: natsCluster
	adminAccount: natsAdmin
}

provision sub.Postgres attach as pgLegacy {
	server: pgServer
}

deploy ff.Ping as Ping1 {
	count: 1
	target: natsAccount.config
}
`

// TestProvisionsRoundTrip drives the provisions vertical end to end:
// slice and attach records survive into the image with their kinds
// explicit, the output reference lands as a symbol-plus-output ref
// tainted by the output's sensitivity, echo renders the provision
// statements and the dotted reference back, and the echoed unit
// rebuilds to an Equal — and, with no defaults in play, byte-identical
// — image.
func TestProvisionsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "provisions.sdl", provisionsUnit)
	outA := filepath.Join(dir, "a.json")
	res := runSDL(t, dir, "build", "-o", outA)
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
	if len(imgA.Records) != 3 ||
		imgA.Records[0].Verb != image.VerbProvision || imgA.Records[0].Kind != image.KindSlice || imgA.Records[0].Name != "natsAccount" ||
		imgA.Records[1].Verb != image.VerbProvision || imgA.Records[1].Kind != image.KindAttach || imgA.Records[1].Name != "pgLegacy" ||
		imgA.Records[2].Verb != image.VerbDeploy || imgA.Records[2].Kind != "" {
		t.Fatalf("records = %+v, want slice natsAccount, attach pgLegacy, deploy Ping1", imgA.Records)
	}
	target := imgA.Records[2].Params[1]
	if target.Key != "target" || target.Ref == nil ||
		target.Ref.Symbol != "natsAccount" || target.Ref.Output != "config" || !target.Sensitive {
		t.Errorf("Ping1 target = %+v, want a sensitive ref to natsAccount.config", target)
	}

	res = runSDL(t, dir, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)
	for _, want := range []string{
		// Echo references through the registered package name, not the
		// original unit's sub alias.
		"provision substrate.NATS slice as natsAccount {",
		"provision substrate.Postgres attach as pgLegacy {",
		"target: natsAccount.config",
	} {
		if !strings.Contains(echoed, want) {
			t.Errorf("echoed unit is missing %q:\n%s", want, echoed)
		}
	}

	dir2 := solutionModule(t, "provisions.sdl", echoed)
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
	if !image.Equal(imgA, imgB) {
		t.Errorf("round-tripped image is not Equal to the original\n--- rebuilt ---\n%s", bytesB)
	}
	if !bytes.Equal(bytesA, bytesB) {
		t.Errorf("round-tripped image differs byte-wise\n--- original ---\n%s--- rebuilt ---\n%s", bytesA, bytesB)
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
	adminAccount: selfRef.config
}
provision sub.NATS slice as loopA {
	adminAccount: loopB.config
}
provision sub.NATS slice as loopB {
	adminAccount: loopA.config
}
deploy ff.Pong as Echo
deploy ff.Ping as P {
	target: natsMissing.config
	count: selfRef.nope
	interval: Echo.config
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
		"broken.sdl:13:29: provision reference cycle: loopA -> loopB -> loopA",
		"broken.sdl:21:10: undefined symbol natsMissing",
		"broken.sdl:22:17: unknown output nope: provision type substrate.NATS declares no such output",
		"broken.sdl:23:12: instance Echo has no outputs: only provision instances emit outputs",
	} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
	}
}

// TestDefaultsRoundTrip is the defaults-bearing round trip the design
// gate demanded: a type default folds into a record that never wrote
// the parameter, echo renders the folded binding explicitly, and the
// rebuild differs from the original only in that binding's Source —
// which Equal masks, so the trip still closes. The bytes must differ,
// or the provenance mask would be proving nothing.
func TestDefaultsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "defaults.sdl", `solution defaults

import ff "github.com/modern-engineering/prototype/examples/ff"

default ff.Ping {
	count: -1
}

deploy ff.Ping as Ping1 {
	target: "pong"
}
`)
	outA := filepath.Join(dir, "a.json")
	res := runSDL(t, dir, "build", "-o", outA)
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
	if len(imgA.Records) != 1 || len(imgA.Records[0].Params) != 2 {
		t.Fatalf("records = %+v, want Ping1 with count and target", imgA.Records)
	}
	count := imgA.Records[0].Params[0]
	if count.Key != "count" || count.Value == nil || count.Value.Int != -1 || count.Source != image.SourceDefaultType {
		t.Fatalf("count binding = %+v, want -1 from %q", count, image.SourceDefaultType)
	}

	res = runSDL(t, dir, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	if !strings.Contains(echoed, "count: -1") {
		t.Errorf("echo must render the folded default explicitly:\n%s", echoed)
	}

	dir2 := solutionModule(t, "defaults.sdl", echoed)
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
	if got := imgB.Records[0].Params[0].Source; got != image.SourceInstance {
		t.Errorf("rebuilt count Source = %q, want %q (echo folds defaults into instance text)", got, image.SourceInstance)
	}
	if !image.Equal(imgA, imgB) {
		t.Errorf("round trip is not Equal despite the provenance mask\n--- rebuilt ---\n%s", bytesB)
	}
	if bytes.Equal(bytesA, bytesB) {
		t.Error("images are byte-identical; the fold should have changed a Source and this test its meaning")
	}
}

// compartmentsUnit is the mockup's compartment material: a verb
// default carrying deployment intent, an instance overriding it, and
// opaque metadata riding along.
const compartmentsUnit = `solution compartments

import ff "github.com/modern-engineering/prototype/examples/ff"

default deploy {
	on {
		location: awsUsEast1
	}
}

deploy ff.Ping as Ping1 {
	count: 1
	target: "pong"
}

deploy ff.Ping as Ping2 {
	count: 2
	target: "pong"

	on {
		location: euCentral1
	}
	metadata {
		team: "search"
	}
}
`

// TestCompartmentsRoundTrip drives on and metadata end to end: the
// verb default's on folds into every deploy record (tokens staying
// opaque), the instance's own on wins its key, metadata rides along,
// echo renders the compartments back after the params, and the echoed
// unit rebuilds to an Equal image — with different bytes, since echo
// folds the verb default into instance text and Equal masks exactly
// that provenance.
func TestCompartmentsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	dir := solutionModule(t, "compartments.sdl", compartmentsUnit)
	outA := filepath.Join(dir, "a.json")
	res := runSDL(t, dir, "build", "-o", outA)
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
	if len(imgA.Records) != 2 {
		t.Fatalf("records = %+v, want Ping1 and Ping2", imgA.Records)
	}
	one, two := imgA.Records[0], imgA.Records[1]
	if len(one.On) != 1 || one.On[0].Key != "location" ||
		one.On[0].Value == nil || one.On[0].Value.Kind != image.KindToken ||
		one.On[0].Value.Tok != "awsUsEast1" || one.On[0].Source != image.SourceDefaultDeploy {
		t.Errorf("Ping1 on = %+v, want the folded default-deploy token awsUsEast1", one.On)
	}
	if len(two.On) != 1 || two.On[0].Value == nil || two.On[0].Value.Tok != "euCentral1" ||
		two.On[0].Source != image.SourceInstance {
		t.Errorf("Ping2 on = %+v, want the instance token euCentral1", two.On)
	}
	if len(two.Metadata) != 1 || two.Metadata[0].Key != "team" ||
		two.Metadata[0].Value == nil || two.Metadata[0].Value.Str != "search" {
		t.Errorf("Ping2 metadata = %+v, want team: search", two.Metadata)
	}

	res = runSDL(t, dir, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)
	for _, want := range []string{
		"location: awsUsEast1",
		"location: euCentral1",
		"team: \"search\"",
	} {
		if !strings.Contains(echoed, want) {
			t.Errorf("echoed unit is missing %q:\n%s", want, echoed)
		}
	}

	dir2 := solutionModule(t, "compartments.sdl", echoed)
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
	if !image.Equal(imgA, imgB) {
		t.Errorf("round trip is not Equal despite the provenance mask\n--- rebuilt ---\n%s", bytesB)
	}
	if bytes.Equal(bytesA, bytesB) {
		t.Error("images are byte-identical; the on fold should have changed a Source and this test its meaning")
	}
}

// TestFactoredForms proves factored spec blocks are pure notation: a
// unit written with factored deploy and provision blocks compiles to
// the byte-identical image of its single-form twin.
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
		cluster: natsCluster
		adminAccount: natsAdmin
	}
	sub.Postgres attach as pgLegacy
)

deploy (
	ff.Ping as Ping1 {
		count: 1
		target: natsAccount.config
	}
	ff.Pong as Pong1 {
		subject: "ping"
	}
)
`
	single := "solution factored\n" + imports + `
provision sub.NATS slice as natsAccount {
	cluster: natsCluster
	adminAccount: natsAdmin
}

provision sub.Postgres attach as pgLegacy

deploy ff.Ping as Ping1 {
	count: 1
	target: natsAccount.config
}

deploy ff.Pong as Pong1 {
	subject: "ping"
}
`
	var images [2][]byte
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
		images[i] = data
	}
	if !bytes.Equal(images[0], images[1]) {
		t.Errorf("factored and single-form units compiled to different images\n--- factored ---\n%s--- single ---\n%s",
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

// TestBuildSample is the CP-C proof: the mockup-6 sample lives in
// examples/sample and compiles to exactly the golden image — extern
// and var symbols, both provision kinds, folded type and verb
// defaults, output references, compartments, and a peer unit — and
// the image plumbing reads the result back.
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
	if img.Solution != "sample" || len(img.Records) != 6 || len(img.Catalogue) != 2 || len(img.Symbols) != 5 {
		t.Errorf("decoded image: solution %q, %d records, %d packages, %d symbols; want sample, 6, 2, 5",
			img.Solution, len(img.Records), len(img.Catalogue), len(img.Symbols))
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
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("records table is missing %q:\n%s", want, res.stdout)
		}
	}
}

// TestSampleRoundTrip closes the loop over the living sample: echoing
// its image into a fresh solution module and rebuilding yields the
// same desired state. The bytes differ — echo folds the sample's
// defaults into instance text, so Sources move — and Equal masks
// exactly that.
func TestSampleRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	outA := filepath.Join(t.TempDir(), "a.json")
	res := runSDL(t, repoRoot, "build", "-o", outA, "examples/sample")
	if res.code != 0 {
		t.Fatalf("sdl build exited %d\n%s", res.code, res.stderr)
	}
	res = runSDL(t, repoRoot, "echo", outA)
	if res.code != 0 {
		t.Fatalf("sdl echo exited %d\n%s", res.code, res.stderr)
	}
	echoed := res.stdout
	t.Logf("echoed unit:\n%s", echoed)

	dir := solutionModule(t, "sample.sdl", echoed)
	if res := runSDL(t, dir, "fmt", "-l", "."); res.code != 0 || res.stdout != "" {
		t.Errorf("echoed unit is not canonical: fmt -l exited %d, listed %q\n%s",
			res.code, res.stdout, res.stderr)
	}
	outB := filepath.Join(dir, "b.json")
	res = runSDL(t, dir, "build", "-o", outB)
	if res.code != 0 {
		t.Fatalf("sdl build of the echoed unit exited %d\n%s", res.code, res.stderr)
	}

	bytesA, err := os.ReadFile(outA)
	if err != nil {
		t.Fatal(err)
	}
	bytesB, err := os.ReadFile(outB)
	if err != nil {
		t.Fatal(err)
	}
	imgA, err := image.Decode(bytes.NewReader(bytesA))
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := image.Decode(bytes.NewReader(bytesB))
	if err != nil {
		t.Fatal(err)
	}
	if !image.Equal(imgA, imgB) {
		t.Errorf("round-tripped sample is not Equal to the original\n--- rebuilt ---\n%s", bytesB)
	}
	if bytes.Equal(bytesA, bytesB) {
		t.Error("images are byte-identical; the sample's defaults should fold into different Sources")
	}
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
	greeting: "hi"
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
