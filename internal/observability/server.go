package observability

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// StartMetricsServer launches an HTTP server exposing:
//
//	GET /metrics   — Prometheus exposition
//	GET /healthz   — 200 OK
//
// The server shuts down when ctx is cancelled. No cleanup return value needed:
// the caller's existing signal.NotifyContext is the shutdown signal.
func StartMetricsServer(ctx context.Context, addr string, m *Metrics, logger *slog.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.InfoContext(ctx, "metrics server listening",
			slog.String("addr", addr),
			slog.String("metrics_path", "/metrics"),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Bind failure (port already in use) is worth knowing about.
			logger.ErrorContext(ctx, "metrics server error", slog.Any("err", err))
		}
	}()

	go func() {
		<-ctx.Done()
		// Give in-flight scrapes a moment to finish before force-closing.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.WarnContext(ctx, "metrics server shutdown", slog.Any("err", err))
		}
	}()
}
