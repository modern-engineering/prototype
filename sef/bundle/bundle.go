// Package bundle transfers solution directories as tar and tar.gz archives.
// The root module context is authoritative; artifact and support files retain
// their bytes. Decoding keeps file contents in memory and owns no disk resources.
package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/modern-engineering/prototype/solution"
)

// EncodeTar writes b as a tar archive, replacing root manifest bytes with b.Module.
// The filesystem must remain stable during encoding. Only regular files and
// directories with portable relative paths are accepted, and nested module
// manifests are rejected. Module and directory validation precede output; later
// read or write failures may leave a partial archive. w remains open.
func EncodeTar(w io.Writer, b solution.Bundle) error {
	var mod, sum bytes.Buffer
	if err := b.Module.Write(&mod, &sum); err != nil {
		return err
	}
	if b.Files == nil {
		return fmt.Errorf("%w: nil filesystem", solution.ErrBundle)
	}
	type entry struct {
		name string
		info fs.FileInfo
	}
	var entries []entry
	if err := fs.WalkDir(b.Files, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			if !d.IsDir() {
				return invalid(name, "root is not a directory")
			}
			return nil
		}
		if !validName(name) {
			return invalid(name, "invalid path")
		}
		if !d.IsDir() && !d.Type().IsRegular() {
			return invalid(name, "unsupported file type")
		}
		if base := path.Base(name); (base == "go.mod" || base == "go.sum") && name != base {
			return invalid(name, "nested module manifest")
		}
		if name == "go.mod" || name == "go.sum" {
			if d.IsDir() {
				return invalid(name, "manifest is a directory")
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entries = append(entries, entry{name, info})
		return nil
	}); err != nil {
		return err
	}
	writer := tar.NewWriter(w)
	for _, manifest := range []struct {
		name string
		data []byte
	}{{"go.mod", mod.Bytes()}, {"go.sum", sum.Bytes()}} {
		if err := writer.WriteHeader(&tar.Header{Name: manifest.name, Mode: 0o644, Size: int64(len(manifest.data))}); err != nil {
			return errors.Join(err, writer.Close())
		}
		if _, err := writer.Write(manifest.data); err != nil {
			return errors.Join(err, writer.Close())
		}
	}
	for _, entry := range entries {
		header, err := tar.FileInfoHeader(entry.info, "")
		if err != nil {
			return errors.Join(err, writer.Close())
		}
		header.Name = entry.name
		if err := writer.WriteHeader(header); err != nil {
			return errors.Join(err, writer.Close())
		}
		if entry.info.IsDir() {
			continue
		}
		file, err := b.Files.Open(entry.name)
		if err != nil {
			return errors.Join(err, writer.Close())
		}
		_, copyErr := io.Copy(writer, file)
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return errors.Join(err, writer.Close())
		}
	}
	return writer.Close()
}

// DecodeTar reads a tar archive into an independent in-memory filesystem and
// opens its root module context. Directory entries may be implicit. Leading ./
// and a directory's trailing slash are accepted. Absolute paths, traversal,
// backslashes, drive prefixes, duplicate entries, links, special files, and
// nested module manifests return an error wrapping solution.ErrBundle.
// File contents, permission bits, and modification times are retained; owners
// and other archive metadata are not. r remains open. Errors return a zero bundle.
func DecodeTar(r io.Reader) (solution.Bundle, error) {
	reader := tar.NewReader(r)
	files := newMemoryFS()
	explicit := make(map[string]bool)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return solution.Bundle{}, err
		}
		name := header.Name
		if name == "" || strings.HasPrefix(name, "/") {
			return solution.Bundle{}, invalid(name, "invalid path")
		}
		for strings.HasPrefix(name, "./") {
			name = strings.TrimPrefix(name, "./")
		}
		if header.Typeflag == tar.TypeDir {
			name = strings.TrimSuffix(name, "/")
		}
		if name == "" && header.Typeflag == tar.TypeDir {
			name = "."
		}
		if !validName(name) {
			return solution.Bundle{}, invalid(header.Name, "invalid path")
		}
		if header.Typeflag != tar.TypeDir && header.Typeflag != tar.TypeReg {
			return solution.Bundle{}, invalid(name, "unsupported file type")
		}
		if name == "." && header.Typeflag != tar.TypeDir {
			return solution.Bundle{}, invalid(name, "root is not a directory")
		}
		if explicit[name] {
			return solution.Bundle{}, invalid(name, "duplicate entry")
		}
		explicit[name] = true
		mode := fs.FileMode(header.Mode).Perm()
		if header.Typeflag == tar.TypeDir {
			mode |= fs.ModeDir
		}
		if old, found := files[name]; found && !old.mode.IsDir() {
			return solution.Bundle{}, invalid(name, "entry is already a file")
		}
		if old, found := files[name]; found && old.mode.IsDir() && !mode.IsDir() {
			return solution.Bundle{}, invalid(name, "entry is already a directory")
		}
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if old, found := files[parent]; found {
				if !old.mode.IsDir() {
					return solution.Bundle{}, invalid(parent, "parent is a file")
				}
			} else {
				files[parent] = &memoryEntry{name: path.Base(parent), mode: fs.ModeDir | 0o755}
			}
		}
		var data []byte
		if !mode.IsDir() {
			data, err = io.ReadAll(reader)
			if err != nil {
				return solution.Bundle{}, err
			}
		}
		files[name] = &memoryEntry{name: path.Base(name), data: data, mode: mode, modified: header.ModTime}
	}
	return solution.OpenBundle(files)
}

// EncodeTarGzip writes a gzip-compressed tar archive using EncodeTar's contract.
// It finishes the compressed stream without closing w.
func EncodeTarGzip(w io.Writer, b solution.Bundle) error {
	writer := gzip.NewWriter(w)
	if err := EncodeTar(writer, b); err != nil {
		return err
	}
	return writer.Close()
}

// DecodeTarGzip reads a gzip-compressed tar archive using DecodeTar's contract.
// It verifies the compressed stream through EOF, including its checksum, and
// leaves r open. The entire uncompressed file content is retained in memory.
func DecodeTarGzip(r io.Reader) (solution.Bundle, error) {
	reader, err := gzip.NewReader(r)
	if err != nil {
		return solution.Bundle{}, err
	}
	result, decodeErr := DecodeTar(reader)
	var drainErr error
	if decodeErr == nil {
		_, drainErr = io.Copy(io.Discard, reader)
	}
	if err := errors.Join(decodeErr, drainErr, reader.Close()); err != nil {
		return solution.Bundle{}, err
	}
	return result, nil
}

func validName(name string) bool {
	return fs.ValidPath(name) && !strings.Contains(name, `\`) && (len(name) <= 1 || name[1] != ':')
}

func invalid(name, reason string) error {
	return fmt.Errorf("%w: %s: %q", solution.ErrBundle, reason, name)
}
