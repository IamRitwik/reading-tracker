# reading-tracker

A personal reading tracker with a fast CLI and an interactive terminal UI.

Track books from *want to read* through *reading* to *completed*. Log progress by
page, see per-category stats, and search your library — all from the terminal.
Data lives in a local SQLite file; no accounts, no cloud, no tracking.

```
$ reading-tracker list
ID   TITLE                          AUTHOR                 CATEGORY      PROGRESS         STATUS
1    Mastering Go                   Mihalis Tsoukalos      Programming   526/711 (74%)    reading
2    Concurrency in Go              Katherine Cox-Buday    Programming   0/224 (0%)       want_to_read
3    Sapiens                        Yuval Noah Harari      History       443/443 (100%)   completed
```

```
$ reading-tracker tui
╭──────────────────────────────────────────────╮
│  📚 Reading Tracker                          │
│                                              │
│  ▸ Mastering Go                              │
│      Mihalis Tsoukalos  ███████████░░░░  74%  READING
│    Concurrency in Go                         │
│      Katherine Cox-Buday  ░░░░░░░░░░░░░░░   0%  WANT
│    Sapiens                                   │
│      Yuval Noah Harari  ███████████████ 100%  DONE
│                                              │
│  ↑/k ↓/j move · enter detail · a add · u update · d delete · s stats · q quit
╰──────────────────────────────────────────────╯
```

## Features

- **Fast CLI** — add, list, search, update progress, view stats, all under a second
- **Interactive TUI** — browse, add, update, delete, and see stats with keyboard shortcuts
- **Progress tracking** — pages read, automatic status transitions (want → reading → done)
- **Search** — substring search across title, author, description, and category
- **Statistics** — books by status, pages read, per-category breakdown
- **Local-first** — SQLite database at `~/Library/Application Support/reading-tracker/reading.db` (macOS) or `~/.config/reading-tracker/reading.db` (Linux). No network, no accounts.
- **Structured logging** — text or JSON via `RT_LOG_FORMAT`
- **Prometheus metrics** — optional `/metrics` endpoint via `--metrics-addr`

## Install

### One-liner (macOS & Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/IamRitwik/reading-tracker/main/scripts/install.sh | sh
```

Installs the latest release to `~/.local/bin/reading-tracker` and verifies the
download against `SHA256SUMS`.

If `reading-tracker: command not found` after install, add `~/.local/bin` to your PATH:

```sh
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.zshrc
source ~/.zshrc
```

Install a specific version or a custom directory:

```sh
curl -fsSL ... | sh -s -- --version v0.1.0
curl -fsSL ... | sh -s -- --dir /usr/local/bin
```

### Manual download

Grab the archive for your platform from the
[releases page](https://github.com/IamRitwik/reading-tracker/releases/latest),
verify it against `SHA256SUMS`, extract, and run.

| Platform | Archive |
|---|---|
| macOS (Apple Silicon — M1/M2/M3/M4) | `reading-tracker_<ver>_darwin_arm64.tar.gz` |
| macOS (Intel) | `reading-tracker_<ver>_darwin_amd64.tar.gz` |
| Linux (x86_64) | `reading-tracker_<ver>_linux_amd64.tar.gz` |
| Linux (ARM64) | `reading-tracker_<ver>_linux_arm64.tar.gz` |

Example (macOS Apple Silicon):

```sh
VERSION=v0.1.0
BASE="https://github.com/IamRitwik/reading-tracker/releases/download/$VERSION"

curl -fsSLO "$BASE/reading-tracker_0.1.0_darwin_arm64.tar.gz"
curl -fsSLO "$BASE/reading-tracker_0.1.0_SHA256SUMS"
shasum -a 256 -c reading-tracker_0.1.0_SHA256SUMS --ignore-missing

tar -xzf reading-tracker_0.1.0_darwin_arm64.tar.gz
./reading-tracker version
./reading-tracker tui
```

> **macOS Gatekeeper note:** if macOS refuses to run the binary, clear the quarantine flag once:
> `xattr -d com.apple.quarantine ./reading-tracker`

### From source (Go 1.23+)

```sh
git clone https://github.com/IamRitwik/reading-tracker.git
cd reading-tracker
make build           # produces ./reading-tracker
make install         # installs to $GOPATH/bin
```

## Quick start

```sh
reading-tracker seed                                     # insert demo books (once)
reading-tracker add --title "Sapiens" --author "Yuval Noah Harari" \
                    --pages 443 --category History
reading-tracker list
reading-tracker progress 1 120                           # set pages read
reading-tracker stats
reading-tracker tui                                      # interactive mode
```

## CLI reference

### `add`

Add a new book.

```sh
reading-tracker add --title "The Go Programming Language" \
                    --author "Donovan & Kernighan" \
                    --pages 380 \
                    --category Programming \
                    --description "The definitive Go reference"
```

Required: `--title`, `--author`, `--pages`.

### `list`

List books with optional filters.

```sh
reading-tracker list                              # all books
reading-tracker list --status reading             # filter by status
reading-tracker list --category Programming       # filter by category
reading-tracker list --author "Harari"            # filter by author
reading-tracker list --search concurrency         # substring search
reading-tracker list --limit 10                   # max results
```

### `get`

Show details for one book.

```sh
reading-tracker get 1
```

### `progress`

Update pages read. Status transitions automatically:

- `0 / N` pages → `want_to_read`
- `1..N-1` pages → `reading`
- `N / N` pages → `completed`

```sh
reading-tracker progress 1 120
reading-tracker progress 1 380        # marks completed if total is 380
```

### `status`

Explicitly set status. Enforces transition rules (e.g. `want_to_read` → `completed`
is rejected; use `progress` to finish a book).

```sh
reading-tracker status 1 reading
reading-tracker status 1 abandoned
reading-tracker status 1 completed    # sets pages_read = total_pages
```

### `delete`

Remove a book.

```sh
reading-tracker delete 3              # interactive confirmation
reading-tracker delete 3 --yes        # skip confirmation
```

### `stats`

Reading statistics: counts by status, total pages, per-category breakdown.

```sh
reading-tracker stats
```

### `seed`

Insert demo books. Only runs if the library is empty.

```sh
reading-tracker seed
```

### `tui`

Launch the interactive interface.

```sh
reading-tracker tui
```

Keyboard shortcuts:

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `g` / `G` | Jump to first / last |
| `Enter` | Open book detail |
| `Esc` | Back |
| `a` | Add a book |
| `u` | Update progress |
| `d` | Delete (with confirmation) |
| `s` | Show stats |
| `r` | Refresh from database |
| `q` / `Ctrl+C` | Quit |

### `version`

Print version, commit, build date, and runtime info.

```sh
reading-tracker version
```

## Configuration

Configuration is layered. Highest priority wins:

```
flags  >  env vars  >  config file  >  defaults
```

### Flags (global)

```
--config <path>           path to config file
--db-path <path>          path to SQLite database
--log-level <level>       debug | info | warn | error   (default: info)
--log-format <format>     text | json                   (default: text)
--log-source              include file:line in logs
--metrics-addr <addr>     address for /metrics, e.g. :2112
--default-timeout <dur>   per-operation timeout, e.g. 2s
```

### Environment variables

| Variable | Equivalent flag |
|---|---|
| `RT_DB_PATH` | `--db-path` |
| `RT_LOG_LEVEL` | `--log-level` |
| `RT_LOG_FORMAT` | `--log-format` |
| `RT_LOG_SOURCE` | `--log-source` |
| `RT_METRICS_ADDR` | `--metrics-addr` |
| `RT_DEFAULT_TIMEOUT` | `--default-timeout` |

### Config file

Location (default): `~/Library/Application Support/reading-tracker/config.yaml` on macOS,
`~/.config/reading-tracker/config.yaml` on Linux.

```yaml
db-path: /Users/you/Dropbox/reading.db
log-level: info
log-format: text
default-timeout: 2s
```

Use `--config /path/to/config.yaml` to point elsewhere.

## Data

Your library lives in SQLite:

| OS | Path |
|---|---|
| macOS | `~/Library/Application Support/reading-tracker/reading.db` |
| Linux | `~/.config/reading-tracker/reading.db` |

**Backup** — copy the `.db` file while the app is not running:

```sh
cp ~/Library/Application\ Support/reading-tracker/reading.db ~/Desktop/reading-backup.db
```

**Move to another machine** — copy the `.db` file to the same path on the other machine.

**Sync across machines** — point `db-path` at a Dropbox/iCloud-synced folder. Don't
run the app on two machines at once against the same file; SQLite is not designed
for that.

**Inspect** — with `sqlite3`:

```sh
sqlite3 ~/Library/Application\ Support/reading-tracker/reading.db
sqlite> .mode column
sqlite> .headers on
sqlite> SELECT id, title, status, pages_read, total_pages FROM books;
```

## Observability

### Logging

```sh
RT_LOG_FORMAT=json reading-tracker list             # JSON logs
RT_LOG_LEVEL=debug reading-tracker list             # verbose
RT_LOG_SOURCE=1 reading-tracker list                # include file:line
```

Logs go to stderr. Command output goes to stdout. Pipe-safe.

### Metrics

Start the metrics server on any command:

```sh
reading-tracker tui --metrics-addr=:2112
# or
RT_METRICS_ADDR=:2112 reading-tracker list
```

Then scrape:

```sh
curl -s localhost:2112/metrics | grep reading_tracker
```

Available metrics:

| Metric | Type | Description |
|---|---|---|
| `reading_tracker_books_added_total` | counter | Books added |
| `reading_tracker_books_updated_total` | counter | Successful updates |
| `reading_tracker_books_deleted_total` | counter | Books deleted |
| `reading_tracker_db_query_duration_seconds` | histogram | Repo op duration (labels: `operation`, `result`) |
| `reading_tracker_db_query_errors_total` | counter | Repo errors (label: `operation`) |

Standard Go runtime metrics (`go_goroutines`, `go_memstats_*`, `process_*`) are
included by default.

## Build from source

Requires Go 1.23+.

```sh
make build      # build ./reading-tracker with embedded version info
make test       # run tests with the race detector
make vet        # go vet
make fmt        # gofmt -w
make install    # build + install to $GOPATH/bin
make clean      # remove build artifacts and dist/
```

## Release (maintainers)

Prerequisites: `gh` CLI authenticated, clean working tree, and `vX.Y.Z` tag on `HEAD`.

```sh
git add .
git commit -m "feat: ..."
git push

git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

make release     # cross-compile, package tarballs, generate SHA256SUMS

gh release create v0.2.0 \
  --title "v0.2.0" \
  --generate-notes \
  dist/*.tar.gz dist/*_SHA256SUMS
```

Verify before publishing:

```sh
git describe --tags --always --dirty    # → v0.2.0
ls dist/                                # → 4 tarballs + SHA256SUMS, no -dirty
tar -tzf dist/reading-tracker_0.2.0_darwin_arm64.tar.gz   # → reading-tracker
```

## Architecture

```
cmd/reading-tracker/        entry point
internal/
  cli/                      cobra commands; wiring
  config/                   viper config loading (flags > env > file > defaults)
  db/                       sqlite opener, pragmas, embedded migrations
  domain/                   pure types (Book, Status, sentinel errors)
  observability/            slog config, prometheus metrics, repo decorator
  repository/
    sqlite/                 repository.BookRepository via database/sql
    memory/                 in-memory impl for service tests
  service/                  business logic (validation, state transitions, timeouts)
  tui/                      bubble tea model/view/update
```

Dependency direction (no cycles):

```
cmd → cli → service → repository → db
         → tui  ↗
         domain (imported everywhere)
```

Key patterns used:

- **Repository interface** — service depends on `repository.BookRepository`, not on `*sql.DB`
- **Decorator** — `InstrumentedBookRepo` wraps any repo to add Prometheus metrics without the inner type knowing
- **Context propagation** — `context.Context` flows from `signal.NotifyContext` in `main` through every layer down to SQLite
- **Elm architecture** — TUI is `Init` / `Update` / `View` with all async work in `tea.Cmd`
- **Embedded migrations** — `//go:embed migrations/*.sql`; schema ships inside the binary

## Uninstall

```sh
rm ~/.local/bin/reading-tracker
rm -rf ~/Library/Application\ Support/reading-tracker   # macOS
rm -rf ~/.config/reading-tracker                        # Linux
```

## License

MIT — see [LICENSE](LICENSE).