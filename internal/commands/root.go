package commands

import "github.com/spf13/cobra"

var rootPath string

func NewRootCommand() *cobra.Command {
	rootPath = "."
	cmd := &cobra.Command{
		Use:          "skillctl",
		Short:        "Manage upstream skills, local patches, and agent skill installs",
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVar(&rootPath, "root", ".", "skills-manager repository root")

	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDiffCommand())
	cmd.AddCommand(newPatchCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newRollbackCommand())

	return cmd
}
