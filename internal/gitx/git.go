package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Runner struct{}

func (Runner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	return execGit(ctx, dir, baseConfigArgs(), args...)
}

func execGit(ctx context.Context, dir string, configs []string, args ...string) (string, error) {
	gitArgs := make([]string, 0, len(configs)+len(args))
	gitArgs = append(gitArgs, configs...)
	gitArgs = append(gitArgs, args...)

	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(gitArgs, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func (r Runner) CloneOrFetch(ctx context.Context, repo, ref, dir string) error {
	desiredRepo := normalizeRepoURL(repo)
	if ok, err := r.isRepoRoot(ctx, dir); err != nil {
		return err
	} else if ok {
		if err := r.ensureOrigin(ctx, dir, desiredRepo); err != nil {
			return err
		}
		if err := r.configureHTTP(ctx, dir); err != nil {
			return err
		}
		return r.fetch(ctx, dir, ref)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	_, err := r.runNetwork(ctx, "", "clone", "--no-checkout", "--no-tags", desiredRepo, dir)
	if err != nil {
		return err
	}
	if err := r.configureHTTP(ctx, dir); err != nil {
		return err
	}
	return r.fetch(ctx, dir, ref)
}

func (r Runner) isRepoRoot(ctx context.Context, dir string) (bool, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return false, nil
	}
	topLevel, err := filepath.Abs(strings.TrimSpace(out))
	if err != nil {
		return false, err
	}
	target, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	return filepath.Clean(topLevel) == filepath.Clean(target), nil
}

func (r Runner) Resolve(ctx context.Context, dir, rev string) (string, error) {
	out, err := r.Run(ctx, dir, "rev-parse", "refs/remotes/origin/"+rev)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) fetch(ctx context.Context, dir, ref string) error {
	_, err := r.runNetwork(ctx, dir, "fetch", "origin", fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", ref, ref))
	return err
}

func (r Runner) runNetwork(ctx context.Context, dir string, args ...string) (string, error) {
	configs := append(baseConfigArgs(), networkConfigArgs()...)
	return execGit(ctx, dir, configs, args...)
}

func baseConfigArgs() []string {
	return []string{
		"-c", "gc.auto=0",
		"-c", "maintenance.auto=false",
	}
}

func networkConfigArgs() []string {
	return []string{"-c", "http.version=HTTP/1.1"}
}

func (r Runner) ensureOrigin(ctx context.Context, dir, repo string) error {
	out, err := r.Run(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		_, addErr := r.Run(ctx, dir, "remote", "add", "origin", repo)
		return addErr
	}
	current := strings.TrimSpace(out)
	if current == repo {
		return nil
	}
	_, err = r.Run(ctx, dir, "remote", "set-url", "origin", repo)
	return err
}

func (r Runner) configureHTTP(ctx context.Context, dir string) error {
	_, err := r.Run(ctx, dir, "config", "http.version", "HTTP/1.1")
	return err
}

func normalizeRepoURL(repo string) string {
	trimmed := strings.TrimSpace(repo)
	switch {
	case strings.HasPrefix(trimmed, "git@github.com:"):
		return "https://github.com/" + strings.TrimPrefix(trimmed, "git@github.com:")
	case strings.HasPrefix(trimmed, "ssh://git@github.com/"):
		return "https://github.com/" + strings.TrimPrefix(trimmed, "ssh://git@github.com/")
	default:
		return trimmed
	}
}
