package bundle_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	bundlecodec "github.com/modern-engineering/prototype/sef/bundle"
	"github.com/modern-engineering/prototype/solution"
)

var formats = []struct {
	name   string
	encode func(io.Writer, solution.Bundle) error
	decode func(io.Reader) (solution.Bundle, error)
}{
	{"tar", bundlecodec.EncodeTar, bundlecodec.DecodeTar},
	{"tar.gz", bundlecodec.EncodeTarGzip, bundlecodec.DecodeTarGzip},
}

// Shipped files retain their content while root manifests follow the bundle's
// authoritative context, even when its original filesystem still has old bytes.
func TestArchiveTransfer(t *testing.T) {
	directory := t.TempDir()
	if err := os.CopyFS(directory, os.DirFS("testdata/tree")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	source, err := solution.OpenBundle(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	source.Module.Require[0].Version = "v1.2.4"
	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var data bytes.Buffer
			if err := format.encode(&data, source); err != nil {
				t.Fatal(err)
			}
			decoded, err := format.decode(&data)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(decoded.Module.Sum, source.Module.Sum) {
				t.Fatalf("checksums changed: %+v", decoded.Module.Sum)
			}
			if decoded.Module.Require[0].Version != "v1.2.4" {
				t.Fatalf("requirement = %+v", decoded.Module.Require)
			}
			if err := fstest.TestFS(decoded.Files, "go.mod", "go.sum", "solutions/service.sef", "assets/message.txt", "empty"); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"solutions/service.sef", "assets/message.txt"} {
				original, err := fs.ReadFile(source.Files, name)
				if err != nil {
					t.Fatal(err)
				}
				shipped, err := fs.ReadFile(decoded.Files, name)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(original, shipped) {
					t.Fatalf("%s changed", name)
				}
			}
			empty, err := fs.Stat(decoded.Files, "empty")
			if err != nil {
				t.Fatal(err)
			}
			if !empty.IsDir() || empty.Mode().Perm() != 0o750 {
				t.Fatalf("empty directory mode = %v", empty.Mode())
			}
		})
	}
}

// Archive members cannot escape or change the identity of another member.
// Rejection happens before any caller is given a filesystem to extract.
func TestArchiveRejectsAmbiguousMembers(t *testing.T) {
	for _, test := range []struct {
		name string
		kind byte
	}{
		{"../escape", tar.TypeReg}, {"/absolute", tar.TypeReg}, {"/", tar.TypeDir}, {`C:/drive`, tar.TypeReg},
		{`nested\escape`, tar.TypeReg}, {"nested/go.mod", tar.TypeReg}, {"nested/go.sum", tar.TypeReg},
		{"linked", tar.TypeSymlink}, {"hardlinked", tar.TypeLink}, {"pipe", tar.TypeFifo}, {"go.mod", tar.TypeReg},
	} {
		var data bytes.Buffer
		writer := tar.NewWriter(&data)
		manifest := []byte("module example.test/solution\n")
		if err := writer.WriteHeader(&tar.Header{Name: "go.mod", Mode: 0o644, Size: int64(len(manifest))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(manifest); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteHeader(&tar.Header{Name: test.name, Typeflag: test.kind, Linkname: "go.mod", Mode: 0o644}); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := bundlecodec.DecodeTar(&data); !errors.Is(err, solution.ErrBundle) {
			t.Errorf("%q: error = %v, want ErrBundle", test.name, err)
		}
	}
}

// Tar produced from a directory commonly prefixes members with ./ and does not
// necessarily record parent directories before their files.
func TestArchiveDirectoryConventions(t *testing.T) {
	var data bytes.Buffer
	writer := tar.NewWriter(&data)
	for _, entry := range []struct{ name, content string }{
		{"./go.mod", "module example.test/solution\n"}, {"./assets/message.txt", "ready\n"},
	} {
		if err := writer.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, entry.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	decoded, err := bundlecodec.DecodeTar(&data)
	if err != nil {
		t.Fatal(err)
	}
	if err := fstest.TestFS(decoded.Files, "go.mod", "assets/message.txt"); err != nil {
		t.Fatal(err)
	}
}

// Gzip's checksum follows tar's logical EOF and must still be verified.
func TestArchiveTransportFailures(t *testing.T) {
	source, err := solution.OpenBundle(os.DirFS("testdata/tree"))
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := bundlecodec.EncodeTarGzip(&data, source); err != nil {
		t.Fatal(err)
	}
	corrupted := bytes.Clone(data.Bytes())
	corrupted[len(corrupted)-8] ^= 1
	if _, err := bundlecodec.DecodeTarGzip(bytes.NewReader(corrupted)); !errors.Is(err, gzip.ErrChecksum) {
		t.Fatalf("corrupted checksum = %v", err)
	}
	failure := errors.New("transport failed")
	for _, format := range formats {
		if err := format.encode(failingStream{failure}, source); !errors.Is(err, failure) {
			t.Errorf("%s writer: %v", format.name, err)
		}
		if _, err := format.decode(failingStream{failure}); !errors.Is(err, failure) {
			t.Errorf("%s reader: %v", format.name, err)
		}
		var output bytes.Buffer
		invalid := source
		invalid.Files = fstest.MapFS{"link": {Mode: fs.ModeSymlink}}
		if err := format.encode(&output, invalid); !errors.Is(err, solution.ErrBundle) {
			t.Errorf("%s invalid bundle: %v", format.name, err)
		}
		if output.Len() != 0 {
			t.Errorf("%s wrote bytes before validation", format.name)
		}
	}
}

type failingStream struct{ err error }

func (s failingStream) Read([]byte) (int, error)  { return 0, s.err }
func (s failingStream) Write([]byte) (int, error) { return 0, s.err }

// Ship a directory without choosing an artifact encoding.
func ExampleEncodeTarGzip() {
	source, err := solution.OpenBundle(os.DirFS("testdata/tree"))
	if err != nil {
		fmt.Println(err)
		return
	}
	var transfer bytes.Buffer
	if err := bundlecodec.EncodeTarGzip(&transfer, source); err != nil {
		fmt.Println(err)
		return
	}
	received, err := bundlecodec.DecodeTarGzip(&transfer)
	if err != nil {
		fmt.Println(err)
		return
	}
	message, err := fs.ReadFile(received.Files, "assets/message.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(received.Module.Path)
	fmt.Print(string(message))
	// Output:
	// example.test/solution
	// ready to serve
}
