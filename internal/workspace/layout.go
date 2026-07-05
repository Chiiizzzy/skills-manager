package workspace

import "path/filepath"

type Layout struct {
	Root string
}

func New(root string) Layout {
	return Layout{Root: root}
}

func (l Layout) ConfigPath() string {
	return filepath.Join(l.Root, "skillctl.yaml")
}

func (l Layout) LockPath() string {
	return filepath.Join(l.Root, "skillctl.lock")
}

func (l Layout) SourceDir(source string) string {
	return filepath.Join(l.Root, "sources", source)
}

func (l Layout) PatchFile(skill string) string {
	return filepath.Join(l.Root, "patches", skill, "local.patch")
}

func (l Layout) DistSkillDir(skill string) string {
	return filepath.Join(l.Root, "dist", skill)
}

func (l Layout) BackupDir(profile string) string {
	return filepath.Join(l.Root, "backups", profile)
}
