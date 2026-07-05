package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/your-org/skills-manager/internal/gitx"
	patchsvc "github.com/your-org/skills-manager/internal/patch"
)

func newPatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage local skill patches",
	}

	cmd.AddCommand(newPatchRefreshCommand())

	return cmd
}

func newPatchRefreshCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh [skill]",
		Short: "Refresh local skill patches",
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
			patchFile := layout.PatchFile(skillName)
			diff, err := patchsvc.Service{Git: git}.Refresh(cmd.Context(), repoDir, patchFile)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(patchFile), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(patchFile, []byte(diff), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", patchFile)
			return nil
		},
	}
}
