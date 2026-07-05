package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/doctor"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check skill manager configuration and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			var issues []doctor.Issue
			for _, skill := range sortedSkillNames(cfg) {
				issues = append(issues, doctor.CheckSkillDir(layout.DistSkillDir(skill))...)
			}
			out := cmd.OutOrStdout()
			if len(issues) == 0 {
				fmt.Fprintln(out, doctor.FormatIssues(nil))
				return nil
			}
			fmt.Fprint(out, doctor.FormatIssues(issues))
			return fmt.Errorf("doctor found %d issue(s)", len(issues))
		},
	}
}
