// Command readers shows how your loading code can supply file content to a
// descriptor while choosing when each file can close.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"strings"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// Embed the sample input so the command and its tests run from any directory.
// Your loading code can supply another fs.FS, including os.DirFS.
//
//go:embed testdata/message.txt
var fixtures embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, fixtures); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Inspect reads the same content through an eager input and a borrowed input.
// Keep acquisitions in your loading code: Make only allocates and binds state.
var Inspect = job.Descriptor{
	Name: "inspect",
	MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
		var eager []byte
		borrowed := new(borrowedText)
		var inputs input.List
		inputs.BytesVar(&eager, "eager", "content read during configuration")
		inputs.Var(borrowed, "borrowed", "content read during execution")
		run := program.RunnerFunc(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if borrowed.reader == nil {
				return errors.New("borrowed content is missing")
			}
			if _, err := fmt.Printf("eager: %s", eager); err != nil {
				return err
			}
			body, err := io.ReadAll(borrowed.reader)
			if err != nil {
				return err
			}
			_, err = fmt.Printf("borrowed: %s", body)
			return err
		})
		return run, &inputs, nil
	},
}

type borrowedText struct {
	reader io.Reader
}

func (v *borrowedText) Set(text string) error {
	v.reader = strings.NewReader(text)
	return nil
}

// BorrowReader is a capability agreed between this descriptor and its loading
// code. If you introduce one in your organization, document its lifetime:
// this method retains the reader, and the caller keeps it open through Run.
func (v *borrowedText) BorrowReader(reader io.Reader) error {
	v.reader = reader
	return nil
}

func run(ctx context.Context, files fs.FS) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	runner, inputs, _ := Inspect.Make()
	eager, _ := inputs.Lookup("eager")
	borrowed, _ := inputs.Lookup("borrowed")

	file, err := files.Open("testdata/message.txt")
	if err != nil {
		return err
	}
	// An eager input lets your loading code release the file immediately.
	_, readErr := eager.Value.(io.ReaderFrom).ReadFrom(struct{ io.Reader }{file})
	if err := errors.Join(readErr, file.Close()); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err = files.Open("testdata/message.txt")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	// Keep a borrowed source open until Run returns, including on failure.
	// Pass only io.Reader so the descriptor cannot close your file.
	borrower := borrowed.Value.(interface{ BorrowReader(io.Reader) error })
	if err := borrower.BorrowReader(struct{ io.Reader }{file}); err != nil {
		return err
	}
	return runner.Run(ctx)
}
