package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/syncer"
)

func newRollbackCommand() *cobra.Command {
	var profile string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback synced skills for a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			profileName, selectedProfile, err := selectProfile(cfg, profile)
			if err != nil {
				return err
			}
			backupDir := layout.BackupDir(profileName)
			entries, err := os.ReadDir(backupDir)
			if err != nil {
				return fmt.Errorf("read backups for profile %s: %w", profileName, err)
			}
			var backups []string
			for _, entry := range entries {
				if entry.IsDir() {
					backups = append(backups, entry.Name())
				}
			}
			if len(backups) == 0 {
				return fmt.Errorf("no backups found for profile %s", profileName)
			}
			sort.Strings(backups)
			latest := backups[len(backups)-1]
			latestDir := filepath.Join(backupDir, latest)
			skillEntries, err := os.ReadDir(latestDir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, entry := range skillEntries {
				if !entry.IsDir() {
					continue
				}
				skillName := entry.Name()
				targetDir := filepath.Join(selectedProfile.Target, skillName)
				if err := os.RemoveAll(targetDir); err != nil {
					return err
				}
				if err := syncer.CopyDir(filepath.Join(latestDir, skillName), targetDir); err != nil {
					return fmt.Errorf("rollback %s: %w", skillName, err)
				}
				fmt.Fprintf(out, "rolled back %s from %s\n", skillName, latest)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "Profile to rollback")

	return cmd
}
