package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newGetCmd(appp **App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show details for one book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q: %w", args[0], err)
			}
			b, err := (*appp).Service.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			renderBook(os.Stdout, b)
			return nil
		},
	}
}
