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
	"github.com/IamRitwik/reading-tracker/internal/observability"
	"github.com/IamRitwik/reading-tracker/internal/repository/sqlite"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unkwnown"
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

	// 1. Logging - one call, ev-driven.
	logger := observability.NewLogger(observability.LoggerConfigFromEnv())
	slog.SetDefault(logger)

	logger.InfoContext(ctx, "reading-tracker starting",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("built", date),
	)

	// 2. Data Directory.
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

	// 3. Open DB.
	database, err := db.Open(ctx, dbPath)
	if err != nil {
		return err
	}
	defer database.Close()
	logger.InfoContext(ctx, "database opened", slog.String("path", dbPath))

	// 4. Metrics registry.
	metrics := observability.NewMetrics()

	// 5. Optionally expose /metrics. Set RT_METRICS_ADDR=:2112 to enable.
	if addr := os.Getenv("RT_METRICS_ADDR"); addr != "" {
		observability.StartMetricsServer(
			ctx, addr, metrics,
			logger.With(slog.String("component", "metrics")),
		)
	}

	// 6. Compose the repo stack.
	baseRepo := sqlite.NewBookRepo(database)
	repo := observability.NewInstrumentedBookRepo(
		baseRepo,
		metrics,
		logger.With(slog.String("component", "repo")),
	)

	// 7. Service on top.
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

	logger.InfoContext(ctx, "waiting for shutdown signal")
	<-ctx.Done()

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
			Title:       "Learning Go",
			Author:      "Jon Bodner",
			Category:    "Programming",
			Description: "An idiomatic approach to real world Go programming",
			TotalPages:  353,
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
