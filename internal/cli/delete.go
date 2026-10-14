package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newDeleteCmd(appp **App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a book",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", args[0], err)
			}
			app := *appp

			if !yes {
				b, err := app.Service.Get(cmd.Context(), id)
				if err != nil {
					return err
				}
				if !promptYes(fmt.Sprintf("Delete %q by %s?", b.Title, b.Author)) {
					fmt.Println("aborted")
					return nil
				}
			}

			if err := app.Service.Delete(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Printf("deleted #%d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	return cmd
}
