package text_test

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	seftext "github.com/modern-engineering/prototype/sef/text"
)

func FuzzParse(f *testing.F) {
	for _, path := range []string{"testdata/catalogue.sef", "testdata/strings.sef", "testdata/layout.input", "testdata/escapes.input", "testdata/malformed.sef"} {
		data, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, source []byte) {
		file, err := seftext.Parse("input.sef", bytes.NewReader(source))
		if err != nil {
			return
		}
		var formatted bytes.Buffer
		if err := seftext.Format(&formatted, file); err != nil {
			t.Fatal(err)
		}
		reparsed, err := seftext.Parse("formatted.sef", bytes.NewReader(formatted.Bytes()))
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
		if !bytes.Equal(again.Bytes(), formatted.Bytes()) {
			t.Fatal("formatting is not idempotent")
		}
		var encoded bytes.Buffer
		if err := seftext.Encode(&encoded, file.Artifact()); err != nil {
			t.Fatal(err)
		}
		restored, err := seftext.Parse("encoded.sef", &encoded)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(restored.Artifact(), file.Artifact()) {
			t.Fatal("encoding changed structural information")
		}
	})
}
