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
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func (r Runner) CloneOrFetch(ctx context.Context, repo, ref, dir string) error {
	if ok, err := r.isRepoRoot(ctx, dir); err != nil {
		return err
	} else if ok {
		return r.fetch(ctx, dir, ref)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	_, err := r.Run(ctx, "", "clone", "--no-checkout", "--no-tags", repo, dir)
	if err != nil {
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
	_, err := r.Run(ctx, dir, "fetch", "origin", fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", ref, ref))
	return err
}
