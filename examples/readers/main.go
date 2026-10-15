// Command readers demonstrates eager consumption and borrowed readers with
// resource lifetimes owned by the loader.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/modern-engineering/prototype/bind/input"
	"github.com/modern-engineering/prototype/examples/internal/loader"
	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

//go:embed message.txt
var fixtures embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Inspect reads the same content through an eager input and a borrowed input.
// Make only binds state; the loader acquires both readers during configuration.
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

// BorrowReader retains the handle without reading. The loader keeps it open
// through Run and closes it afterward, including on execution failure.
func (v *borrowedText) BorrowReader(reader io.Reader) error {
	v.reader = reader
	return nil
}

func run(ctx context.Context) (err error) {
	l := loader.Loader{FS: fixtures}
	i := l.Make(Inspect)
	defer func() { err = errors.Join(err, i.Close()) }()
	if err := i.ReadFile(ctx, "eager", "message.txt"); err != nil {
		return err
	}
	if err := i.ReadFile(ctx, "borrowed", "message.txt"); err != nil {
		return err
	}
	return i.Run(ctx)
}
