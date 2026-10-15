// Package cue evaluates a single CUE source file into an experimental SEF
// artifact. The file's concrete data must satisfy the embedded artifact schema.
// CUE expressions and definitions may construct that data; graph references
// remain explicit node/output records rather than references to CUE fields.
//
// Version must be an explicitly supplied string matching sef.Version. Identity
// and revision default to empty strings, and omitted definition collections are
// empty. Structural string fields accept CUE strings or byte literals. Byte
// literals preserve data that is not valid UTF-8, including embedded code.
//
// Parse evaluates and checks the information schema, not solution semantics.
// It does not look up descriptors, inspect I/O, or call Make. It uses no CUE
// module loader; imports must be built-ins available to the CUE context.
// This encoding is experimental and makes no compatibility promise.
package cue

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strings"

	cueapi "cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
)

//go:embed schema.cue
var schema string

// File retains a source syntax tree separately from its evaluated artifact.
// The tree is private and formatting does not rewrite it or the artifact.
type File struct {
	source   *ast.File
	artifact solution.Artifact
}

// Artifact returns an independent structural snapshot of the evaluated data.
// Editing this snapshot does not change the source or later formatting.
func (f *File) Artifact() solution.Artifact {
	artifact := f.artifact
	artifact.Definition.Modules = slices.Clone(artifact.Definition.Modules)
	artifact.Definition.Nodes = slices.Clone(artifact.Definition.Nodes)
	artifact.Definition.Bindings = slices.Clone(artifact.Definition.Bindings)
	return artifact
}

// Parse reads one CUE file, evaluates it against the embedded SEF schema, and
// requires a concrete artifact. It returns nil on failure. CUE failures retain
// their source positions and underlying errors for diagnostic inspection.
// It leaves r open and preserves reader error causes.
func Parse(filename string, r io.Reader) (*File, error) {
	source, err := parseSource(filename, r)
	if err != nil {
		return nil, err
	}
	ctx := cuecontext.New()
	value := ctx.BuildFile(source)
	if err := value.Err(); err != nil {
		return nil, diagnostic(err)
	}
	versionValue := value.LookupPath(cueapi.ParsePath("version"))
	if !versionValue.Exists() {
		return nil, fmt.Errorf("%s: %w: version field is required", filename, sef.ErrVersion)
	}
	version, err := versionValue.String()
	if err != nil {
		return nil, diagnostic(err)
	}
	if version != sef.Version {
		return nil, fmt.Errorf("%s: %w: %q", filename, sef.ErrVersion, version)
	}
	constraint := ctx.CompileString(schema, cueapi.Filename("sef/cue/schema.cue")).LookupPath(cueapi.ParsePath("#Artifact"))
	value = value.Unify(constraint)
	if err := value.Validate(cueapi.Concrete(true)); err != nil {
		return nil, diagnostic(err)
	}
	var decoded wireArtifact
	if err := value.Decode(&decoded); err != nil {
		return nil, diagnostic(err)
	}
	return &File{source: source, artifact: decoded.artifact()}, nil
}

// Format writes the original source in canonical CUE format, retaining comments,
// expressions, and definitions. It does not serialize an edited Artifact
// snapshot; use Encode for evaluated artifact data. It leaves w open.
func Format(w io.Writer, file *File) error {
	if file == nil || file.source == nil {
		return fmt.Errorf("sef/cue: no source file to format")
	}
	return writeSource(w, file.source)
}

// FormatSource formats syntactically valid CUE without evaluating it or requiring
// a concrete SEF artifact. This permits formatting incomplete constraints and
// schema-invalid source during authoring. It leaves both streams open.
func FormatSource(w io.Writer, filename string, r io.Reader) error {
	source, err := parseSource(filename, r)
	if err != nil {
		return err
	}
	return writeSource(w, source)
}

func parseSource(filename string, r io.Reader) (*ast.File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%s: read CUE: %w", filename, err)
	}
	source, err := parser.ParseFile(filename, data, parser.ParseComments, parser.AllErrors)
	if err != nil {
		return nil, diagnostic(err)
	}
	return source, nil
}

func writeSource(w io.Writer, source *ast.File) error {
	data, err := format.Node(source)
	if err != nil {
		return diagnostic(err)
	}
	_, err = io.Copy(w, bytes.NewReader(data))
	return err
}

type diagnosticError struct{ cause error }

func (e *diagnosticError) Error() string {
	return strings.TrimSuffix(cueerrors.Details(e.cause, nil), "\n")
}

func (e *diagnosticError) Unwrap() error { return e.cause }

func diagnostic(err error) error { return &diagnosticError{cause: err} }
