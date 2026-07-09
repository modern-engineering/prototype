// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package imagecmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/modern-engineering/prototype/cmd/sdl/internal/base"
	"github.com/modern-engineering/prototype/solution/image"
)

// cmdRecords is the sdl image records command.
var cmdRecords = &base.Command{
	UsageLine: "sdl image records [image]",
	Short:     "list an image's records as a table",
	Long: `Records prints one line per record of the named desired-state image —
read from the file, or from standard input when no file is given — as
an aligned table of the record's verb, its provision kind (blank for
deploys), its element as the import-path-qualified name it pins in the
catalogue, and its instance name, in image (statement) order.

Exit status 0 means the table was printed; 2 reports an unreadable or
undecodable image.`,
}

// cmdSymbols is the sdl image symbols command.
var cmdSymbols = &base.Command{
	UsageLine: "sdl image symbols [image]",
	Short:     "list an image's symbol table as a table",
	Long: `Symbols prints one line per row of the named desired-state image's
symbol table — read from the file, or from standard input when no file
is given — as an aligned table of the symbol's name, its linkage class,
and its type or value: an extern shows the import-path-qualified symbol
type the site must bind against, a var its compile-bound literal in
canonical SDL spelling.

Exit status 0 means the table was printed; 2 reports an unreadable or
undecodable image.`,
}

func init() {
	cmdRecords.Run = queryRunner(records)
	cmdSymbols.Run = queryRunner(symbols)
}

// queryRunner adapts one read-only image query into a command Run:
// decode the image argument (or standard input) and render onto
// standard output.
func queryRunner(query func(img *image.Image, w io.Writer) error) func(context.Context, *base.Command, []string) error {
	return func(ctx context.Context, cmd *base.Command, args []string) error {
		var in io.Reader = os.Stdin
		switch len(args) {
		case 0:
		case 1:
			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			// The image is only read; a close fault has nothing to add
			// to the decode's own verdict.
			defer func() { _ = f.Close() }()
			in = f
		default:
			return &base.UsageError{Msg: fmt.Sprintf("image %s takes at most one image argument, got %d", cmd.Name(), len(args))}
		}
		img, err := image.Decode(in)
		if err != nil {
			return err
		}
		return query(img, os.Stdout)
	}
}

// table starts one aligned output table with its header row. Row
// writes land in the tabwriter's buffer and are deliberately
// unchecked: Flush performs the real write and carries the verdict.
func table(w io.Writer, columns string) *tabwriter.Writer {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, columns)
	return tw
}

// records renders the records table: VERB KIND ELEMENT NAME, one row
// per record in image order.
func records(img *image.Image, w io.Writer) error {
	tw := table(w, "VERB\tKIND\tELEMENT\tNAME")
	for _, rec := range img.Records {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s.%s\t%s\n", rec.Verb, rec.Kind, rec.Element.Package, rec.Element.Name, rec.Name)
	}
	return tw.Flush()
}

// symbols renders the symbol table: NAME CLASS TYPE/VALUE, one row per
// symbol in image order.
func symbols(img *image.Image, w io.Writer) error {
	tw := table(w, "NAME\tCLASS\tTYPE/VALUE")
	for _, def := range img.Symbols {
		var tv string
		switch def.Class {
		case image.ClassExtern:
			if def.Type == nil {
				return fmt.Errorf("extern symbol %s has no type", def.Name)
			}
			tv = def.Type.Package + "." + def.Type.Name
		case image.ClassVar:
			text, err := valueText(def.Value)
			if err != nil {
				return fmt.Errorf("var symbol %s: %w", def.Name, err)
			}
			tv = text
		default:
			return fmt.Errorf("symbol %s: unknown class %q", def.Name, def.Class)
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", def.Name, def.Class, tv)
	}
	return tw.Flush()
}

// valueText renders a symbol-table literal in its canonical SDL
// spelling. Symbol values are the four literal kinds; anything else is
// a malformed image.
func valueText(v *image.Value) (string, error) {
	if v == nil {
		return "", fmt.Errorf("has no value")
	}
	switch v.Kind {
	case image.KindString:
		return strconv.Quote(v.Str), nil
	case image.KindInt:
		return strconv.FormatInt(v.Int, 10), nil
	case image.KindBool:
		return strconv.FormatBool(v.Bool), nil
	case image.KindDuration:
		return v.Dur.String(), nil
	}
	return "", fmt.Errorf("cannot render value kind %q", v.Kind)
}
