package commands

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skillctl",
		Short: "Manage upstream skills, local patches, and agent skill installs",
	}

	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDiffCommand())
	cmd.AddCommand(newPatchCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newRollbackCommand())

	return cmd
}
