package main

import (
	"log/slog"
	"os"
)

func init() {
	initLogging()
}

func initLogging() {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		//AddSource: true,
	})
	l := slog.New(h)
	slog.SetDefault(l)
}
