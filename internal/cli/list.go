package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

func newListCmd(appp **App) *cobra.Command {
	var (
		statusStr, category, author, search string
		limit                               int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List books with optional filters",
		Example: `  reading-tracker list --status reading
  reading-tracker list --category Programming --limit 10
  reading-tracker list --search "concurrency"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := *appp
			in := service.ListInput{Limit: limit}
			if statusStr != "" {
				s := domain.Status(statusStr)
				if !s.Valid() {
					return fmt.Errorf("invalid status %q (want: want_to_read|reading|completed|abandoned)", statusStr)
				}
				in.Status = &s
			}
			if category != "" {
				in.Category = &category
			}
			if author != "" {
				in.Author = &author
			}
			if search != "" {
				in.Search = &search
			}

			books, err := app.Service.List(cmd.Context(), in)
			if err != nil {
				return err
			}
			renderBooksTable(os.Stdout, books)
			return nil
		},
	}

	cmd.Flags().StringVar(&statusStr, "status", "", "filter by status")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&author, "author", "", "filter by author")
	cmd.Flags().StringVar(&search, "search", "", "substring search across title/author/description/category")
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of results (0 = unlimited)")
	return cmd
}
