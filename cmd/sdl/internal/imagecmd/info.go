// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package imagecmd

import (
	"bytes"
	"fmt"
	"io"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/solution/image"
)

// cmdInfo is the sdl image info command.
var cmdInfo = &base.Command{
	UsageLine: "sdl image info [image]",
	Short:     "print an image's identity and provenance",
	Long: `Info prints the identity and provenance of the named desired-state
image — read from the file, or from standard input when no file is
given — one tab-separated fact per line, the way go version -m reports
a binary: the solution name, the generation, the format, then the
governance block's build settings (key=value) and one line per
compiled unit with the SHA-256 digest of its source (compare with
shasum -a 256), and finally one line per pinned catalogue package with
its element count. An image predating the governance block simply
prints no setting or unit lines.

Exit status 0 means the report was printed; 2 reports an unreadable or
undecodable image.`,
}

func init() {
	cmdInfo.Run = queryRunner(info)
}

// info renders the identity-and-provenance report: header facts, the
// governance block when the image carries one, and the catalogue pin
// summary. Lines buffer first so the report reaches w in one checked
// write.
func info(img *image.Image, w io.Writer) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "solution\t%s\n", img.Solution)
	fmt.Fprintf(&b, "generation\t%d\n", img.Generation)
	fmt.Fprintf(&b, "format\t%s\n", img.Format)
	if img.Build != nil {
		for _, s := range img.Build.Settings {
			fmt.Fprintf(&b, "setting\t%s=%s\n", s.Key, s.Value)
		}
		for _, u := range img.Build.Units {
			fmt.Fprintf(&b, "unit\t%s\t%s\n", u.Name, u.SHA256)
		}
	}
	for _, pkg := range img.Catalogue {
		fmt.Fprintf(&b, "catalogue\t%s\t%s\t%s\n", pkg.Path, pkg.Name, countNoun(len(pkg.Elements), "element"))
	}
	_, err := w.Write(b.Bytes())
	return err
}

// countNoun renders a count with its pluralized noun.
func countNoun(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
