package command_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/modern-engineering/prototype/bind/output"
	"github.com/modern-engineering/prototype/program"
	"github.com/modern-engineering/prototype/program/command"
)

// Commands can publish every declared result without knowing its field names
// or conflating empty text, numeric-looking text, and multiline content.
func TestOutputFormattingPreservesNamesAndText(t *testing.T) {
	var outputs output.List
	text := outputs.String("text", "first\nsecond", "")
	outputs.String("count", "001", "")
	outputs.String("empty", "", "")
	outputs.String("token", "result-token", "", output.Sensitive)
	*text += "\n"
	var buffer bytes.Buffer
	if err := command.WriteOutputs(&buffer, &outputs); err != nil {
		t.Fatal(err)
	}
	want := "{\"count\":\"001\",\"empty\":\"\",\"text\":\"first\\nsecond\\n\",\"token\":\"result-token\"}\n"
	if got := buffer.String(); got != want {
		t.Fatalf("WriteOutputs = %q; want %q", got, want)
	}
	buffer.Reset()
	if err := command.WriteOutputs(&buffer, nil); err != nil {
		t.Fatal(err)
	}
	if got := buffer.String(); got != "{}\n" {
		t.Fatalf("empty WriteOutputs = %q; want empty object", got)
	}
	writeErr := errors.New("output unavailable")
	if err := command.WriteOutputs(failingWriter{writeErr}, &outputs); !errors.Is(err, writeErr) {
		t.Fatalf("WriteOutputs error = %v; want writer error", err)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

// Your command can forward all results after Run, even when a catalogue later
// adds an output. Each getter is read when the results are written.
func ExampleWriteOutputs() {
	var outputs output.List
	outputs.String("status", "ready", "service status")
	outputs.Var(program.GetterFunc(func() string { return "https://service.example" }), "url", "service URL")
	if err := command.WriteOutputs(os.Stdout, &outputs); err != nil {
		fmt.Println(err)
	}
	// Output: {"status":"ready","url":"https://service.example"}
}
