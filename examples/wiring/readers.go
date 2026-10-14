package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/modern-engineering/prototype/job"
	"github.com/modern-engineering/prototype/program"
)

// inspect reads the same content through an eager input and a borrowed input.
// Make only binds state; the loader acquires both readers during configuration.
func inspect(trace io.Writer) job.Descriptor {
	return job.Descriptor{
		Name: "inspect",
		MakeFunc: func() (program.Runner, program.Inputs, program.Outputs) {
			eager := &eagerText{trace: trace}
			borrowed := &borrowedText{trace: trace}
			inputs := program.InputList{
				{Field: program.Field{Name: "eager", Help: "content read during configuration"}, Value: eager},
				{Field: program.Field{Name: "borrowed", Help: "content read during execution"}, Value: borrowed},
			}
			run := program.RunnerFunc(func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if borrowed.reader == nil {
					return errors.New("borrowed content is missing")
				}
				if _, err := fmt.Fprintf(trace, "run: eager = %q\n", eager.text); err != nil {
					return err
				}
				body, err := io.ReadAll(borrowed.reader)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(trace, "run: borrowed = %q\n", body)
				return err
			})
			return run, inputs, program.OutputList(nil)
		},
	}
}

type eagerText struct {
	text  string
	trace io.Writer
}

func (v *eagerText) Set(text string) error { v.text = text; return nil }

// ReadFrom consumes the reader entirely during assignment. Only the copied
// text survives into Run; this input has no reader to retain or close.
func (v *eagerText) ReadFrom(reader io.Reader) (int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return int64(len(data)), err
	}
	v.text = string(data)
	_, err = fmt.Fprintln(v.trace, "consume eager")
	return int64(len(data)), err
}

type borrowedText struct {
	reader io.Reader
	trace  io.Writer
}

func (v *borrowedText) Set(text string) error {
	v.reader = strings.NewReader(text)
	return nil
}

// BorrowReader retains the handle without reading. The loader keeps it open
// through Run and closes it afterward, including on execution failure.
func (v *borrowedText) BorrowReader(reader io.Reader) error {
	if _, err := fmt.Fprintln(v.trace, "borrow reader"); err != nil {
		return err
	}
	v.reader = reader
	return nil
}

func runReaders(ctx context.Context, trace io.Writer) (err error) {
	// This loader stages an immutable snapshot in a file. The getter can expose
	// the same content as a string or open independent readers over that file.
	dir, err := os.MkdirTemp("", "wiring-readers-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	const text = "Hello, Ada!"
	path := filepath.Join(dir, "content.txt")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return err
	}
	source := &fileContent{path: path, text: text, trace: trace}
	l := loader{}
	i, err := l.load(ctx, inspect(trace), map[string]program.Getter{
		"eager": source, "borrowed": source,
	})
	if err != nil {
		return err
	}
	return i.Run(ctx)
}

// fileContent offers two representations of the loader's immutable snapshot.
// Each reader has its own file position and a separate lifetime.
type fileContent struct {
	path, text string
	trace      io.Writer
	opens      int
}

func (v *fileContent) Get() string { return v.text }

func (v *fileContent) Open(ctx context.Context) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(v.path)
	if err != nil {
		return nil, err
	}
	v.opens++
	if _, err := fmt.Fprintf(v.trace, "open %d\n", v.opens); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return &tracedFile{File: file, id: v.opens, trace: v.trace}, nil
}

type tracedFile struct {
	*os.File
	id    int
	trace io.Writer
}

func (f *tracedFile) Close() error {
	err := f.File.Close()
	_, traceErr := fmt.Fprintf(f.trace, "close %d\n", f.id)
	return errors.Join(err, traceErr)
}
