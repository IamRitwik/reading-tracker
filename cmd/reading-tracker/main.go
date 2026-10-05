package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// signal.NotifyContext returns a context that is cancelled
	// when the process receives SIGINT or SIGTERM.
	// This is THE modern Go pattern for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	err := run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Set up structured logging. For now, dev mode (text).
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	logger.InfoContext(ctx, "reading-tracker starting",
		slog.String("version", "0.0.1"),
	)

	// We'll add CLI/TUI wiring here in later phases.
	// For now, just confirm the skeleton works.
	logger.InfoContext(ctx, "reading-tracker exiting cleanly")
	return nil
}
