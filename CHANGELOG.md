# Changelog

All notable changes to this project are documented here.

## [Unreleased]

## [0.1.0] - 2026-10-14

### Added
- CLI commands: `add`, `list`, `get`, `progress`, `status`, `delete`, `stats`, `seed`, `tui`, `version`
- Interactive TUI with list, detail, add, update, delete, and stats views
- SQLite storage with embedded migrations
- Structured logging (`slog`) with text and JSON formats
- Prometheus metrics via repository decorator
- Install script with SHA256 verification
- Cross-compiled release binaries for darwin/linux × arm64/amd64
