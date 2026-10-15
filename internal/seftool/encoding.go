package seftool

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/modern-engineering/prototype/sef"
	sefcue "github.com/modern-engineering/prototype/sef/cue"
	seftext "github.com/modern-engineering/prototype/sef/text"
	"github.com/modern-engineering/prototype/solution"
)

func validFormat(format string) bool {
	return format == "auto" || format == "text" || format == "json" || format == "gob" || format == "cue"
}

func formatFor(path string) string {
	switch {
	case strings.HasSuffix(path, ".cue.sef"), strings.HasSuffix(path, ".cue"):
		return "cue"
	case strings.HasSuffix(path, ".json.sef"):
		return "json"
	case strings.HasSuffix(path, ".gob.sef"):
		return "gob"
	default:
		return "text"
	}
}

func decode(path, format string, data []byte) (solution.Artifact, error) {
	if format == "auto" {
		format = formatFor(path)
	}
	reader := bytes.NewReader(data)
	switch format {
	case "cue":
		file, err := sefcue.Parse(path, reader)
		if err != nil {
			return solution.Artifact{}, err
		}
		return file.Artifact(), nil
	case "json", "gob":
		var artifact solution.Artifact
		var err error
		if format == "json" {
			artifact, err = sef.DecodeJSON(reader)
		} else {
			artifact, err = sef.DecodeGob(reader)
		}
		if err != nil {
			return solution.Artifact{}, fmt.Errorf("%s: %w", path, err)
		}
		return artifact, nil
	default:
		file, err := seftext.Parse(path, reader)
		if err != nil {
			return solution.Artifact{}, err
		}
		return file.Artifact(), nil
	}
}

func encode(w io.Writer, format string, artifact solution.Artifact) error {
	switch format {
	case "cue":
		m := artifact.Module
		if m.Path != "" || m.Go != "" || m.Toolchain != "" || len(m.Require)+len(m.Exclude)+len(m.Replace)+len(m.Sum) != 0 {
			return errors.New("CUE encoding does not yet carry artifact module context; use JSON, gob, or text for a standalone snapshot")
		}
		return sefcue.Encode(w, artifact)
	case "json":
		return sef.EncodeJSON(w, artifact)
	case "gob":
		return sef.EncodeGob(w, artifact)
	default:
		return seftext.Encode(w, artifact)
	}
}

func (c Command) read(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(c.Stdin)
	}
	return os.ReadFile(path)
}

func (c Command) write(path string, data []byte) error {
	if path == "-" {
		_, err := io.Copy(c.Stdout, bytes.NewReader(data))
		return err
	}
	mode := os.FileMode(0o600)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to replace symbolic link %q", path)
		}
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".sef-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	_, writeErr := file.Write(data)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
