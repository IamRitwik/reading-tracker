package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newProgressCmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:   "progress <id> <pages-read>",
		Short: "Update a book's current page",
		Example: `  reading-tracker progress 1 120
  reading-tracker progress 1 443   # marks completed if total is 443`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", args[0], err)
			}
			pages, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid pages %q: %w", args[1], err)
			}
			b, err := (*appp).Service.UpdateProgress(cmd.Context(), id, pages)
			if err != nil {
				return err
			}
			fmt.Printf("#%d %q → %d/%d pages (%.1f%%) status=%s\n",
				b.ID, b.Title, b.PagesRead, b.TotalPages, b.ProgressPercent(), b.Status)
			return nil
		},
	}
}
