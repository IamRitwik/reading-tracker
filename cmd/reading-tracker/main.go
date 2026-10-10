package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/IamRitwik/reading-tracker/internal/db"
	"github.com/IamRitwik/reading-tracker/internal/repository/sqlite"
	"github.com/IamRitwik/reading-tracker/internal/service"
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

	dataDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("user config dir: %w", err)
	}

	appDir := filepath.Join(dataDir, "reading-tracker")
	err = os.MkdirAll(appDir, 0o755)
	if err != nil {
		return fmt.Errorf("create app dir: %w", err)
	}

	dbPath := filepath.Join(appDir, "reading.db")
	logger.InfoContext(ctx, "opening database", slog.String("path", dbPath))

	database, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer database.Close()

	// Per-component child loggers. Any log emitted through this logger will
	// carry component="service". In production with a log aggregator, this
	// is how you filter "show me only service errors."
	repo := sqlite.NewBookRepo(database)
	svc := service.NewBookService(
		repo,
		logger.With(slog.String("component", "service")),
		2*time.Second,
	)

	// Seed a demo book on first run so there's something to see.
	if err := seedIfEmpty(ctx, svc, logger); err != nil {
		return err
	}

	books, err := svc.List(ctx, service.ListInput{})
	if err != nil {
		return fmt.Errorf("list books: %w", err)
	}
	logger.InfoContext(ctx, "library loaded", slog.Int("count", len(books)))

	fmt.Println()

	for _, b := range books {
		fmt.Printf("  [%d] %-40s — %-25s (%d/%d pages, %5.1f%%, %s)\n",
			b.ID, b.Title, b.Author, b.PagesRead, b.TotalPages,
			b.ProgressPercent(), b.Status)
	}
	fmt.Println()

	return nil
}

func seedIfEmpty(ctx context.Context, svc *service.BookService, logger *slog.Logger) error {
	books, err := svc.List(ctx, service.ListInput{Limit: 1})
	if err != nil {
		return err
	}
	if len(books) > 0 {
		return nil
	}

	seed := []service.AddBookInput{
		{
			Title:       "Mastering Go",
			Author:      "Mihalis Tsoukalos",
			Category:    "Programming",
			Description: "Leverage Go's expertise for advanced utilities",
			TotalPages:  711,
		},
		{
			Title:       "Concurrency in GO",
			Author:      "Katherine Cox-Buday",
			Category:    "Programming",
			Description: "Tools & Techniques for developers",
			TotalPages:  224,
		},
	}
	var firstID int64
	for i, in := range seed {
		b, err := svc.Add(ctx, in)
		if err != nil {
			return fmt.Errorf("seed %q: %w", in.Title, err)
		}
		if i == 0 {
			firstID = b.ID
		}
	}

	if firstID > 0 {
		_, err := svc.UpdateProgress(ctx, firstID, 526)
		if err != nil {
			return fmt.Errorf("seed progress: %w", err)
		}
	}

	logger.InfoContext(ctx, "seeded demo books", slog.Int("count", len(seed)))
	return nil
}
