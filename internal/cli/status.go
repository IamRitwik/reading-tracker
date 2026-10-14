package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/domain"
)

func newStatusCmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:   "status <id> <status>",
		Short: "Set a book's status explicitly",
		Example: `  reading-tracker status 1 completed
  reading-tracker status 1 abandoned`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", args[0], err)
			}
			target := domain.Status(args[1])
			b, err := (*appp).Service.SetStatus(cmd.Context(), id, target)
			if err != nil {
				return err
			}
			fmt.Printf("#%d %q → status=%s\n", b.ID, b.Title, b.Status)
			return nil
		},
	}
}
