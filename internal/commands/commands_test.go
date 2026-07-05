package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCommand(t *testing.T) {
	root := writeCommandTestConfig(t)
	writeSkillFile(t, filepath.Join(root, "dist", "brainstorming", "SKILL.md"), "hello")

	out, err := executeCommand("--root", root, "status")
	if err != nil {
		t.Fatalf("status error = %v, want nil", err)
	}
	for _, want := range []string{
		"Sources:",
		"superpowers",
		"Skills:",
		"brainstorming",
		"dist=ready",
		"Profiles:",
		"test",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output = %q, want substring %q", out, want)
		}
	}
}

func TestDoctorCommandReportsIssues(t *testing.T) {
	root := writeCommandTestConfig(t)

	out, err := executeCommand("--root", root, "doctor")
	if err == nil {
		t.Fatal("doctor error = nil, want issue error")
	}
	for _, want := range []string{"SKILL.md is missing", "doctor found 1 issue"} {
		if !strings.Contains(out+err.Error(), want) {
			t.Fatalf("doctor output/error = %q / %q, want substring %q", out, err, want)
		}
	}
}

func TestSyncCommandDryRun(t *testing.T) {
	root := writeCommandTestConfig(t)
	writeSkillFile(t, filepath.Join(root, "dist", "brainstorming", "SKILL.md"), "hello")

	out, err := executeCommand("--root", root, "sync", "--profile", "test", "--dry-run")
	if err != nil {
		t.Fatalf("sync --dry-run error = %v, want nil", err)
	}
	if !strings.Contains(out, "would sync") {
		t.Fatalf("sync --dry-run output = %q, want dry-run line", out)
	}
	target := filepath.Join(root, "target", "brainstorming")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target stat error = %v, want not exist", err)
	}
}

func TestSyncAndRollbackCommands(t *testing.T) {
	root := writeCommandTestConfig(t)
	distFile := filepath.Join(root, "dist", "brainstorming", "SKILL.md")
	targetFile := filepath.Join(root, "target", "brainstorming", "SKILL.md")
	writeSkillFile(t, distFile, "new")
	writeSkillFile(t, targetFile, "old")

	if out, err := executeCommand("--root", root, "sync", "--profile", "test"); err != nil {
		t.Fatalf("sync output = %q error = %v, want nil", out, err)
	}
	if got := readFile(t, targetFile); got != "new" {
		t.Fatalf("target after sync = %q, want new", got)
	}
	if err := os.WriteFile(targetFile, []byte("broken"), 0o644); err != nil {
		t.Fatalf("mutate target: %v", err)
	}
	if out, err := executeCommand("--root", root, "rollback", "--profile", "test"); err != nil {
		t.Fatalf("rollback output = %q error = %v, want nil", out, err)
	}
	if got := readFile(t, targetFile); got != "old" {
		t.Fatalf("target after rollback = %q, want old", got)
	}
}

func TestUpdateDiffPatchRefreshCommands(t *testing.T) {
	requireGit(t)

	tmp := t.TempDir()
	remoteDir := filepath.Join(tmp, "remote.git")
	workDir := filepath.Join(tmp, "work")
	root := filepath.Join(tmp, "root")

	runGitCommand(t, "", "init", "--bare", remoteDir)
	runGitCommand(t, "", "init", workDir)
	runGitCommand(t, workDir, "config", "user.name", "Test User")
	runGitCommand(t, workDir, "config", "user.email", "test@example.com")
	writeSkillFile(t, filepath.Join(workDir, "skills", "brainstorming", "SKILL.md"), "# Skill\n")
	runGitCommand(t, workDir, "add", ".")
	runGitCommand(t, workDir, "commit", "-m", "initial")
	runGitCommand(t, workDir, "branch", "-M", "main")
	runGitCommand(t, workDir, "remote", "add", "origin", remoteDir)
	runGitCommand(t, workDir, "push", "-u", "origin", "main")

	writeCommandTestConfigWithRepo(t, root, remoteDir)
	if _, err := executeCommand("--root", root, "update", "--all"); err != nil {
		t.Fatalf("update --all error = %v, want nil", err)
	}
	distFile := filepath.Join(root, "dist", "brainstorming", "SKILL.md")
	if got := readFile(t, distFile); got != "# Skill\n" {
		t.Fatalf("dist file = %q, want initial skill", got)
	}

	if err := os.WriteFile(distFile, []byte("# Skill\n\nlocal edit\n"), 0o644); err != nil {
		t.Fatalf("edit dist: %v", err)
	}
	diffOut, err := executeCommand("--root", root, "diff", "brainstorming")
	if err != nil {
		t.Fatalf("diff error = %v, want nil", err)
	}
	if !strings.Contains(diffOut, "local edit") {
		t.Fatalf("diff output = %q, want local edit", diffOut)
	}
	if _, err := executeCommand("--root", root, "patch", "refresh", "brainstorming"); err != nil {
		t.Fatalf("patch refresh error = %v, want nil", err)
	}
	if got := readFile(t, filepath.Join(root, "patches", "brainstorming", "local.patch")); !strings.Contains(got, "local edit") {
		t.Fatalf("patch file = %q, want local edit", got)
	}

	writeSkillFile(t, filepath.Join(workDir, "skills", "brainstorming", "README.md"), "upstream\n")
	runGitCommand(t, workDir, "add", ".")
	runGitCommand(t, workDir, "commit", "-m", "upstream")
	runGitCommand(t, workDir, "push", "origin", "main")
	if _, err := executeCommand("--root", root, "update", "--all"); err != nil {
		t.Fatalf("update --all with patch error = %v, want nil", err)
	}
	if got := readFile(t, distFile); !strings.Contains(got, "local edit") {
		t.Fatalf("dist file after patch replay = %q, want local edit", got)
	}
	if got := readFile(t, filepath.Join(root, "dist", "brainstorming", "README.md")); got != "upstream\n" {
		t.Fatalf("upstream file = %q, want upstream", got)
	}
}

func TestUpdatePreservesConflictedDist(t *testing.T) {
	requireGit(t)

	tmp := t.TempDir()
	remoteDir := filepath.Join(tmp, "remote.git")
	workDir := filepath.Join(tmp, "work")
	root := filepath.Join(tmp, "root")

	runGitCommand(t, "", "init", "--bare", remoteDir)
	runGitCommand(t, "", "init", workDir)
	runGitCommand(t, workDir, "config", "user.name", "Test User")
	runGitCommand(t, workDir, "config", "user.email", "test@example.com")
	writeSkillFile(t, filepath.Join(workDir, "skills", "brainstorming", "SKILL.md"), "# Skill\n")
	runGitCommand(t, workDir, "add", ".")
	runGitCommand(t, workDir, "commit", "-m", "initial")
	runGitCommand(t, workDir, "branch", "-M", "main")
	runGitCommand(t, workDir, "remote", "add", "origin", remoteDir)
	runGitCommand(t, workDir, "push", "-u", "origin", "main")

	writeCommandTestConfigWithRepo(t, root, remoteDir)
	if _, err := executeCommand("--root", root, "update", "--all"); err != nil {
		t.Fatalf("update --all error = %v, want nil", err)
	}
	distFile := filepath.Join(root, "dist", "brainstorming", "SKILL.md")
	if err := os.WriteFile(distFile, []byte("# Skill\n\nlocal edit\n"), 0o644); err != nil {
		t.Fatalf("edit dist: %v", err)
	}
	if _, err := executeCommand("--root", root, "patch", "refresh", "brainstorming"); err != nil {
		t.Fatalf("patch refresh error = %v, want nil", err)
	}

	writeSkillFile(t, filepath.Join(workDir, "skills", "brainstorming", "SKILL.md"), "# Skill\nupstream edit\n")
	runGitCommand(t, workDir, "add", ".")
	runGitCommand(t, workDir, "commit", "-m", "conflicting-upstream")
	runGitCommand(t, workDir, "push", "origin", "main")
	if _, err := executeCommand("--root", root, "update", "--all"); err == nil {
		t.Fatal("conflicting update error = nil, want error")
	}
	if got := readFile(t, distFile); !strings.Contains(got, "<<<<<<<") {
		t.Fatalf("conflicted dist = %q, want conflict marker", got)
	}
}

func executeCommand(args ...string) (string, error) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func writeCommandTestConfig(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCommandTestConfigWithRepo(t, root, "/tmp/nonexistent.git")
	return root
}

func writeCommandTestConfigWithRepo(t *testing.T, root, repo string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	data := `sources:
  superpowers:
    repo: ` + repo + `
    ref: main
    skills:
      brainstorming:
        path: skills/brainstorming
profiles:
  test:
    target: ` + filepath.ToSlash(filepath.Join(root, "target")) + `
    skills:
      - brainstorming
`
	if err := os.WriteFile(filepath.Join(root, "skillctl.yaml"), []byte(data), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func writeSkillFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s error = %v output = %s", strings.Join(args, " "), err, out)
	}
}
