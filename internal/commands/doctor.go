package commands

import "github.com/spf13/cobra"

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check skill manager configuration and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
