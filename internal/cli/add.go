package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/service"
)

func newAddCmd(appp **App) *cobra.Command {
	var in service.AddBookInput

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new book",
		Example: `  reading-tracker add --title "AI Engineering" --author "Chip Huyen" \
      --pages 443 --category AI/ML`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := *appp
			b, err := app.Service.Add(cmd.Context(), in)
			if err != nil {
				return err
			}
			fmt.Printf("added #%d: %s — %s (%d pages)\n",
				b.ID, b.Title, b.Author, b.TotalPages)
			return nil
		},
	}

	cmd.Flags().StringVar(&in.Title, "title", "", "book title (required)")
	cmd.Flags().StringVar(&in.Author, "author", "", "book author (required)")
	cmd.Flags().StringVar(&in.Category, "category", "", "category")
	cmd.Flags().StringVar(&in.Description, "description", "", "short description")
	cmd.Flags().IntVar(&in.TotalPages, "pages", 0, "total pages (required)")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("author")
	_ = cmd.MarkFlagRequired("pages")
	return cmd
}
