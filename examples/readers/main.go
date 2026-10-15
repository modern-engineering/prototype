// Command readers demonstrates eager consumption and borrowed readers with
// resource lifetimes owned by the command that opens them.
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

//go:embed message.txt
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
// Make only binds state; the command opens both files during configuration.
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

// BorrowReader retains the handle without reading. The caller keeps it open
// through Run and closes it afterward, including on execution failure.
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

	file, err := files.Open("message.txt")
	if err != nil {
		return err
	}
	// Eager consumption needs the file only while ReadFrom is running.
	_, readErr := eager.Value.(io.ReaderFrom).ReadFrom(struct{ io.Reader }{file})
	if err := errors.Join(readErr, file.Close()); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err = files.Open("message.txt")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	// The borrowed handle stays usable until Run returns, even on failure.
	// Only the loading code receives the file's closing capability.
	borrower := borrowed.Value.(interface{ BorrowReader(io.Reader) error })
	if err := borrower.BorrowReader(struct{ io.Reader }{file}); err != nil {
		return err
	}
	return runner.Run(ctx)
}
