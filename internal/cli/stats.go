package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/IamRitwik/reading-tracker/internal/domain"
	"github.com/IamRitwik/reading-tracker/internal/service"
)

func newStatsCmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show reading statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			books, err := (*appp).Service.List(cmd.Context(), service.ListInput{})
			if err != nil {
				return err
			}

			byStatus := map[domain.Status]int{}
			byCategory := map[string]int{}
			var pagesRead, pagesTotal int

			for _, b := range books {
				byStatus[b.Status]++
				if b.Category != "" {
					byCategory[b.Category]++
				}
				pagesRead += b.PagesRead
				pagesTotal += b.TotalPages
			}

			fmt.Printf("Total books:       %d\n", len(books))
			fmt.Printf("  Want to read:    %d\n", byStatus[domain.StatusWantToRead])
			fmt.Printf("  Reading:         %d\n", byStatus[domain.StatusReading])
			fmt.Printf("  Completed:       %d\n", byStatus[domain.StatusCompleted])
			fmt.Printf("  Abandoned:       %d\n", byStatus[domain.StatusAbandoned])
			fmt.Printf("Pages read:        %d\n", pagesRead)
			fmt.Printf("Pages total:       %d\n", pagesTotal)
			if pagesTotal > 0 {
				fmt.Printf("Overall progress:  %.1f%%\n",
					float64(pagesRead)/float64(pagesTotal)*100)
			}

			if len(byCategory) > 0 {
				fmt.Println("\nBy category:")
				type kv struct {
					k string
					v int
				}
				var sorted []kv
				for k, v := range byCategory {
					sorted = append(sorted, kv{k, v})
				}
				sort.Slice(sorted, func(i, j int) bool {
					if sorted[i].v != sorted[j].v {
						return sorted[i].v > sorted[j].v
					}
					return sorted[i].k < sorted[j].k
				})
				for _, p := range sorted {
					fmt.Printf("  %-20s %d\n", p.k, p.v)
				}
			}
			return nil
		},
	}
}
