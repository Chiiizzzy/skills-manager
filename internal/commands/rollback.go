package commands

import "github.com/spf13/cobra"

func newRollbackCommand() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback synced skills for a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile to rollback")

	return cmd
}
