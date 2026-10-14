package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/service"
)

func newSeedCmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Insert demo books (only if the library is empty)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app := *appp
			ctx := cmd.Context()

			existing, err := app.Service.List(ctx, service.ListInput{Limit: 1})
			if err != nil {
				return err
			}
			if len(existing) > 0 {
				fmt.Println("library not empty — skipping seed")
				return nil
			}

			seed := []service.AddBookInput{
				{
					Title:       "Mastering Go",
					Author:      "Mihalis Tsoukalos",
					Category:    "Programming",
					Description: "Leverage Go's expertise for advanced utilities",
					TotalPages:  711,
				},
				{
					Title:       "Learning Go",
					Author:      "Jon Bodner",
					Category:    "Programming",
					Description: "An idiomatic approach to real world Go programming",
					TotalPages:  353,
				},
				{
					Title:       "Concurrency in GO",
					Author:      "Katherine Cox-Buday",
					Category:    "Programming",
					Description: "Tools & Techniques for developers",
					TotalPages:  224,
				},
			}
			var firstID int64
			for i, in := range seed {
				b, err := app.Service.Add(ctx, in)
				if err != nil {
					return fmt.Errorf("seed %q: %w", in.Title, err)
				}
				if i == 0 {
					firstID = b.ID
				}
			}
			if firstID > 0 {
				if _, err := app.Service.UpdateProgress(ctx, firstID, 526); err != nil {
					return err
				}
			}
			fmt.Printf("seeded %d books\n", len(seed))
			return nil
		},
	}
}
