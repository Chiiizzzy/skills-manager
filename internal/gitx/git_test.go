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

func TestCloneOrFetchClonesExistingChildDirInsideParentRepo(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	remoteDir, workDir := initRemoteRepo(t, runner, ctx, tmp)
	commit := commitAndPushMain(t, runner, ctx, workDir, "first\n", "first commit")

	parentDir := filepath.Join(tmp, "parent")
	sourceDir := filepath.Join(parentDir, "sources", "repo")
	runGit(t, runner, ctx, "", "init", parentDir)
	runGit(t, runner, ctx, parentDir, "remote", "add", "origin", remoteDir)
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := runner.CloneOrFetch(ctx, remoteDir, "main", sourceDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, ".git")); err != nil {
		t.Fatalf("target directory was not cloned as its own repo: %v", err)
	}
	resolved := resolveGit(t, runner, ctx, sourceDir, "main")
	if resolved != commit {
		t.Fatalf("resolved commit = %s, want %s", resolved, commit)
	}
}

func TestCloneOrFetchBranchRefDoesNotFallbackToTag(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	remoteDir, workDir := initRemoteRepo(t, runner, ctx, tmp)
	sourceDir := filepath.Join(tmp, "source")

	writeFile(t, filepath.Join(workDir, "README.md"), "tagged\n")
	runGit(t, runner, ctx, workDir, "add", "README.md")
	runGit(t, runner, ctx, workDir, "commit", "-m", "tagged commit")
	tagCommit := gitOutput(t, runner, ctx, workDir, "rev-parse", "HEAD")
	runGit(t, runner, ctx, workDir, "tag", "main", tagCommit)
	runGit(t, runner, ctx, workDir, "push", "origin", "refs/tags/main")
	if heads := gitOutput(t, runner, ctx, "", "ls-remote", "--heads", remoteDir, "main"); heads != "" {
		t.Fatalf("remote unexpectedly has branch main: %s", heads)
	}

	runGit(t, runner, ctx, "", "init", sourceDir)
	runGit(t, runner, ctx, sourceDir, "remote", "add", "origin", remoteDir)
	if err := runner.CloneOrFetch(ctx, remoteDir, "main", sourceDir); err == nil {
		t.Fatal("CloneOrFetch succeeded by falling back from branch main to tag main")
	}
	if _, err := runner.Run(ctx, sourceDir, "rev-parse", "refs/tags/main"); err == nil {
		t.Fatal("tag main was fetched despite branch main fetch failure")
	}
}

func TestResolveBranchRefDoesNotFallbackToTag(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	repoDir := filepath.Join(tmp, "repo")

	runGit(t, runner, ctx, "", "init", repoDir)
	runGit(t, runner, ctx, repoDir, "config", "user.name", "Test User")
	runGit(t, runner, ctx, repoDir, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(repoDir, "README.md"), "tagged\n")
	runGit(t, runner, ctx, repoDir, "add", "README.md")
	runGit(t, runner, ctx, repoDir, "commit", "-m", "tagged commit")
	runGit(t, runner, ctx, repoDir, "tag", "main", "HEAD")

	if resolved, err := runner.Resolve(ctx, repoDir, "main"); err == nil {
		t.Fatalf("Resolve fell back from remote branch main to tag main: %s", resolved)
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

func initRemoteRepo(t *testing.T, runner Runner, ctx context.Context, tmp string) (remoteDir string, workDir string) {
	t.Helper()
	remoteDir = filepath.Join(tmp, "remote.git")
	workDir = filepath.Join(tmp, "work")
	runGit(t, runner, ctx, "", "init", "--bare", remoteDir)
	runGit(t, runner, ctx, "", "init", workDir)
	runGit(t, runner, ctx, workDir, "config", "user.name", "Test User")
	runGit(t, runner, ctx, workDir, "config", "user.email", "test@example.com")
	runGit(t, runner, ctx, workDir, "remote", "add", "origin", remoteDir)
	return remoteDir, workDir
}

func commitAndPushMain(t *testing.T, runner Runner, ctx context.Context, workDir, content, message string) string {
	t.Helper()
	writeFile(t, filepath.Join(workDir, "README.md"), content)
	runGit(t, runner, ctx, workDir, "add", "README.md")
	runGit(t, runner, ctx, workDir, "commit", "-m", message)
	runGit(t, runner, ctx, workDir, "branch", "-M", "main")
	runGit(t, runner, ctx, workDir, "push", "-u", "origin", "main")
	return gitOutput(t, runner, ctx, workDir, "rev-parse", "HEAD")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
