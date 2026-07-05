package doctor

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCheckSkillDirMissingSkillFile(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")

	got := CheckSkillDir(skillDir)
	want := []Issue{{
		Path:    skillFile,
		Message: "SKILL.md is missing",
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckSkillDir() = %#v, want %#v", got, want)
	}
}

func TestCheckSkillDirSkillFileIsDirectory(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.Mkdir(skillFile, 0o755); err != nil {
		t.Fatalf("mkdir SKILL.md: %v", err)
	}

	got := CheckSkillDir(skillDir)
	want := []Issue{{
		Path:    skillFile,
		Message: "SKILL.md is not a regular file",
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckSkillDir() = %#v, want %#v", got, want)
	}
}

func TestCheckSkillDirConflictMarkerInSkillFile(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("<<<<<<< HEAD\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	got := CheckSkillDir(skillDir)
	want := []Issue{{
		Path:    skillFile,
		Message: "git conflict marker found",
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckSkillDir() = %#v, want %#v", got, want)
	}
}

func TestCheckSkillDirSeparatorLineWithoutConflictBoundary(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("=======\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	if got := CheckSkillDir(skillDir); len(got) != 0 {
		t.Fatalf("CheckSkillDir() = %#v, want no issues", got)
	}
}

func TestCheckSkillDirConflictBlock(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")
	data := []byte("<<<<<<< HEAD\nlocal\n=======\nremote\n>>>>>>> branch\n")
	if err := os.WriteFile(skillFile, data, 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	got := CheckSkillDir(skillDir)
	want := []Issue{{
		Path:    skillFile,
		Message: "git conflict marker found",
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckSkillDir() = %#v, want %#v", got, want)
	}
}

func TestCheckSkillDirReadError(t *testing.T) {
	skillDir := t.TempDir()
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("# Test Skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	brokenFile := filepath.Join(skillDir, "broken.md")
	if err := os.Symlink(filepath.Join(skillDir, "missing.md"), brokenFile); err != nil {
		t.Skipf("create broken symlink: %v", err)
	}

	got := CheckSkillDir(skillDir)
	if len(got) != 1 {
		t.Fatalf("CheckSkillDir() = %#v, want one issue", got)
	}
	if got[0].Path != brokenFile {
		t.Fatalf("CheckSkillDir()[0].Path = %q, want %q", got[0].Path, brokenFile)
	}
	if !strings.Contains(got[0].Message, "no such file or directory") {
		t.Fatalf("CheckSkillDir()[0].Message = %q, want read error", got[0].Message)
	}
}

func TestFormatIssuesNilPassed(t *testing.T) {
	if got := FormatIssues(nil); got != "doctor passed" {
		t.Fatalf("FormatIssues(nil) = %q, want %q", got, "doctor passed")
	}
}

func TestFormatIssuesNonEmpty(t *testing.T) {
	got := FormatIssues([]Issue{{
		Path:    "/repo/skill/SKILL.md",
		Message: "git conflict marker found",
	}})
	wantLine := "/repo/skill/SKILL.md: git conflict marker found\n"

	if !strings.Contains(got, wantLine) {
		t.Fatalf("FormatIssues() = %q, want substring %q", got, wantLine)
	}
}
