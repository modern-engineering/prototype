// Command sef authors, inspects, and runs solution artifacts.
package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/modern-engineering/prototype/internal/seftool"
)

func main() {
	signals := []os.Signal{os.Interrupt}
	running := len(os.Args) > 1 && os.Args[1] == "playground"
	if running {
		signals = append(signals, syscall.SIGTERM)
	}
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	input := io.Reader(os.Stdin)
	if running {
		// Keep artifact input cancelable even when inherited stdin is a
		// blocking file. These forwarding goroutines belong to this process.
		reader, writer := io.Pipe()
		input = reader
		go func() {
			_, err := io.Copy(writer, os.Stdin)
			_ = writer.CloseWithError(err)
		}()
		go func() {
			<-ctx.Done()
			_ = reader.CloseWithError(ctx.Err())
		}()
	}
	command := seftool.Command{Context: ctx, Stdin: input, Stdout: os.Stdout, Stderr: os.Stderr}
	status := command.Run(os.Args[1:])
	stop()
	os.Exit(status)
}
