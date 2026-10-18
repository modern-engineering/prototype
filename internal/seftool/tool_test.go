package seftool_test

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/internal/seftool"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/sef"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution"
)

// A command host can inspect artifacts without loading their catalogue code.
func ExampleCommand() {
	var out, diagnostics bytes.Buffer
	command := seftool.Command{Stdout: &out, Stderr: &diagnostics}
	status := command.Run([]string{"check", "testdata/sample.sef"})
	fmt.Print(out.String())
	fmt.Println("status:", status)
	if diagnostics.Len() != 0 {
		fmt.Print(diagnostics.String())
	}
	// Output:
	// ok
	// status: 0
}

// Pipelines must preserve logical content across encodings without turning
// inspection or registration-source generation into program execution.
func TestArtifactWorkflow(t *testing.T) {
	source, err := os.ReadFile("testdata/sample.sef")
	if err != nil {
		t.Fatal(err)
	}
	file, err := seftext.Parse("sample.sef", bytes.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	expected := file.Artifact()
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Stdout: &output, Stderr: &diagnostics}
	for _, format := range []string{"json", "gob", "text"} {
		output.Reset()
		diagnostics.Reset()
		command.Stdin = bytes.NewReader(source)
		if status := command.Run([]string{"convert", "-to", format}); status != 0 {
			t.Fatalf("convert %s status=%d: %s", format, status, &diagnostics)
		}
		encoded := bytes.Clone(output.Bytes())
		output.Reset()
		command.Stdin = bytes.NewReader(encoded)
		if status := command.Run([]string{"dump", "-from", format}); status != 0 {
			t.Fatalf("dump %s status=%d: %s", format, status, &diagnostics)
		}
		got, err := sef.DecodeJSON(&output)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s changed the artifact: got %#v; want %#v", format, got, expected)
		}
	}
	output.Reset()
	if status := command.Run([]string{"inspect", "testdata/sample.sef"}); status != 0 {
		t.Fatalf("inspect status=%d: %s", status, &diagnostics)
	}
	if strings.Contains(output.String(), "credential-not-for-inspection") {
		t.Fatal("inspect disclosed a literal")
	}
	output.Reset()
	if status := command.Run([]string{"gen", "-package", "generated", "-func", "Install", "testdata/sample.sef"}); status != 0 {
		t.Fatalf("gen status=%d: %s", status, &diagnostics)
	}
	generated, err := parser.ParseFile(token.NewFileSet(), "registry.go", output.Bytes(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Name.Name != "generated" {
		t.Errorf("package=%s; want generated", generated.Name.Name)
	}
	output.Reset()
	if status := command.Run([]string{"rename", "-from", "producer", "-to", "primary", "testdata/sample.sef"}); status != 0 {
		t.Fatalf("rename status=%d: %s", status, &diagnostics)
	}
	renamed, err := seftext.Parse("renamed.sef", &output)
	if err != nil {
		t.Fatal(err)
	}
	got := renamed.Artifact()
	if got.Definition.Nodes[0].Name != "primary" || got.Definition.Bindings[0].Node != "primary" || got.Definition.Bindings[1].Value.(solution.Reference).Node != "primary" {
		t.Fatalf("rename left inconsistent relationships: %#v", got.Definition)
	}
}

// Descriptor-aware inspection may call Make only when I/O declarations are
// requested. Even then it must leave setters, getters, and runners untouched.
func TestCatalogueInspection(t *testing.T) {
	makes, sets, gets, runs := 0, 0, 0, 0
	var registry program.Registry
	descriptor := program.MakeFunc(func() (program.Runner, program.Inputs, program.Outputs) {
		makes++
		return program.RunnerFunc(func(context.Context) error { runs++; return nil }),
			program.InputList{{Field: program.Field{Name: "name", Sensitive: true}, Value: program.SetterFunc(func(string) error { sets++; return nil })}},
			program.OutputList{{Field: program.Field{Name: "message"}, Value: program.GetterFunc(func() string { gets++; return "must not read" })}}
	})
	if err := registry.Register("example.test/catalogue", "Service", descriptor); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Registry: &registry, Stdout: &output, Stderr: &diagnostics}
	for _, args := range [][]string{{"check", "-linked", "testdata/sample.sef"}, {"inspect", "-linked", "testdata/sample.sef"}} {
		if status := command.Run(args); status != 0 {
			t.Fatalf("%v status=%d: %s", args, status, &diagnostics)
		}
	}
	if makes != 0 {
		t.Fatalf("linked checks made %d invocations; want zero", makes)
	}
	if status := command.Run([]string{"inspect", "-inputs", "testdata/sample.sef"}); status != 0 {
		t.Fatalf("inputs status=%d: %s", status, &diagnostics)
	}
	if makes != 2 || sets != 0 || gets != 0 || runs != 0 {
		t.Errorf("calls Make/Set/Get/Run=%d/%d/%d/%d; want 2/0/0/0", makes, sets, gets, runs)
	}
}

// Rewriting parses the whole input before replacing the original file, including
// when input and output are the same path. Failed operations leave it intact.
func TestFileReplacement(t *testing.T) {
	source, err := os.ReadFile("testdata/sample.sef")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sample.sef")
	if err := os.WriteFile(path, source, 0o640); err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	command := seftool.Command{Stderr: &diagnostics}
	if status := command.Run([]string{"fmt", "-w", path}); status != 0 {
		t.Fatalf("fmt status=%d: %s", status, &diagnostics)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if status := command.Run([]string{"fmt", "-w", path}); status != 0 {
		t.Fatalf("second fmt status=%d: %s", status, &diagnostics)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("formatting is not idempotent")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode=%o; want640", info.Mode().Perm())
	}
	if status := command.Run([]string{"convert", "-from", "json", "-o", path, path}); status != 1 {
		t.Fatalf("invalid conversion status=%d; want1", status)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unchanged, second) {
		t.Fatal("failed conversion changed input/output file")
	}
	jsonPath := filepath.Join(t.TempDir(), "sample.json.sef")
	if status := command.Run([]string{"convert", "-o", jsonPath, path}); status != 0 {
		t.Fatalf("file convert status=%d: %s", status, &diagnostics)
	}
	if status := command.Run([]string{"check", jsonPath}); status != 0 {
		t.Fatalf("suffix check status=%d: %s", status, &diagnostics)
	}
}

// Usage failures remain distinct from malformed input so shell callers can
// distinguish an invocation mistake from a rejected artifact.
func TestExitStatus(t *testing.T) {
	for _, test := range []struct {
		args   []string
		status int
	}{
		{nil, 2}, {[]string{"unknown"}, 2}, {[]string{"help"}, 0},
		{[]string{"fmt", "-h"}, 0}, {[]string{"fmt", "-w"}, 2},
		{[]string{"convert", "-from", "yaml"}, 2},
		{[]string{"check", "-linked", "testdata/sample.sef"}, 1},
		{[]string{"check"}, 1}, {[]string{"check", "testdata/missing.sef"}, 1},
	} {
		var output, diagnostics bytes.Buffer
		command := seftool.Command{Stdin: strings.NewReader("broken"), Stdout: &output, Stderr: &diagnostics}
		if status := command.Run(test.args); status != test.status {
			t.Errorf("%v status=%d; want%d", test.args, status, test.status)
		}
		if test.status != 0 && output.Len() != 0 {
			t.Errorf("%v produced output on failure", test.args)
		}
	}
}

// Empty occurrence names are exact identities, so a present empty flag value
// must remain distinct from an omitted rename flag.
func TestRenameEmptyName(t *testing.T) {
	var output, diagnostics bytes.Buffer
	command := seftool.Command{Stdout: &output, Stderr: &diagnostics}
	if status := command.Run([]string{"rename", "-from", "producer", "-to", "", "testdata/sample.sef"}); status != 0 {
		t.Fatalf("rename empty status=%d: %s", status, &diagnostics)
	}
	source := bytes.Clone(output.Bytes())
	output.Reset()
	command.Stdin = bytes.NewReader(source)
	if status := command.Run([]string{"rename", "-from", "", "-to", "restored"}); status != 0 {
		t.Fatalf("rename from empty status=%d: %s", status, &diagnostics)
	}
	file, err := seftext.Parse("renamed.sef", &output)
	if err != nil {
		t.Fatal(err)
	}
	if got := file.Artifact().Definition.Nodes[0].Name; got != "restored" {
		t.Errorf("name=%q; want restored", got)
	}
}

// Atomic replacement must not silently replace a symbolic link with a regular
// file, and rejection must leave both the link and its target untouched.
func TestSymlinkOutput(t *testing.T) {
	source, err := os.ReadFile("testdata/sample.sef")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	target, path := filepath.Join(dir, "target.sef"), filepath.Join(dir, "link.sef")
	if err := os.WriteFile(target, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	command := seftool.Command{Stderr: &diagnostics}
	if status := command.Run([]string{"fmt", "-w", path}); status != 1 {
		t.Fatalf("symlink replacement status=%d; want1", status)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symbolic link was replaced")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, source) {
		t.Fatal("symlink target changed")
	}
}
