package solution

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ErrBundle means a directory cannot carry one solution module context.
var ErrBundle = errors.New("invalid solution bundle")

// Bundle is a shippable directory of solution artifacts and supporting files.
// Module is its authoritative Go module context; Files carries the directory's
// content. Artifact encodings remain independent of this filesystem view.
// Consumers apply Module when decoding artifacts from the bundle, replacing any
// artifact-local module context. Archive encoders write Module as the root
// go.mod and go.sum rather than copying potentially stale manifest bytes.
//
// Files must contain only regular files and directories, with no nested go.mod
// or go.sum. It is retained by reference and must remain stable while in use.
// A Bundle owns no open files or temporary directories.
//
// TODO: Define bundle versioning independently of artifact schema versions.
type Bundle struct {
	Module ModuleContext
	Files  fs.FS
}

// OpenBundle reads the root go.mod and optional go.sum from files. A missing
// go.sum means an empty checksum set. It checks the directory shape without
// decoding solution artifacts, resolving modules, or following symbolic links.
// Manifest contents are parsed once; Files continues to refer to the supplied
// filesystem. On error, OpenBundle returns a zero Bundle.
func OpenBundle(files fs.FS) (Bundle, error) {
	if files == nil {
		return Bundle{}, fmt.Errorf("%w: nil filesystem", ErrBundle)
	}
	if err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(name, `\`) || (len(name) > 1 && name[1] == ':') {
			return fmt.Errorf("%w: non-portable path %q", ErrBundle, name)
		}
		if name == "." && !entry.IsDir() {
			return fmt.Errorf("%w: root is not a directory", ErrBundle)
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("%w: unsupported file %q", ErrBundle, name)
		}
		if base := path.Base(name); (base == "go.mod" || base == "go.sum") && name != base {
			return fmt.Errorf("%w: nested module manifest %q", ErrBundle, name)
		}
		return nil
	}); err != nil {
		return Bundle{}, err
	}
	mod, err := fs.ReadFile(files, "go.mod")
	if err != nil {
		return Bundle{}, err
	}
	sum, err := fs.ReadFile(files, "go.sum")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Bundle{}, err
	}
	context, err := ParseModule(bytes.NewReader(mod), bytes.NewReader(sum))
	if err != nil {
		return Bundle{}, err
	}
	return Bundle{Module: context, Files: files}, nil
}
