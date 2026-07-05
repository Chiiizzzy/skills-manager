package gitx

import (
	"context"
	"fmt"
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

func TestRemoteBranchRef(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"

	tests := []struct {
		name       string
		ref        string
		wantBranch string
		wantRemote string
		wantOK     bool
	}{
		{
			name:       "bare branch",
			ref:        "main",
			wantBranch: "main",
			wantRemote: "refs/remotes/origin/main",
			wantOK:     true,
		},
		{
			name:       "origin branch",
			ref:        "origin/main",
			wantBranch: "main",
			wantRemote: "refs/remotes/origin/main",
			wantOK:     true,
		},
		{
			name:       "heads ref",
			ref:        "refs/heads/main",
			wantBranch: "main",
			wantRemote: "refs/remotes/origin/main",
			wantOK:     true,
		},
		{
			name:       "remote tracking ref",
			ref:        "refs/remotes/origin/main",
			wantBranch: "main",
			wantRemote: "refs/remotes/origin/main",
			wantOK:     true,
		},
		{name: "tag ref", ref: "refs/tags/v1.0.0"},
		{name: "sha", ref: sha},
		{name: "tilde expression", ref: "main~1"},
		{name: "caret expression", ref: "main^"},
		{name: "colon expression", ref: "feature:foo"},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, remoteRef, ok := remoteBranchRef(tt.ref)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if branch != tt.wantBranch {
				t.Fatalf("branch = %q, want %q", branch, tt.wantBranch)
			}
			if remoteRef != tt.wantRemote {
				t.Fatalf("remoteRef = %q, want %q", remoteRef, tt.wantRemote)
			}
		})
	}
}

func TestCloneOrFetchResolvesRemoteBranchAliases(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	remoteDir, workDir := initRemoteRepo(t, runner, ctx, tmp)
	refs := []string{
		"main",
		"origin/main",
		"refs/heads/main",
		"refs/remotes/origin/main",
	}

	firstCommit := commitAndPushMain(t, runner, ctx, workDir, "first\n", "first commit")
	for i, ref := range refs {
		sourceDir := filepath.Join(tmp, "sources", fmt.Sprintf("repo-%d", i))
		if err := runner.CloneOrFetch(ctx, remoteDir, ref, sourceDir); err != nil {
			t.Fatalf("%s: clone/fetch first commit: %v", ref, err)
		}
		resolved := resolveGit(t, runner, ctx, sourceDir, ref)
		if resolved != firstCommit {
			t.Fatalf("%s: resolved first commit = %s, want %s", ref, resolved, firstCommit)
		}
		runGit(t, runner, ctx, sourceDir, "branch", "-f", "main", firstCommit)
	}

	secondCommit := commitAndPushMain(t, runner, ctx, workDir, "second\n", "second commit")
	for i, ref := range refs {
		sourceDir := filepath.Join(tmp, "sources", fmt.Sprintf("repo-%d", i))
		if err := runner.CloneOrFetch(ctx, remoteDir, ref, sourceDir); err != nil {
			t.Fatalf("%s: clone/fetch second commit: %v", ref, err)
		}
		resolved := resolveGit(t, runner, ctx, sourceDir, ref)
		if resolved != secondCommit {
			t.Fatalf("%s: resolved second commit = %s, want %s", ref, resolved, secondCommit)
		}
		if resolved == firstCommit {
			t.Fatalf("%s: resolved stale first commit after fetch: %s", ref, resolved)
		}
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
