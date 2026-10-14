// Package cli wires up the command-line interface with Cobra.
//
// The App struct (see app.go) is built once per invocation in the root
// command's PersistentPreRunE and shut down in Execute via a deferred Close.
// Subcommands receive a **App so they can access it after PersistentPreRunE
// has populated it — this avoids package-level global state.
package cli

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/config"
)

// Injected at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Execute is the entry point called from main. It guarantees App.Close is
// called exactly once, even if a subcommand returns an error.
func Execute(ctx context.Context) (err error) {
	var app *App
	defer func() {
		if cerr := app.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	root := newRootCmd(&app)
	return root.ExecuteContext(ctx)
}

func newRootCmd(appp **App) *cobra.Command {
	root := &cobra.Command{
		Use:   "reading-tracker",
		Short: "A personal reading tracker with CLI and TUI",
		Long: `reading-tracker manages your personal reading list.

Books have a title, author, category, description, and page count.
Progress flows from "want to read" through "reading" to "completed",
with support for abandoning and rereading.

Run 'reading-tracker tui' for the interactive interface.`,

		// Do not print usage on error, and do not print the error here —
		// main.go handles error output.
		SilenceUsage:  true,
		SilenceErrors: true,

		// Build the dependency graph once for every command that runs.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			app, err := buildApp(cmd)
			if err != nil {
				return err
			}
			*appp = app

			app.Logger.InfoContext(cmd.Context(), "reading-tracker starting",
				slog.String("version", Version),
				slog.String("commit", Commit),
				slog.String("built", BuildDate),
				slog.String("command", cmd.Name()),
			)
			return nil
		},

		// Bare `reading-tracker` prints help.
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Global flags available to every subcommand.
	pf := root.PersistentFlags()
	pf.String("config", "", "path to config file (default: "+config.ConfigFilePath()+")")
	pf.String("db-path", "", "path to SQLite database file")
	pf.String("log-level", "", "log level: debug|info|warn|error (default info)")
	pf.String("log-format", "", "log format: text|json (default text)")
	pf.Bool("log-source", false, "include source file:line in log records")
	pf.String("metrics-addr", "", "address for the Prometheus /metrics server, e.g. :2112")
	pf.Duration("default-timeout", 0, "default per-operation timeout, e.g. 2s")

	root.AddCommand(
		newAddCmd(appp),
		newListCmd(appp),
		newGetCmd(appp),
		newProgressCmd(appp),
		newStatusCmd(appp),
		newDeleteCmd(appp),
		newStatsCmd(appp),
		newSeedCmd(appp),
		newTUICmd(appp),
		newVersionCmd(),
	)
	return root
}
