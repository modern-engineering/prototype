package bind_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/modern-engineering/prototype/bind"
)

// A byte binding can consume a reader and provide the same content to a writer.
func ExampleBytes() {
	var content []byte
	value := bind.Bytes(&content)
	read, err := value.ReadFrom(strings.NewReader("payload"))
	if err != nil {
		fmt.Println(err)
		return
	}
	var output bytes.Buffer
	written, err := value.WriteTo(&output)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("read %d, wrote %d: %s\n", read, written, output.String())
	content[0] = 'P'
	fmt.Println(value.Get())
	// Output:
	// read 7, wrote 7: payload
	// Payload
}

// Mutable byte storage stays bound to the variable, while returned strings are
// snapshots that survive later changes to those bytes.
func TestByteSnapshots(t *testing.T) {
	content := []byte("seed")
	value := bind.Bytes(&content)
	snapshot := value.Get()
	content[0] = 'S'
	if snapshot != "seed" || value.Get() != "Seed" {
		t.Fatalf("snapshot = %q, current value = %q; want seed and Seed", snapshot, value.Get())
	}
	text := "next"
	if err := value.Set(text); err != nil {
		t.Fatal(err)
	}
	if string(content) != "next" {
		t.Fatalf("assigned bytes = %q; want next", content)
	}
	content[0] = 'N'
	if text != "next" || value.Get() != "Next" {
		t.Errorf("source text = %q, current value = %q; want next and Next", text, value.Get())
	}
}

// Partial transfers preserve usable bound data and leave resource ownership
// with the caller, whether the transfer succeeds or fails.
func TestByteTransferFailuresAndOwnership(t *testing.T) {
	content := []byte("seed")
	value := bind.Bytes(&content)
	readErr := errors.New("source failed")
	source := &closeTrackingReader{Reader: io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr))}
	n, err := value.ReadFrom(source)
	if n != 7 || !errors.Is(err, readErr) || string(content) != "seed" {
		t.Fatalf("failed read = %d, %v, %q; want 7, source error, and unchanged seed", n, err, content)
	}
	if source.closed {
		t.Fatal("failed read closed the source")
	}
	source.Reader = strings.NewReader("replacement")
	n, err = value.ReadFrom(source)
	if n != 11 || err != nil || string(content) != "replacement" || source.closed {
		t.Fatalf("successful read = %d, %v, %q, closed=%t; want replacement and an open source", n, err, content, source.closed)
	}

	writeErr := errors.New("destination failed")
	destination := &writeResult{n: 2, err: writeErr}
	n, err = value.WriteTo(destination)
	if n != 2 || !errors.Is(err, writeErr) {
		t.Errorf("failed write = %d, %v; want 2 and destination error", n, err)
	}
	destination.err = nil
	n, err = value.WriteTo(destination)
	if n != 2 || !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("short write = %d, %v; want 2 and short-write error", n, err)
	}
	destination.n = 11
	n, err = value.WriteTo(destination)
	if n != 11 || err != nil {
		t.Errorf("complete write = %d, %v; want 11 and no error", n, err)
	}
	if string(content) != "replacement" || destination.closed {
		t.Errorf("written value = %q, destination closed=%t; want unchanged replacement and an open destination", content, destination.closed)
	}
}

type closeTrackingReader struct {
	io.Reader
	closed bool
}

func (r *closeTrackingReader) Close() error { r.closed = true; return nil }

type writeResult struct {
	n      int
	err    error
	closed bool
}

func (w *writeResult) Write([]byte) (int, error) { return w.n, w.err }
func (w *writeResult) Close() error              { w.closed = true; return nil }
