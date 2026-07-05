package gitx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitAvailable(t *testing.T) {
	requireGit(t)
	runner := Runner{}
	if _, err := runner.Run(context.Background(), "", "--version"); err != nil {
		t.Fatal(err)
	}
}

func TestCloneOrFetchRefreshesRemoteBranch(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	remoteDir := filepath.Join(tmp, "remote.git")
	workDir := filepath.Join(tmp, "work")
	sourceDir := filepath.Join(tmp, "sources", "repo")

	runGit(t, runner, ctx, "", "init", "--bare", remoteDir)
	runGit(t, runner, ctx, "", "init", workDir)
	runGit(t, runner, ctx, workDir, "config", "user.name", "Test User")
	runGit(t, runner, ctx, workDir, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(workDir, "README.md"), "first\n")
	runGit(t, runner, ctx, workDir, "add", "README.md")
	runGit(t, runner, ctx, workDir, "commit", "-m", "first commit")
	runGit(t, runner, ctx, workDir, "branch", "-M", "main")
	runGit(t, runner, ctx, workDir, "remote", "add", "origin", remoteDir)
	runGit(t, runner, ctx, workDir, "push", "-u", "origin", "main")
	firstCommit := gitOutput(t, runner, ctx, workDir, "rev-parse", "HEAD")

	if err := runner.CloneOrFetch(ctx, remoteDir, "main", sourceDir); err != nil {
		t.Fatal(err)
	}
	resolvedFirst := resolveGit(t, runner, ctx, sourceDir, "main")
	if resolvedFirst != firstCommit {
		t.Fatalf("resolved first commit = %s, want %s", resolvedFirst, firstCommit)
	}
	runGit(t, runner, ctx, sourceDir, "branch", "-f", "main", firstCommit)

	writeFile(t, filepath.Join(workDir, "README.md"), "second\n")
	runGit(t, runner, ctx, workDir, "add", "README.md")
	runGit(t, runner, ctx, workDir, "commit", "-m", "second commit")
	runGit(t, runner, ctx, workDir, "push", "origin", "main")
	secondCommit := gitOutput(t, runner, ctx, workDir, "rev-parse", "HEAD")

	if err := runner.CloneOrFetch(ctx, remoteDir, "main", sourceDir); err != nil {
		t.Fatal(err)
	}
	resolvedSecond := resolveGit(t, runner, ctx, sourceDir, "main")
	if resolvedSecond != secondCommit {
		t.Fatalf("resolved second commit = %s, want %s", resolvedSecond, secondCommit)
	}
	if resolvedSecond == firstCommit {
		t.Fatalf("resolved stale first commit after fetch: %s", resolvedSecond)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func runGit(t *testing.T, runner Runner, ctx context.Context, dir string, args ...string) {
	t.Helper()
	if _, err := runner.Run(ctx, dir, args...); err != nil {
		t.Fatal(err)
	}
}

func gitOutput(t *testing.T, runner Runner, ctx context.Context, dir string, args ...string) string {
	t.Helper()
	out, err := runner.Run(ctx, dir, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

func resolveGit(t *testing.T, runner Runner, ctx context.Context, dir, rev string) string {
	t.Helper()
	out, err := runner.Resolve(ctx, dir, rev)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
