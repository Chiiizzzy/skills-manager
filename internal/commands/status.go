package commands

import "github.com/spf13/cobra"

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show managed skills status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
