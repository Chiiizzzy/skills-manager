package commands

import "github.com/spf13/cobra"

func newDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [skill]",
		Short: "Show managed skill diffs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
