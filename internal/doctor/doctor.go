package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Issue struct {
	Path    string
	Message string
}

func CheckSkillDir(skillDir string) []Issue {
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		return []Issue{{
			Path:    skillFile,
			Message: "SKILL.md is missing",
		}}
	}

	var issues []Issue
	err := filepath.WalkDir(skillDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, Issue{
				Path:    path,
				Message: err.Error(),
			})
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{
				Path:    path,
				Message: err.Error(),
			})
			return nil
		}
		if hasConflictMarker(string(data)) {
			issues = append(issues, Issue{
				Path:    path,
				Message: "git conflict marker found",
			})
		}
		return nil
	})
	if err != nil {
		issues = append(issues, Issue{
			Path:    skillDir,
			Message: err.Error(),
		})
	}

	return issues
}

func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return "doctor passed"
	}

	var b strings.Builder
	for _, issue := range issues {
		fmt.Fprintf(&b, "%s: %s\n", issue.Path, issue.Message)
	}
	return b.String()
}

func hasConflictMarker(data string) bool {
	return strings.Contains(data, "<<<<<<<") ||
		strings.Contains(data, "=======") ||
		strings.Contains(data, ">>>>>>>")
}
