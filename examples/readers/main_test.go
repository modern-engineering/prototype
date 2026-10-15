package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
)

// Your command can read an eager input during configuration and let the runner
// consume a borrowed input later, while owning both files itself.
func Example() {
	if err := run(context.Background(), fixtures); err != nil {
		fmt.Println(err)
	}
	// Output:
	// eager: Hello, Ada!
	// borrowed: Hello, Ada!
}

// The eager source closes before the borrowed source opens, and the borrowed
// source remains readable until the runner has consumed it.
func TestFilesFollowConsumptionLifetime(t *testing.T) {
	files := &trackedFS{t: t}
	if err := run(t.Context(), files); err != nil {
		t.Fatal(err)
	}
	if len(files.opened) != 2 {
		t.Fatalf("opened %d files, want 2", len(files.opened))
	}
	for i, file := range files.opened {
		if file.reads == 0 || file.closes != 1 {
			t.Errorf("file %d: %d reads, %d closes; want reads and one close", i+1, file.reads, file.closes)
		}
	}
}

// Errors while configuring or running the job must not strand an open file.
// A closing error must also survive when it follows a read failure.
func TestFilesCloseOnFailure(t *testing.T) {
	readError := errors.New("read failed")
	closeError := errors.New("close failed")
	for _, test := range []struct {
		name      string
		readFile  int
		closeFile int
	}{
		{name: "eager read", readFile: 1},
		{name: "borrowed read", readFile: 2},
		{name: "eager close", closeFile: 1},
		{name: "borrowed close", closeFile: 2},
		{name: "read and close", readFile: 2, closeFile: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			files := &trackedFS{t: t, readFile: test.readFile, readError: readError, closeFile: test.closeFile, closeError: closeError}
			err := run(t.Context(), files)
			if test.readFile != 0 && !errors.Is(err, readError) {
				t.Errorf("run error = %v, want read failure", err)
			}
			if test.closeFile != 0 && !errors.Is(err, closeError) {
				t.Errorf("run error = %v, want close failure", err)
			}
			for i, file := range files.opened {
				if file.closes != 1 {
					t.Errorf("file %d closed %d times, want once", i+1, file.closes)
				}
			}
		})
	}
}

// Cancellation before configuration acquires nothing. Cancellation after
// configuration still closes the file the runner would have borrowed.
func TestCancellationReleasesFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	files := &trackedFS{t: t}
	if err := run(ctx, files); !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v, want cancellation", err)
	}
	if len(files.opened) != 0 {
		t.Fatalf("opened %d files with a canceled context", len(files.opened))
	}

	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	files = &trackedFS{t: t, cancelAfterOpen: cancel}
	if err := run(ctx, files); !errors.Is(err, context.Canceled) {
		t.Fatalf("run error = %v, want cancellation", err)
	}
	if len(files.opened) != 2 {
		t.Fatalf("opened %d files, want 2", len(files.opened))
	}
	for i, file := range files.opened {
		if file.closes != 1 {
			t.Errorf("file %d closed %d times, want once", i+1, file.closes)
		}
	}
}

type trackedFS struct {
	t               *testing.T
	opened          []*trackedFile
	readFile        int
	readError       error
	closeFile       int
	closeError      error
	cancelAfterOpen context.CancelFunc
}

func (f *trackedFS) Open(name string) (fs.File, error) {
	if len(f.opened) != 0 && f.opened[0].closes != 1 {
		f.t.Error("borrowed source opened before eager source closed")
	}
	file, err := fixtures.Open(name)
	if err != nil {
		return nil, err
	}
	tracked := &trackedFile{File: file}
	f.opened = append(f.opened, tracked)
	if len(f.opened) == f.readFile {
		tracked.readError = f.readError
	}
	if len(f.opened) == f.closeFile {
		tracked.closeError = f.closeError
	}
	if len(f.opened) == 2 && f.cancelAfterOpen != nil {
		f.cancelAfterOpen()
	}
	return tracked, nil
}

type trackedFile struct {
	fs.File
	reads      int
	closes     int
	readError  error
	closeError error
}

func (f *trackedFile) Read(p []byte) (int, error) {
	f.reads++
	if f.closes != 0 {
		return 0, fs.ErrClosed
	}
	if f.readError != nil {
		return 0, f.readError
	}
	return f.File.Read(p)
}

func (f *trackedFile) Close() error {
	f.closes++
	return errors.Join(f.File.Close(), f.closeError)
}
