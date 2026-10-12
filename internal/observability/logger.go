// Package observability centralizes logging configuration and Prometheus
// metrics. It exposes decorators that wrap repositories with instrumentation
// without those repositories knowing metrics exist.
package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// LoggerConfig controls how the root logger behaves.
// Populated from env in LoggerConfigFromEnv, or set directly in tests.
type LoggerConfig struct {
	Level  string    // "debug", "info", "warn", "error"
	Format string    // "text" or "json"
	Source bool      // include file:line in every record (dev only — it's expensive)
	Writer io.Writer // nil → os.Stderr
}

// NewLogger builds a *slog.Logger from cfg. This is the only place in the
// codebase that knows whether output is text or JSON.
func NewLogger(cfg LoggerConfig) *slog.Logger {
	if cfg.Writer == nil {
		cfg.Writer = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.Level),
		AddSource: cfg.Source,
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "json":
		handler = slog.NewJSONHandler(cfg.Writer, opts)
	default:
		handler = slog.NewTextHandler(cfg.Writer, opts)
	}
	return slog.New(handler)
}

// LoggerConfigFromEnv reads RT_LOG_* env vars with sane defaults.
//
//	RT_LOG_LEVEL=debug|info|warn|error   (default: info)
//	RT_LOG_FORMAT=text|json              (default: text)
//	RT_LOG_SOURCE=1|true|yes             (default: false)
func LoggerConfigFromEnv() LoggerConfig {
	return LoggerConfig{
		Level:  envDefault("RT_LOG_LEVEL", "info"),
		Format: envDefault("RT_LOG_FORMAT", "text"),
		Source: isTruthy(os.Getenv("RT_LOG_SOURCE")),
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func isTruthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
