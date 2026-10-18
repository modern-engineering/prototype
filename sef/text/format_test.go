package text_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	seftext "github.com/modern-engineering/prototype/sef/text"
)

// Golden pairs describe formatting and comment placement. Formatting the result
// again must be a fixed point, with the same artifact before and after.
func TestFormatPreservesMeaningAndComments(t *testing.T) {
	paths, err := filepath.Glob("testdata/*.input")
	if err != nil || len(paths) == 0 {
		t.Fatalf("format fixtures = %v, %v", paths, err)
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(path, ".input") + ".golden")
			if err != nil {
				t.Fatal(err)
			}
			file, err := seftext.Parse(path, bytes.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := seftext.Format(&output, file); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(output.Bytes(), want) {
				t.Fatalf("formatted output:\n%s\nwant:\n%s", output.Bytes(), want)
			}
			reparsed, err := seftext.Parse(path, bytes.NewReader(output.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reparsed.Artifact(), file.Artifact()) {
				t.Fatal("formatting changed structural information")
			}
			var again bytes.Buffer
			if err := seftext.Format(&again, reparsed); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(again.Bytes(), output.Bytes()) {
				t.Fatalf("formatting is not idempotent:\n%s", again.Bytes())
			}
		})
	}
}

// Source positions let semantic tools identify repeated assignments by index,
// without assuming a unique node/input pair or parsing diagnostic text.
func TestParsedDeclarationsHaveSourcePositions(t *testing.T) {
	input, err := os.ReadFile("testdata/catalogue.sef")
	if err != nil {
		t.Fatal(err)
	}
	file, err := seftext.Parse("catalogue.sef", bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		lookup func(int) (seftext.Position, bool)
		index  int
		line   int
	}{
		{lookup: file.NodePosition, index: 0, line: 10},
		{lookup: file.BindingPosition, index: 2, line: 16},
		{lookup: file.BindingPosition, index: 3, line: 17},
	} {
		position, ok := test.lookup(test.index)
		if !ok || position.Filename != "catalogue.sef" || position.Line != test.line || position.Column != 5 {
			t.Errorf("source position = %+v, %t; want catalogue.sef:%d:5", position, ok, test.line)
		}
	}
	if _, ok := file.NodePosition(-1); ok {
		t.Error("negative node index has a position")
	}
	if _, ok := file.BindingPosition(4); ok {
		t.Error("out-of-range binding has a position")
	}
}

func TestFormatRejectsIncompleteFiles(t *testing.T) {
	file, err := seftext.Parse("broken.sef", strings.NewReader("sef"))
	if err == nil {
		t.Fatal("malformed file parsed successfully")
	}
	var output bytes.Buffer
	if err := seftext.Format(&output, file); err == nil || output.Len() != 0 {
		t.Fatalf("Format incomplete file wrote %d bytes, err = %v", output.Len(), err)
	}
	valid, err := seftext.Parse("valid.sef", strings.NewReader(`sef "0.1"`))
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("storage unavailable")
	if err := seftext.Format(errorWriter{failure}, valid); !errors.Is(err, failure) {
		t.Errorf("Format failing writer = %v", err)
	}
}
