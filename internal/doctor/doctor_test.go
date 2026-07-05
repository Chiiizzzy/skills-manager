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
