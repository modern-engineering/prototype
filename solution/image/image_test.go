// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution/image"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	img := testImage()
	var buf bytes.Buffer
	if err := img.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.HasSuffix(buf.Bytes(), []byte("}\n")) {
		t.Error("Encode output must end in a single trailing newline")
	}
	got, err := image.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !image.Equal(img, got) {
		t.Error("Decode(Encode(img)) is not Equal to img")
	}
	if got.Generation != img.Generation {
		t.Errorf("Decode Generation = %d, want %d (carried even though Equal masks it)", got.Generation, img.Generation)
	}
}

// TestBuildRoundTrip pins the governance block's carriage: encoding
// keeps it (in header position, between generation and catalogue) and
// decoding restores it verbatim — Equal masks the block, so the
// comparison here is direct — while a document without one decodes to
// a nil Build, the shape of every image predating the block.
func TestBuildRoundTrip(t *testing.T) {
	img := testImage()
	img.Build = &image.Build{
		Units: []image.UnitDigest{
			{Name: "main.sdl", SHA256: strings.Repeat("12", 32)},
			{Name: "peer.sdl", SHA256: strings.Repeat("34", 32)},
		},
		Settings: []image.Setting{
			{Key: "prototype.version", Value: "v0.1.0"},
			{Key: "sdl.version", Value: "v0.1.2"},
		},
	}
	var buf bytes.Buffer
	if err := img.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	head, _, _ := strings.Cut(buf.String(), `"catalogue"`)
	if !strings.Contains(head, `"build"`) || !strings.Contains(head, `"generation"`) {
		t.Errorf("build must encode between generation and catalogue; header:\n%s", head)
	}
	got, err := image.Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Build == nil || !slices.Equal(got.Build.Units, img.Build.Units) || !slices.Equal(got.Build.Settings, img.Build.Settings) {
		t.Errorf("decoded Build = %+v, want %+v", got.Build, img.Build)
	}

	var plain bytes.Buffer
	if err := testImage().Encode(&plain); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if strings.Contains(plain.String(), `"build"`) {
		t.Error("an image without a Build block must omit the build key")
	}
	old, err := image.Decode(&plain)
	if err != nil {
		t.Fatalf("Decode of a build-less document: %v", err)
	}
	if old.Build != nil {
		t.Errorf("decoded Build = %+v, want nil for a document without one", old.Build)
	}
}

func TestEncodeDeterminism(t *testing.T) {
	var first, second bytes.Buffer
	if err := testImage().Encode(&first); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := testImage().Encode(&second); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("two encodes of the same image differ")
	}
}

// TestEncodeRejectsInvalidUTF8 pins the belt-and-braces guard behind
// the parser's own literal check: encoding/json would silently rewrite
// invalid bytes as U+FFFD, so a non-UTF-8 string reaching Encode must
// fail loudly instead of producing an image that decodes differently.
func TestEncodeRejectsInvalidUTF8(t *testing.T) {
	img := &image.Image{
		Format:   image.Format,
		Solution: "sample",
		Symbols:  []image.SymbolDef{{Name: "v", Class: image.ClassVar, Value: image.String("\xff")}},
	}
	var buf bytes.Buffer
	err := img.Encode(&buf)
	if err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("Encode = %v, want a UTF-8 error", err)
	}
}

// TestDecodeRejectsTrailingData pins that an image is one whole JSON
// document: trailing whitespace (Encode's own newline included) stays
// legal, anything else fails instead of being silently ignored.
func TestDecodeRejectsTrailingData(t *testing.T) {
	var buf bytes.Buffer
	if err := testImage().Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if _, err := image.Decode(strings.NewReader(buf.String() + "\n \t\n")); err != nil {
		t.Errorf("Decode with trailing whitespace = %v, want nil", err)
	}
	for _, trailing := range []string{"{}", "null", "garbage"} {
		_, err := image.Decode(strings.NewReader(buf.String() + trailing))
		if err == nil || !strings.Contains(err.Error(), "trailing data") {
			t.Errorf("Decode with trailing %q = %v, want a trailing-data error", trailing, err)
		}
	}
}

func TestDecodeRejectsForeignFormat(t *testing.T) {
	_, err := image.Decode(strings.NewReader(`{"format":"solution-image/9"}`))
	if err == nil || !strings.Contains(err.Error(), `format "solution-image/9" is not "solution-image/1"`) {
		t.Errorf("Decode foreign format: err = %v, want format error", err)
	}
	_, err = image.Decode(strings.NewReader(`not json`))
	if err == nil {
		t.Error("Decode(garbage) = nil error, want error")
	}
}
