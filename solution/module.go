package solution

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// ErrModule means module files contain invalid or unsupported module intent.
var ErrModule = errors.New("invalid solution module context")

// ModuleContext carries authored Go module requirements and expected checksums.
// It is independent of source formatting and of a build's selected module list.
// Go is the minimum Go version; Toolchain is a suggested toolchain, not proof of
// the compiler used. A local replacement is relative to its owning project root,
// which is supplied separately by project tooling. It is not a portable source
// snapshot. The zero value represents absent module context.
type ModuleContext struct {
	Path      string
	Go        string
	Toolchain string
	Require   []Module
	Exclude   []Module
	Replace   []ModuleReplacement
	Sum       []ModuleSum
}

// ModuleReplacement substitutes New for Old. An empty Old.Version matches every
// version of Old.Path. An empty New.Version makes New.Path a local directory.
// Indirect has no meaning on either side of a replacement and must be false.
type ModuleReplacement struct{ Old, New Module }

// ModuleSum is one expected go.sum checksum. Version may end in /go.mod to name
// the module manifest rather than the module archive. Hash includes its algorithm
// prefix, such as h1:. Checksums authenticate content; they do not select versions.
type ModuleSum struct{ Path, Version, Hash string }

// Clone returns a module context with independent slices.
func (m ModuleContext) Clone() ModuleContext {
	m.Require = slices.Clone(m.Require)
	m.Exclude = slices.Clone(m.Exclude)
	m.Replace = slices.Clone(m.Replace)
	m.Sum = slices.Clone(m.Sum)
	return m
}

// ParseModule reads go.mod and optional go.sum through EOF without closing them.
// A nil goSum means no checksum file. The module directive is required. Supported
// directives are module, go, toolchain, require, exclude, and replace; others
// return ErrModule rather than being discarded. Comments and source layout are
// not retained. ParseModule performs no module resolution or network access.
// An error returns a zero context and preserves underlying reader/parser errors.
func ParseModule(goMod, goSum io.Reader) (ModuleContext, error) {
	if goMod == nil {
		return ModuleContext{}, fmt.Errorf("%w: missing go.mod reader", ErrModule)
	}
	data, err := io.ReadAll(goMod)
	if err != nil {
		return ModuleContext{}, fmt.Errorf("read go.mod: %w", err)
	}
	parsed, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return ModuleContext{}, errors.Join(ErrModule, err)
	}
	if parsed.Module == nil {
		return ModuleContext{}, fmt.Errorf("%w: missing module directive", ErrModule)
	}
	for _, directive := range []struct {
		name  string
		count int
	}{{"godebug", len(parsed.Godebug)}, {"retract", len(parsed.Retract)}, {"tool", len(parsed.Tool)}, {"ignore", len(parsed.Ignore)}} {
		if directive.count != 0 {
			return ModuleContext{}, fmt.Errorf("%w: unsupported %s directive", ErrModule, directive.name)
		}
	}
	m := ModuleContext{Path: parsed.Module.Mod.Path}
	if err := module.CheckImportPath(m.Path); err != nil {
		return ModuleContext{}, errors.Join(ErrModule, err)
	}
	if parsed.Go != nil {
		m.Go = parsed.Go.Version
	}
	if parsed.Toolchain != nil {
		m.Toolchain = parsed.Toolchain.Name
	}
	for _, r := range parsed.Require {
		m.Require = append(m.Require, Module{Path: r.Mod.Path, Version: r.Mod.Version, Indirect: r.Indirect})
	}
	for _, r := range parsed.Exclude {
		m.Exclude = append(m.Exclude, Module{Path: r.Mod.Path, Version: r.Mod.Version})
	}
	for _, r := range parsed.Replace {
		m.Replace = append(m.Replace, ModuleReplacement{Old: Module{Path: r.Old.Path, Version: r.Old.Version}, New: Module{Path: r.New.Path, Version: r.New.Version}})
	}
	if goSum != nil {
		data, err := io.ReadAll(goSum)
		if err != nil {
			return ModuleContext{}, fmt.Errorf("read go.sum: %w", err)
		}
		m.Sum, err = parseSums(data)
		if err != nil {
			return ModuleContext{}, err
		}
	}
	return m, nil
}

// Write emits canonical go.mod and go.sum content without closing either writer.
// A nil goSum omits checksum output. It validates the complete context before
// writing bytes, returning ErrModule for invalid data. Comments are not emitted;
// directive and checksum collections retain their order. Write performs no module
// resolution. Writer failures are preserved and may leave partial output.
func (m ModuleContext) Write(goMod, goSum io.Writer) error {
	if goMod == nil {
		return fmt.Errorf("%w: missing go.mod writer", ErrModule)
	}
	file := new(modfile.File)
	if err := file.AddModuleStmt(m.Path); err != nil {
		return errors.Join(ErrModule, err)
	}
	if m.Go != "" {
		if err := file.AddGoStmt(m.Go); err != nil {
			return errors.Join(ErrModule, err)
		}
	}
	if m.Toolchain != "" {
		if err := file.AddToolchainStmt(m.Toolchain); err != nil {
			return errors.Join(ErrModule, err)
		}
	}
	for _, r := range m.Require {
		file.AddNewRequire(r.Path, r.Version, r.Indirect)
	}
	for _, r := range m.Exclude {
		if r.Indirect {
			return fmt.Errorf("%w: indirect exclude", ErrModule)
		}
		if err := file.AddExclude(r.Path, r.Version); err != nil {
			return errors.Join(ErrModule, err)
		}
	}
	for _, r := range m.Replace {
		if r.Old.Indirect || r.New.Indirect {
			return fmt.Errorf("%w: indirect replacement", ErrModule)
		}
		if err := file.AddReplace(r.Old.Path, r.Old.Version, r.New.Path, r.New.Version); err != nil {
			return errors.Join(ErrModule, err)
		}
	}
	data, err := file.Format()
	if err != nil {
		return errors.Join(ErrModule, err)
	}
	var sums bytes.Buffer
	for _, sum := range m.Sum {
		fmt.Fprintf(&sums, "%s %s %s\n", sum.Path, sum.Version, sum.Hash)
	}
	parsed, err := ParseModule(bytes.NewReader(data), bytes.NewReader(sums.Bytes()))
	if err != nil {
		return err
	}
	if parsed.Path != m.Path || parsed.Go != m.Go || parsed.Toolchain != m.Toolchain || !slices.Equal(parsed.Require, m.Require) || !slices.Equal(parsed.Exclude, m.Exclude) || !slices.Equal(parsed.Replace, m.Replace) || !slices.Equal(parsed.Sum, m.Sum) {
		return fmt.Errorf("%w: module context cannot be represented without normalization or loss", ErrModule)
	}
	if _, err := bytes.NewReader(data).WriteTo(goMod); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}
	if goSum != nil {
		if _, err := sums.WriteTo(goSum); err != nil {
			return fmt.Errorf("write go.sum: %w", err)
		}
	}
	return nil
}

func parseSums(data []byte) ([]ModuleSum, error) {
	var sums []ModuleSum
	for i, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("%w: go.sum:%d: expected path, version, hash", ErrModule, i+1)
		}
		if err := module.Check(fields[0], strings.TrimSuffix(fields[1], "/go.mod")); err != nil {
			return nil, errors.Join(ErrModule, err)
		}
		algorithm, digest, ok := strings.Cut(fields[2], ":")
		if !ok || algorithm == "" || digest == "" {
			return nil, fmt.Errorf("%w: go.sum:%d: invalid checksum", ErrModule, i+1)
		}
		sums = append(sums, ModuleSum{Path: fields[0], Version: fields[1], Hash: fields[2]})
	}
	return sums, nil
}
