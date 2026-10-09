package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const pragmaQuery = "_pragma=journal_mode(WAL)" +
	"&_pragma=busy_timeout(5000)" +
	"&_pragma=foreign_keys(1)" +
	"&_pragma=synchronous(NORMAL)"

// Open opens the SQLite database, applies pragmas, and runs migrations.
// It honors ctx for the initial connection and migration phase.
func Open(ctx context.Context, path string) (*sql.DB, error) {

	dsn := buildDSN(path)

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// CRITICAL for SQLite with Go's database/sql.
	// database/sql is designed for connection pools (Postgres, MySQL).
	// SQLite is a single-writer embedded DB. Multiple concurrent writers
	// produce "database is locked" (SQLITE_BUSY) errors. Setting MaxOpenConns(1)
	// serializes ALL access through one connection, eliminating that entire
	// class of bug. For a personal app, throughput is a non-issue.

	// Caveat: this also means a slow query blocks everything. For a reading
	// tracker, our queries are sub-millisecond. If you later add heavy
	// analytics, use a separate read-only pool or an in-memory cache.
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxLifetime(0) //SQLite connections live for process lifetime

	err = database.PingContext(ctx)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	err = migrate(ctx, database)
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return database, nil
}

// buildDSN adds our pragmas to whatever path we're given.
func buildDSN(path string) string {
	// For in-memory DSNs, modernc wants file::memory: form when pragmas are attached.
	if path == ":memory:" {
		return "file::memory:?cache=shared&" + pragmaQuery
	}
	if strings.Contains(path, "?") {
		return path + "&" + pragmaQuery
	}
	return path + "?" + pragmaQuery
}

// migrate applies any migrations not yet recorded in schema_migrations.
// Each migration runs inside its own transaction: either it fully applies
// or it fully rolls back, and the version record is committed atomically
// with the schema change.
func migrate(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := migrationApplied(ctx, database, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		err = applyMigration(ctx, database, name, string(content))
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "migration applied", slog.String("version", name))
	}
	return nil

}

func migrationApplied(ctx context.Context, database *sql.DB, version string) (bool, error) {
	var count int
	err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check %s: %w", version, err)
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, database *sql.DB, version, sql string) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for %s: %w", version, err)
	}
	// Deferred rollback is a safety net — if Commit succeeds, Rollback is a no-op.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, sql); err != nil {
		return fmt.Errorf("exec %s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		version, nowRFC3339(),
	); err != nil {
		return fmt.Errorf("record %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", version, err)
	}
	return nil
}
