package seftool

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"

	"github.com/modern-engineering/prototype/sef"
	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/build"
	"github.com/modern-engineering/prototype/solution/link"
)

func toolkitVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	const module = "github.com/modern-engineering/prototype"
	version := ""
	if info.Main.Path == module {
		version = info.Main.Version
	}
	for _, dependency := range info.Deps {
		if dependency.Path == module {
			version = dependency.Version
		}
	}
	if version == "(devel)" {
		return ""
	}
	return version
}

func (c Command) generated(name string, artifact solution.Artifact, inputs bool, output string, options build.Options, dev bool) (status int, err error) {
	if err := link.CheckStructure(artifact.Definition); err != nil {
		return 1, err
	}
	var materialized bytes.Buffer
	if err := sef.EncodeGob(&materialized, artifact); err != nil {
		return 1, err
	}
	definition := artifact.Definition
	if dev {
		definition, options, err = development(c.Context, definition, options)
		if err != nil {
			return 1, err
		}
	}
	result, err := build.Host(c.Context, definition, options)
	if result != nil {
		if options.KeepWork {
			_, _ = fmt.Fprintf(c.Stderr, "WORK=%s\n", result.WorkDir)
		} else {
			defer func() {
				err = errors.Join(err, result.Close())
				if err != nil && status == 0 {
					status = 1
				}
			}()
		}
	}
	if err != nil {
		return 1, err
	}
	if name == "build" {
		if err := saveExecutable(output, result.Binary); err != nil {
			return 1, err
		}
		return 0, nil
	}
	args := []string{name}
	if inputs {
		args = append(args, "-inputs")
	}
	if err := result.Run(c.Context, args, &materialized, c.Stdout, c.Stderr); err != nil {
		if exited, ok := errors.AsType[*exec.ExitError](err); ok && c.Context.Err() == nil && exited.ExitCode() > 0 {
			return exited.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func saveExecutable(path, source string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	if err := errors.Join(copyErr, output.Close()); err != nil {
		return errors.Join(err, os.Remove(path))
	}
	return nil
}
