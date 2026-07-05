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

func TestResolveTagRefDoesNotFallbackToBranch(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	repoDir := filepath.Join(tmp, "repo")

	runGit(t, runner, ctx, "", "init", repoDir)
	runGit(t, runner, ctx, repoDir, "config", "user.name", "Test User")
	runGit(t, runner, ctx, repoDir, "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(repoDir, "README.md"), "branch\n")
	runGit(t, runner, ctx, repoDir, "add", "README.md")
	runGit(t, runner, ctx, repoDir, "commit", "-m", "branch commit")
	runGit(t, runner, ctx, repoDir, "branch", "v1.0.0", "HEAD")

	if resolved, err := runner.Resolve(ctx, repoDir, "v1.0.0"); err == nil {
		t.Fatalf("Resolve fell back from tag v1.0.0 to branch v1.0.0: %s", resolved)
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
		{
			name:       "feature branch with slash",
			ref:        "feature/foo",
			wantBranch: "feature/foo",
			wantRemote: "refs/remotes/origin/feature/foo",
			wantOK:     true,
		},
		{name: "tag ref", ref: "refs/tags/v1.0.0"},
		{name: "bare v version tag", ref: "v1.0.0"},
		{name: "bare version tag", ref: "1.0.0"},
		{name: "sha", ref: sha},
		{name: "tilde expression", ref: "main~1"},
		{name: "caret expression", ref: "main^"},
		{name: "colon expression", ref: "feature:foo"},
		{name: "whitespace", ref: "feature foo"},
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

func TestTagRef(t *testing.T) {
	const sha = "0123456789abcdef0123456789abcdef01234567"

	tests := []struct {
		name     string
		ref      string
		wantTag  string
		wantFull string
		wantOK   bool
	}{
		{
			name:     "full tag ref",
			ref:      "refs/tags/v1.0.0",
			wantTag:  "v1.0.0",
			wantFull: "refs/tags/v1.0.0",
			wantOK:   true,
		},
		{
			name:     "full tag ref with slash",
			ref:      "refs/tags/release/v1.0.0",
			wantTag:  "release/v1.0.0",
			wantFull: "refs/tags/release/v1.0.0",
			wantOK:   true,
		},
		{
			name:     "bare v version tag",
			ref:      "v1.0.0",
			wantTag:  "v1.0.0",
			wantFull: "refs/tags/v1.0.0",
			wantOK:   true,
		},
		{
			name:     "bare version tag",
			ref:      "1.0.0",
			wantTag:  "1.0.0",
			wantFull: "refs/tags/1.0.0",
			wantOK:   true,
		},
		{
			name:     "bare prerelease version tag",
			ref:      "v1.0.0-rc.1",
			wantTag:  "v1.0.0-rc.1",
			wantFull: "refs/tags/v1.0.0-rc.1",
			wantOK:   true,
		},
		{name: "bare branch", ref: "main"},
		{name: "origin branch", ref: "origin/main"},
		{name: "heads ref", ref: "refs/heads/main"},
		{name: "remote tracking ref", ref: "refs/remotes/origin/main"},
		{name: "sha", ref: sha},
		{name: "tag expression", ref: "v1.0.0^{}"},
		{name: "full tag expression", ref: "refs/tags/v1.0.0^{}"},
		{name: "empty full tag ref", ref: "refs/tags/"},
		{name: "double slash full tag ref", ref: "refs/tags//v1.0.0"},
		{name: "whitespace", ref: "v1.0.0 test"},
		{name: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, fullRef, ok := tagRef(tt.ref)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tag != tt.wantTag {
				t.Fatalf("tag = %q, want %q", tag, tt.wantTag)
			}
			if fullRef != tt.wantFull {
				t.Fatalf("fullRef = %q, want %q", fullRef, tt.wantFull)
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

func TestCloneOrFetchResolvesTags(t *testing.T) {
	requireGit(t)

	ctx := context.Background()
	runner := Runner{}
	tmp := t.TempDir()
	remoteDir, workDir := initRemoteRepo(t, runner, ctx, tmp)
	tagCommit := commitAndPushMain(t, runner, ctx, workDir, "tagged\n", "tagged commit")
	refs := []struct {
		ref     string
		fullRef string
	}{
		{ref: "v1.0.0", fullRef: "refs/tags/v1.0.0"},
		{ref: "refs/tags/v1.0.0", fullRef: "refs/tags/v1.0.0"},
		{ref: "1.0.0", fullRef: "refs/tags/1.0.0"},
	}
	sourceDirs := make([]string, len(refs))

	for i := range refs {
		sourceDir := filepath.Join(tmp, "sources", fmt.Sprintf("repo-%d", i))
		if err := runner.CloneOrFetch(ctx, remoteDir, "main", sourceDir); err != nil {
			t.Fatalf("pre-tag clone: %v", err)
		}
		sourceDirs[i] = sourceDir
	}

	runGit(t, runner, ctx, workDir, "tag", "v1.0.0", tagCommit)
	runGit(t, runner, ctx, workDir, "tag", "1.0.0", tagCommit)
	runGit(t, runner, ctx, workDir, "push", "origin", "refs/tags/v1.0.0", "refs/tags/1.0.0")

	for i, tt := range refs {
		sourceDir := sourceDirs[i]
		if err := runner.CloneOrFetch(ctx, remoteDir, tt.ref, sourceDir); err != nil {
			t.Fatalf("%s: clone/fetch tag: %v", tt.ref, err)
		}
		resolved := resolveGit(t, runner, ctx, sourceDir, tt.ref)
		if resolved != tagCommit {
			t.Fatalf("%s: resolved tag = %s, want %s", tt.ref, resolved, tagCommit)
		}
		localTag := gitOutput(t, runner, ctx, sourceDir, "rev-parse", tt.fullRef)
		if localTag != tagCommit {
			t.Fatalf("%s: local tag ref = %s, want %s", tt.ref, localTag, tagCommit)
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
