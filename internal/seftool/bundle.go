package seftool

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	sefbundle "github.com/modern-engineering/prototype/sef/bundle"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
)

// Directory inputs use their root manifest. Individual files are standalone:
// their meaning does not change when copied beneath a different Go module.
func (c Command) readArtifact(path, format, member string) (solution.Artifact, string, error) {
	var bundle solution.Bundle
	var directory string
	isBundle := strings.HasSuffix(path, ".tar") || strings.HasSuffix(path, ".tar.gz")
	if path != "-" {
		info, err := os.Stat(path)
		if err != nil {
			return solution.Artifact{}, "", err
		}
		isBundle = isBundle || info.IsDir()
		if info.IsDir() {
			directory, err = filepath.Abs(path)
			if err != nil {
				return solution.Artifact{}, "", err
			}
		} else {
			directory, err = filepath.Abs(filepath.Dir(path))
			if err != nil {
				return solution.Artifact{}, "", err
			}
		}
	}
	if !isBundle {
		if member != "" {
			return solution.Artifact{}, "", errors.New("-artifact requires a bundle directory or archive")
		}
		data, err := c.read(path)
		if err != nil {
			return solution.Artifact{}, "", err
		}
		artifact, err := decode(path, format, data)
		return artifact, directory, err
	}
	// An archive has no local source directory for development replacements.
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		directory = ""
	}
	bundle, err := readBundle(path)
	if err != nil {
		return solution.Artifact{}, "", err
	}
	if member == "" {
		names, err := artifactNames(bundle.Files)
		if err != nil {
			return solution.Artifact{}, "", err
		}
		if len(names) != 1 {
			return solution.Artifact{}, "", fmt.Errorf("bundle has %d artifacts; select one with -artifact", len(names))
		}
		member = names[0]
	}
	if !fs.ValidPath(member) {
		return solution.Artifact{}, "", fmt.Errorf("invalid bundle artifact path %q", member)
	}
	data, err := fs.ReadFile(bundle.Files, member)
	if err != nil {
		return solution.Artifact{}, "", err
	}
	artifact, err := decode(member, format, data)
	if err != nil {
		return solution.Artifact{}, "", err
	}
	artifact.Module = bundle.Module.Clone()
	return artifact, directory, nil
}

func readBundle(path string) (solution.Bundle, error) {
	info, err := os.Stat(path)
	if err != nil {
		return solution.Bundle{}, err
	}
	if info.IsDir() {
		return solution.OpenBundle(os.DirFS(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return solution.Bundle{}, err
	}
	defer func() { _ = file.Close() }()
	switch {
	case strings.HasSuffix(path, ".tar.gz"):
		return sefbundle.DecodeTarGzip(file)
	case strings.HasSuffix(path, ".tar"):
		return sefbundle.DecodeTar(file)
	default:
		return solution.Bundle{}, errors.New("bundle input must be a directory, .tar, or .tar.gz")
	}
}

func artifactNames(files fs.FS) ([]string, error) {
	var names []string
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && (strings.HasSuffix(path, ".sef") || strings.HasSuffix(path, ".cue")) {
			names = append(names, path)
		}
		return nil
	})
	return names, err
}

func (c Command) bundleCommand(command string, args []string) (int, error) {
	if command == "mod" {
		if len(args) == 0 || args[0] != "tidy" {
			return 2, errors.New("usage: sef mod tidy [-dev] [directory]")
		}
		return c.tidy(args[1:])
	}
	flags := flag.NewFlagSet("sef "+command, flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	output := ""
	if command == "pack" {
		flags.StringVar(&output, "o", "", "new .tar or .tar.gz archive")
	} else {
		flags.StringVar(&output, "outdir", "", "new directory for the unpacked bundle")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, nil
	}
	if flags.NArg() != 1 || output == "" {
		return 2, fmt.Errorf("%s requires one input and a named output", command)
	}
	input := flags.Arg(0)
	if command == "pack" && !strings.HasSuffix(output, ".tar") && !strings.HasSuffix(output, ".tar.gz") {
		return 2, errors.New("pack output must end in .tar or .tar.gz")
	}
	if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return 1, err
		}
		return 1, fmt.Errorf("output already exists: %s", output)
	}
	bundle, err := readBundle(input)
	if err != nil {
		return 1, err
	}
	if command == "pack" {
		// Stage the archive before creating its output, including when the
		// destination is inside the source directory.
		var data bytes.Buffer
		if strings.HasSuffix(output, ".tar.gz") {
			err = sefbundle.EncodeTarGzip(&data, bundle)
		} else {
			err = sefbundle.EncodeTar(&data, bundle)
		}
		if err != nil {
			return 1, err
		}
		file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return 1, err
		}
		_, writeErr := file.Write(data.Bytes())
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return 1, errors.Join(err, os.Remove(output))
		}
		return 0, nil
	}
	if err := os.Mkdir(output, 0o700); err != nil {
		return 1, err
	}
	if err := os.CopyFS(output, bundle.Files); err != nil {
		return 1, errors.Join(err, os.RemoveAll(output))
	}
	return 0, nil
}

func (c Command) tidy(args []string) (int, error) {
	flags := flag.NewFlagSet("sef mod tidy", flag.ContinueOnError)
	flags.SetOutput(c.Stderr)
	dev := flags.Bool("dev", false, "enable local replacements and development workspace modules")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, nil
		}
		return 2, nil
	}
	if flags.NArg() > 1 {
		return 2, errors.New("mod tidy expects at most one directory")
	}
	directory := "."
	if flags.NArg() == 1 {
		directory = flags.Arg(0)
	}
	bundle, err := solution.OpenBundle(os.DirFS(directory))
	if err != nil {
		return 1, err
	}
	names, err := artifactNames(bundle.Files)
	if err != nil {
		return 1, err
	}
	if len(names) == 0 {
		return 1, errors.New("bundle contains no solution artifacts")
	}
	artifact := solution.Artifact{Module: bundle.Module.Clone()}
	// The generated package needs each referenced descriptor, independently
	// of whether two solutions use the same occurrence names.
	seen := make(map[solution.Symbol]bool)
	for _, name := range names {
		data, err := fs.ReadFile(bundle.Files, name)
		if err != nil {
			return 1, err
		}
		member, err := decode(name, "auto", data)
		if err != nil {
			return 1, err
		}
		for _, node := range member.Definition.Nodes {
			if !seen[node.Definition] {
				seen[node.Definition] = true
				artifact.Definition.Nodes = append(artifact.Definition.Nodes, solution.Node{Name: "descriptor" + strconv.Itoa(len(seen)), Definition: node.Definition})
			}
		}
	}
	prepared, err := build.Tidy(c.Context, artifact, build.Options{Dev: *dev, Dir: directory})
	if err != nil {
		return 1, err
	}
	// Preserve authoring comments through modfile's editing API. The portable
	// artifact context intentionally carries only interpreted directives.
	original, err := fs.ReadFile(bundle.Files, "go.mod")
	if err != nil {
		return 1, err
	}
	manifest, err := modfile.Parse("go.mod", original, nil)
	if err != nil {
		return 1, err
	}
	var requires []*modfile.Require
	for _, requirement := range prepared.Require {
		requires = append(requires, &modfile.Require{Mod: module.Version{Path: requirement.Path, Version: requirement.Version}, Indirect: requirement.Indirect})
	}
	manifest.SetRequire(requires)
	if prepared.Go != "" {
		if err := manifest.AddGoStmt(prepared.Go); err != nil {
			return 1, err
		}
	}
	if prepared.Toolchain == "" {
		manifest.DropToolchainStmt()
	} else if err := manifest.AddToolchainStmt(prepared.Toolchain); err != nil {
		return 1, err
	}
	manifest.Cleanup()
	data, err := manifest.Format()
	if err != nil {
		return 1, err
	}
	var sums bytes.Buffer
	if err := prepared.Write(io.Discard, &sums); err != nil {
		return 1, err
	}
	if err := c.write(filepath.Join(directory, "go.mod"), data); err != nil {
		return 1, err
	}
	if err := c.write(filepath.Join(directory, "go.sum"), sums.Bytes()); err != nil {
		return 1, err
	}
	return 0, nil
}
