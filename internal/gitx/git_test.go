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

func TestNormalizeRepoURLPrefersHTTPSForGitHub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		repo string
		want string
	}{
		{
			name: "github scp style",
			repo: "git@github.com:obra/superpowers.git",
			want: "https://github.com/obra/superpowers.git",
		},
		{
			name: "github ssh scheme",
			repo: "ssh://git@github.com/obra/superpowers.git",
			want: "https://github.com/obra/superpowers.git",
		},
		{
			name: "github https unchanged",
			repo: "https://github.com/obra/superpowers.git",
			want: "https://github.com/obra/superpowers.git",
		},
		{
			name: "local path unchanged",
			repo: "/tmp/remote.git",
			want: "/tmp/remote.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeRepoURL(tt.repo); got != tt.want {
				t.Fatalf("normalizeRepoURL(%q) = %q, want %q", tt.repo, got, tt.want)
			}
		})
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

func TestCloneOrFetchUpdatesOriginURLToConfiguredRepo(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	sourceDir := filepath.Join(tmp, "sources", "repo")

	remoteA, workA := initRemoteRepo(t, runner, ctx, filepath.Join(tmp, "remote-a"))
	firstCommit := commitAndPushMain(t, runner, ctx, workA, "first\n", "first commit")

	if err := runner.CloneOrFetch(ctx, remoteA, "main", sourceDir); err != nil {
		t.Fatal(err)
	}
	if resolved := resolveGit(t, runner, ctx, sourceDir, "main"); resolved != firstCommit {
		t.Fatalf("resolved first commit = %s, want %s", resolved, firstCommit)
	}

	writeFile(t, filepath.Join(workA, "README.md"), "second\n")
	runGit(t, runner, ctx, workA, "add", "README.md")
	runGit(t, runner, ctx, workA, "commit", "-m", "second commit")
	runGit(t, runner, ctx, workA, "push", "origin", "main")
	secondCommit := gitOutput(t, runner, ctx, workA, "rev-parse", "HEAD")

	remoteB, workB := initRemoteRepo(t, runner, ctx, filepath.Join(tmp, "remote-b"))
	_ = commitAndPushMain(t, runner, ctx, workB, "other\n", "other commit")

	runGit(t, runner, ctx, sourceDir, "remote", "set-url", "origin", remoteB)

	if err := runner.CloneOrFetch(ctx, remoteA, "main", sourceDir); err != nil {
		t.Fatal(err)
	}

	if got := gitOutput(t, runner, ctx, sourceDir, "remote", "get-url", "origin"); got != remoteA {
		t.Fatalf("origin url = %s, want %s", got, remoteA)
	}
	if resolved := resolveGit(t, runner, ctx, sourceDir, "main"); resolved != secondCommit {
		t.Fatalf("resolved updated commit = %s, want %s", resolved, secondCommit)
	}
}

func TestEnsureOriginUpdatesRemoteURL(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	repoDir := filepath.Join(tmp, "repo")

	runGit(t, runner, ctx, "", "init", repoDir)
	runGit(t, runner, ctx, repoDir, "remote", "add", "origin", "/tmp/old.git")

	if err := runner.ensureOrigin(ctx, repoDir, "/tmp/new.git"); err != nil {
		t.Fatal(err)
	}
	if got := gitOutput(t, runner, ctx, repoDir, "remote", "get-url", "origin"); got != "/tmp/new.git" {
		t.Fatalf("origin url = %s, want %s", got, "/tmp/new.git")
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
