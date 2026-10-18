// Command sef formats, converts, checks, and inspects solution artifacts.
package main

import (
	"github.com/modern-engineering/prototype/internal/seftool"
	"os"
)

func main() {
	command := seftool.Command{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
	os.Exit(command.Run(os.Args[1:]))
}
