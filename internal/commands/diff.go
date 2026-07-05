package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/gitx"
)

func newDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff [skill]",
		Short: "Show managed skill diffs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			layout, cfg, err := loadConfig(rootPath)
			if err != nil {
				return err
			}
			skillName := args[0]
			if _, ok := collectSkills(cfg)[skillName]; !ok {
				return fmt.Errorf("unknown skill %q", skillName)
			}
			git := gitx.Runner{}
			repoDir, cleanup, err := prepareDiffRepo(cmd.Context(), layout, cfg, git, skillName)
			if err != nil {
				return err
			}
			defer cleanup()
			out, err := git.Run(cmd.Context(), repoDir, "diff", "--binary")
			if err != nil {
				return err
			}
			if out == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes")
				return nil
			}
			fmt.Fprint(cmd.OutOrStdout(), out)
			return nil
		},
	}
}
