package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/your-org/skills-manager/internal/config"
	"github.com/your-org/skills-manager/internal/gitx"
	patchsvc "github.com/your-org/skills-manager/internal/patch"
	"github.com/your-org/skills-manager/internal/syncer"
	"github.com/your-org/skills-manager/internal/workspace"
)

type managedSkill struct {
	Name       string
	SourceName string
	Source     config.Source
	Skill      config.Skill
}

func loadConfig(root string) (workspace.Layout, *config.Config, error) {
	layout := workspace.New(root)
	cfg, err := config.LoadConfig(layout.ConfigPath())
	if err != nil {
		return layout, nil, fmt.Errorf("load config %s: %w", layout.ConfigPath(), err)
	}
	return layout, cfg, nil
}

func collectSkills(cfg *config.Config) map[string]managedSkill {
	skills := make(map[string]managedSkill)
	for sourceName, source := range cfg.Sources {
		for skillName, skill := range source.Skills {
			skills[skillName] = managedSkill{
				Name:       skillName,
				SourceName: sourceName,
				Source:     source,
				Skill:      skill,
			}
		}
	}
	return skills
}

func sortedSkillNames(cfg *config.Config) []string {
	skills := collectSkills(cfg)
	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedSourceNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Sources))
	for name := range cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func selectUpdateSkills(cfg *config.Config, skillName string, all bool) ([]managedSkill, error) {
	if all && skillName != "" {
		return nil, fmt.Errorf("use either --all or a skill name, not both")
	}
	skills := collectSkills(cfg)
	if skillName != "" {
		skill, ok := skills[skillName]
		if !ok {
			return nil, fmt.Errorf("unknown skill %q", skillName)
		}
		return []managedSkill{skill}, nil
	}
	if !all {
		return nil, fmt.Errorf("specify a skill or --all")
	}
	names := sortedSkillNames(cfg)
	selected := make([]managedSkill, 0, len(names))
	for _, name := range names {
		selected = append(selected, skills[name])
	}
	return selected, nil
}

func selectProfile(cfg *config.Config, name string) (string, config.Profile, error) {
	if name != "" {
		profile, ok := cfg.Profiles[name]
		if !ok {
			return "", config.Profile{}, fmt.Errorf("unknown profile %q", name)
		}
		return name, profile, nil
	}
	if len(cfg.Profiles) != 1 {
		return "", config.Profile{}, fmt.Errorf("profile is required")
	}
	for profileName, profile := range cfg.Profiles {
		return profileName, profile, nil
	}
	return "", config.Profile{}, fmt.Errorf("profile is required")
}

func readLock(path string) (config.Lock, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return config.Lock{Skills: map[string]config.LockedSkill{}}, nil
	}
	if err != nil {
		return config.Lock{}, err
	}
	var lock config.Lock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return config.Lock{}, err
	}
	if lock.Skills == nil {
		lock.Skills = map[string]config.LockedSkill{}
	}
	return lock, nil
}

func writeLock(path string, lock config.Lock) error {
	if lock.Skills == nil {
		lock.Skills = map[string]config.LockedSkill{}
	}
	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func updateSkill(ctx context.Context, layout workspace.Layout, git gitx.Runner, skill managedSkill, previous *config.LockedSkill) (string, error) {
	sourceDir := layout.SourceDir(skill.SourceName)
	if err := git.CloneOrFetch(ctx, skill.Source.Repo, skill.Source.Ref, sourceDir); err != nil {
		return "", err
	}
	commit, err := git.Resolve(ctx, sourceDir, skill.Source.Ref)
	if err != nil {
		return "", err
	}

	tmp, err := os.MkdirTemp("", "skillctl-update-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	patchFile := layout.PatchFile(skill.Name)
	_, statErr := os.Stat(patchFile)
	if statErr == nil {
		if previous != nil && previous.Source == skill.SourceName {
			if err := checkoutSourcePath(ctx, git, sourceDir, previous.UpstreamCommit, previous.UpstreamPath, tmp); err != nil {
				return "", err
			}
			if err := initBaselineRepo(ctx, git, tmp); err != nil {
				return "", err
			}
			if err := replaceWorktreeWithSource(ctx, git, sourceDir, commit, skill.Skill.Path, tmp); err != nil {
				return "", err
			}
			if err := commitAll(ctx, git, tmp, "new baseline"); err != nil {
				return "", err
			}
		} else {
			if err := checkoutSourcePath(ctx, git, sourceDir, commit, skill.Skill.Path, tmp); err != nil {
				return "", err
			}
			if err := initBaselineRepo(ctx, git, tmp); err != nil {
				return "", err
			}
		}
		if err := (patchsvc.Service{Git: git}).Apply(ctx, tmp, patchFile); err != nil {
			if copyErr := copyWorktreeToDist(tmp, layout.DistSkillDir(skill.Name)); copyErr != nil {
				return "", fmt.Errorf("%w; additionally failed to save conflicted dist: %v", err, copyErr)
			}
			return "", err
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	} else if err := checkoutSourcePath(ctx, git, sourceDir, commit, skill.Skill.Path, tmp); err != nil {
		return "", err
	}

	if err := copyWorktreeToDist(tmp, layout.DistSkillDir(skill.Name)); err != nil {
		return "", err
	}
	return commit, nil
}

func copyWorktreeToDist(worktreeDir, distDir string) error {
	if err := os.RemoveAll(distDir); err != nil {
		return err
	}
	return filepath.WalkDir(worktreeDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(worktreeDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(distDir, 0o755)
		}
		if isGitMetadataPath(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(distDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFilePreserveMode(path, target)
	})
}

func isGitMetadataPath(rel string) bool {
	return rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator))
}

func copyFilePreserveMode(src, dst string) error {
	info, statErr := os.Stat(src)
	if statErr != nil {
		return statErr
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkdirErr != nil {
		return mkdirErr
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode().Perm())
}

func checkoutSourcePath(ctx context.Context, git gitx.Runner, sourceDir, rev, upstreamPath, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return replaceWorktreeWithSource(ctx, git, sourceDir, rev, upstreamPath, dst)
}

func replaceWorktreeWithSource(ctx context.Context, git gitx.Runner, sourceDir, rev, upstreamPath, dst string) error {
	sourcePath, err := safeJoin(sourceDir, upstreamPath)
	if err != nil {
		return err
	}
	if err := removeWorktreeFiles(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.RemoveAll(sourcePath); err != nil {
		return err
	}
	if _, err := git.Run(ctx, sourceDir, "checkout", "-f", rev, "--", upstreamPath); err != nil {
		return err
	}
	return syncer.CopyDir(sourcePath, dst)
}

func safeJoin(root, rel string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	rootAbs = filepath.Clean(rootAbs)
	pathAbs = filepath.Clean(pathAbs)
	if pathAbs == rootAbs || !strings.HasPrefix(pathAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q must be inside %q", rel, root)
	}
	return pathAbs, nil
}

func initBaselineRepo(ctx context.Context, git gitx.Runner, dir string) error {
	steps := [][]string{
		{"init"},
		{"config", "user.name", "skillctl"},
		{"config", "user.email", "skillctl@example.invalid"},
	}
	for _, args := range steps {
		if _, err := git.Run(ctx, dir, args...); err != nil {
			return err
		}
	}
	return commitAll(ctx, git, dir, "skill baseline")
}

func commitAll(ctx context.Context, git gitx.Runner, dir, message string) error {
	if _, err := git.Run(ctx, dir, "add", "-A"); err != nil {
		return err
	}
	if _, err := git.Run(ctx, dir, "commit", "-m", message); err != nil {
		status, statusErr := git.Run(ctx, dir, "status", "--porcelain")
		if statusErr == nil && strings.TrimSpace(status) == "" {
			return nil
		}
		return err
	}
	return nil
}

func prepareDiffRepo(ctx context.Context, layout workspace.Layout, cfg *config.Config, git gitx.Runner, skillName string) (string, func(), error) {
	lock, err := readLock(layout.LockPath())
	if err != nil {
		return "", nil, fmt.Errorf("load lock %s: %w", layout.LockPath(), err)
	}
	locked, ok := lock.Skills[skillName]
	if !ok {
		return "", nil, fmt.Errorf("skill %q is not in lock file", skillName)
	}
	source, ok := cfg.Sources[locked.Source]
	if !ok {
		return "", nil, fmt.Errorf("lock for skill %q references unknown source %q", skillName, locked.Source)
	}
	sourceDir := layout.SourceDir(locked.Source)
	if cloneErr := git.CloneOrFetch(ctx, source.Repo, source.Ref, sourceDir); cloneErr != nil {
		return "", nil, cloneErr
	}

	tmp, err := os.MkdirTemp("", "skillctl-diff-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	if err := checkoutSourcePath(ctx, git, sourceDir, locked.UpstreamCommit, locked.UpstreamPath, tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := initBaselineRepo(ctx, git, tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := removeWorktreeFiles(tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := syncer.CopyDir(layout.DistSkillDir(skillName), tmp); err != nil {
		cleanup()
		return "", nil, err
	}
	if _, err := git.Run(ctx, tmp, "add", "-N", "."); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmp, cleanup, nil
}

func removeWorktreeFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}
