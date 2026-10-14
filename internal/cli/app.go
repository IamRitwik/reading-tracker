package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/IamRitwik/reading-tracker/internal/config"
	"github.com/IamRitwik/reading-tracker/internal/db"
	"github.com/IamRitwik/reading-tracker/internal/observability"
	"github.com/IamRitwik/reading-tracker/internal/repository"
	"github.com/IamRitwik/reading-tracker/internal/repository/sqlite"
	"github.com/IamRitwik/reading-tracker/internal/service"
	"github.com/spf13/cobra"
)

// App holds all long-lived dependencies for one CLI invocation.
// Build once in PersistentPreRunE, close once at the end of Execute.
type App struct {
	Config  config.Config
	Logger  *slog.Logger
	Metrics *observability.Metrics
	DB      *sql.DB
	Repo    repository.BookRepository
	Service *service.BookService

	metricsCancel context.CancelFunc
}

// Close releases resources. Safe to call multiple times / with nil fields.
func (a *App) Close() error {
	if a == nil {
		return nil
	}
	if a.metricsCancel != nil {
		a.metricsCancel()
		a.metricsCancel = nil
	}
	if a.DB != nil {
		err := a.DB.Close()
		a.DB = nil
		return err
	}
	return nil
}

// buildApp constructs the full dependency graph for a command.
func buildApp(cmd *cobra.Command) (*App, error) {
	cfg, err := config.Load(cmd)
	if err != nil {
		return nil, err
	}

	// Logger first: everything downstream wants it.
	logger := observability.NewLogger(observability.LoggerConfig{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
		Source: cfg.LogSource,
	})
	slog.SetDefault(logger)

	// Ensure the DB directory exists.
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	database, err := db.Open(cmd.Context(), cfg.DBPath)
	if err != nil {
		return nil, err
	}

	metrics := observability.NewMetrics()

	app := &App{
		Config:  cfg,
		Logger:  logger,
		Metrics: metrics,
		DB:      database,
	}

	// Optional Prometheus endpoint.
	if cfg.MetricsAddr != "" {
		metricsCtx, cancel := context.WithCancel(context.Background())
		app.metricsCancel = cancel
		observability.StartMetricsServer(
			metricsCtx, cfg.MetricsAddr, metrics,
			logger.With(slog.String("component", "metrics")),
		)
	}

	// Repo stack: sqlite ← instrumented decorator.
	baseRepo := sqlite.NewBookRepo(database)
	app.Repo = observability.NewInstrumentedBookRepo(
		baseRepo, metrics, logger.With(slog.String("component", "repo")),
	)
	app.Service = service.NewBookService(
		app.Repo,
		logger.With(slog.String("component", "service")),
		cfg.DefaultTimeout,
	)
	return app, nil
}
