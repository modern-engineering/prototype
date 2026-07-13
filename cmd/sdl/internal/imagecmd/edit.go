// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package imagecmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/solution/image"
)

// cmdEdit is the sdl image edit command.
var cmdEdit = &base.Command{
	UsageLine: "sdl image edit [-generation N] <image>",
	Short:     "amend an image file in place",
	Long: `Edit amends the named desired-state image file in place: it decodes
the image, applies the requested edits, and atomically replaces the
file with the canonical re-encoding, so a failure at any point leaves
the original untouched.

The -generation flag stamps the image's generation. The deploying
pipeline owns that counter: it bumps the generation whenever it ships
changed desired content, and consumers treat two images carrying one
generation but different content as the pipeline's error.

Exit status 0 means the file was rewritten; 2 reports a missing edit
flag, an unreadable or undecodable image, or a failed rewrite.`,
}

var flagGeneration int64

func init() {
	cmdEdit.Run = runEdit // break init cycle: Run references cmdEdit's flags
	cmdEdit.Flag.Int64Var(&flagGeneration, "generation", 0, "stamp the image with generation `N`")
}

// runEdit amends the file in place and prints nothing, so the streams
// go unused; faults report through main's error translation.
func runEdit(ctx context.Context, _ base.Streams, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return &base.UsageError{Msg: fmt.Sprintf("image edit takes exactly one image argument, got %d", len(args))}
	}
	switch {
	case flagGeneration == 0:
		return &base.UsageError{Msg: "image edit: no edits specified (set -generation)"}
	case flagGeneration < 0:
		return &base.UsageError{Msg: "image edit: generation must be positive"}
	}
	return edit(args[0], flagGeneration)
}

// edit rewrites one image file with the amended generation. The
// replacement is atomic — decode, amend, re-encode into a temporary
// file beside the original, rename over it — so any fault on the way
// leaves the original file byte-identical.
func edit(path string, generation int64) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	img, derr := image.Decode(f)
	info, serr := f.Stat()
	if cerr := f.Close(); derr == nil && serr == nil && cerr != nil {
		return cerr
	}
	if derr != nil {
		return fmt.Errorf("%s: %w", path, derr)
	}
	if serr != nil {
		return serr
	}

	img.Generation = generation
	return writeFile(path, img, info.Mode().Perm())
}

// writeFile atomically replaces path with the canonical encoding of
// img: the bytes land in a temporary file in the same directory, take
// the original permissions, and move over the original in one rename
// (the sdl fmt -w precedent), so a crash never leaves a half-written
// image.
func writeFile(path string, img *image.Image, perm fs.FileMode) error {
	dir, name := filepath.Split(path)
	tmp, err := os.CreateTemp(dir, name+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	werr := img.Encode(tmp)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(tmpName, perm)
	}
	if werr == nil {
		werr = os.Rename(tmpName, path)
	}
	if werr != nil {
		_ = os.Remove(tmpName) // the write fault is the one worth reporting
		return fmt.Errorf("rewriting %s: %v", path, werr)
	}
	return nil
}
