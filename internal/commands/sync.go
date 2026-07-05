package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/doctor"
	"github.com/your-org/skills-manager/internal/syncer"
)

func newSyncCommand() *cobra.Command {
	var profile string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync managed skills to an agent skills directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			profileName, selectedProfile, err := selectProfile(cfg, profile)
			if err != nil {
				return err
			}
			var issues []doctor.Issue
			for _, skillName := range selectedProfile.Skills {
				issues = append(issues, doctor.CheckSkillDir(layout.DistSkillDir(skillName))...)
			}
			if len(issues) > 0 {
				fmt.Fprint(cmd.OutOrStdout(), doctor.FormatIssues(issues))
				return fmt.Errorf("doctor found %d issue(s)", len(issues))
			}

			out := cmd.OutOrStdout()
			timestamp := syncer.Timestamp()
			backupRoot := filepath.Join(layout.BackupDir(profileName), timestamp)
			for _, skillName := range selectedProfile.Skills {
				distDir := layout.DistSkillDir(skillName)
				targetDir := filepath.Join(selectedProfile.Target, skillName)
				if dryRun {
					fmt.Fprintf(out, "would sync %s -> %s\n", distDir, targetDir)
					continue
				}
				if _, err := os.Stat(targetDir); err == nil {
					if err := syncer.CopyDir(targetDir, filepath.Join(backupRoot, skillName)); err != nil {
						return fmt.Errorf("backup %s: %w", skillName, err)
					}
				} else if !os.IsNotExist(err) {
					return err
				}
				if err := os.RemoveAll(targetDir); err != nil {
					return err
				}
				if err := syncer.CopyDir(distDir, targetDir); err != nil {
					return fmt.Errorf("sync %s: %w", skillName, err)
				}
				fmt.Fprintf(out, "synced %s -> %s\n", skillName, targetDir)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile to sync")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync without writing files")

	return cmd
}
