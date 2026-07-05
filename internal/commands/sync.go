package commands

import "github.com/spf13/cobra"

func newSyncCommand() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync managed skills to an agent skills directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile to sync")

	return cmd
}
