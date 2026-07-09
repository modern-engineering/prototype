// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"bytes"
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
