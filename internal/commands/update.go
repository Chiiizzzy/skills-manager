package commands

import "github.com/spf13/cobra"

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update [skill]",
		Short: "Update managed skills from upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
