package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/config"
	"github.com/your-org/skills-manager/internal/gitx"
)

func newUpdateCommand() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "update [skill]",
		Short: "Update managed skills from upstream",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			skillName := ""
			if len(args) == 1 {
				skillName = args[0]
			}
			selected, err := selectUpdateSkills(cfg, skillName, all)
			if err != nil {
				return err
			}
			lock, err := readLock(layout.LockPath())
			if err != nil {
				return err
			}
			git := gitx.Runner{}
			out := cmd.OutOrStdout()
			for _, skill := range selected {
				var previous *config.LockedSkill
				if locked, ok := lock.Skills[skill.Name]; ok {
					previous = &locked
				}
				commit, err := updateSkill(cmd.Context(), layout, git, skill, previous)
				if err != nil {
					return fmt.Errorf("update %s: %w", skill.Name, err)
				}
				lock.Skills[skill.Name] = config.LockedSkill{
					Source:         skill.SourceName,
					UpstreamCommit: commit,
					UpstreamPath:   skill.Skill.Path,
				}
				if err := writeLock(layout.LockPath(), lock); err != nil {
					return err
				}
				fmt.Fprintf(out, "updated %s at %s\n", skill.Name, commit)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Update all managed skills")
	return cmd
}
