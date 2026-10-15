package bundle

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"
)

type memoryFS map[string]*memoryEntry

type memoryEntry struct {
	name     string
	data     []byte
	mode     fs.FileMode
	modified time.Time
}

func newMemoryFS() memoryFS { return memoryFS{".": {name: ".", mode: fs.ModeDir | 0o755}} }

func (m memoryFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	entry, found := m[name]
	if !found {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	file := &memoryFile{memoryEntry: entry, reader: bytes.NewReader(entry.data)}
	if entry.IsDir() {
		for child, entry := range m {
			if child != "." && path.Dir(child) == name {
				file.entries = append(file.entries, fs.FileInfoToDirEntry(entry))
			}
		}
		slices.SortFunc(file.entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	}
	return file, nil
}

func (e *memoryEntry) Name() string       { return e.name }
func (e *memoryEntry) Size() int64        { return int64(len(e.data)) }
func (e *memoryEntry) Mode() fs.FileMode  { return e.mode }
func (e *memoryEntry) ModTime() time.Time { return e.modified }
func (e *memoryEntry) IsDir() bool        { return e.mode.IsDir() }
func (e *memoryEntry) Sys() any           { return nil }

type memoryFile struct {
	*memoryEntry
	reader  *bytes.Reader
	entries []fs.DirEntry
	offset  int
	closed  bool
}

func (f *memoryFile) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	return f.memoryEntry, nil
}
func (f *memoryFile) Close() error { f.closed = true; return nil }
func (f *memoryFile) Read(p []byte) (int, error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	if f.IsDir() {
		return 0, fs.ErrInvalid
	}
	return f.reader.Read(p)
}
func (f *memoryFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	if !f.IsDir() {
		return nil, fs.ErrInvalid
	}
	remaining := len(f.entries) - f.offset
	if n > 0 && remaining == 0 {
		return nil, io.EOF
	}
	if n <= 0 || n > remaining {
		n = remaining
	}
	entries := slices.Clone(f.entries[f.offset : f.offset+n])
	f.offset += n
	return entries, nil
}
