package commands

import "github.com/spf13/cobra"

func newPatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage local skill patches",
	}

	cmd.AddCommand(newPatchRefreshCommand())

	return cmd
}

func newPatchRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh local skill patches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
