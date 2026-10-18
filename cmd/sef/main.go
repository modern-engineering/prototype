// Command sef formats, converts, checks, and inspects solution artifacts.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/modern-engineering/prototype/internal/seftool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	command := seftool.Command{Context: ctx, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	status := command.Run(os.Args[1:])
	stop()
	os.Exit(status)
}
