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
		"broken.sdl:14:9: output reference acct.config not yet supported",
	} {
		if !strings.Contains(res.stderr, want) {
			t.Errorf("stderr is missing %q:\n%s", want, res.stderr)
		}
	}
}

// TestFmtExamples is (f): the checked-in examples are canonical, so the
// repository holds the form the toolchain prints.
func TestFmtExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	res := runSDL(t, repoRoot, "fmt", "-l", "examples/pingpong")
	if res.code != 0 || res.stdout != "" {
		t.Errorf("fmt -l exited %d and listed %q; the examples must stay canonical\n%s",
			res.code, res.stdout, res.stderr)
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
