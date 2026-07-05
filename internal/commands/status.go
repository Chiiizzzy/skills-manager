package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show managed skills status",
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Sources:")
			for _, name := range sortedSourceNames(cfg) {
				source := cfg.Sources[name]
				fmt.Fprintf(out, "- %s: %s @ %s\n", name, source.Repo, source.Ref)
			}

			skills := collectSkills(cfg)
			fmt.Fprintln(out, "Skills:")
			for _, name := range sortedSkillNames(cfg) {
				skill := skills[name]
				status := "missing"
				if _, err := os.Stat(filepath.Join(layout.DistSkillDir(name), "SKILL.md")); err == nil {
					status = "ready"
				}
				fmt.Fprintf(out, "- %s: source=%s path=%s dist=%s\n", name, skill.SourceName, skill.Skill.Path, status)
			}

			fmt.Fprintln(out, "Profiles:")
			for _, name := range sortedProfileNames(cfg) {
				profile := cfg.Profiles[name]
				fmt.Fprintf(out, "- %s: target=%s skills=%s\n", name, profile.Target, strings.Join(profile.Skills, ","))
			}
			return nil
		},
	}
}
