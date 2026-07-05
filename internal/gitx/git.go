package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
	if _, err := r.Run(ctx, dir, "rev-parse", "--git-dir"); err == nil {
		_, err = r.Run(ctx, dir, "fetch", "origin", ref)
		return err
	}
	_, err := r.Run(ctx, "", "clone", "--no-checkout", repo, dir)
	if err != nil {
		return err
	}
	_, err = r.Run(ctx, dir, "fetch", "origin", ref)
	return err
}

func (r Runner) Resolve(ctx context.Context, dir, rev string) (string, error) {
	out, err := r.Run(ctx, dir, "rev-parse", rev)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
