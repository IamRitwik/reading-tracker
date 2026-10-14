package cli

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/tui"
)

func newTUICmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:     "tui",
		Aliases: []string{"ui"},
		Short:   "Launch the interactive interface",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := *appp
			return tui.Run(
				cmd.Context(),
				app.Service,
				app.Logger.With(slog.String("component", "tui")),
			)
		},
	}
}
