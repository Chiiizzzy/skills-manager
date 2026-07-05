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
	if _, err := r.Run(ctx, dir, "rev-parse", "--git-dir"); err == nil {
		return r.fetch(ctx, dir, ref)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return err
	}
	_, err := r.Run(ctx, "", "clone", "--no-checkout", repo, dir)
	if err != nil {
		return err
	}
	return r.fetch(ctx, dir, ref)
}

func (r Runner) Resolve(ctx context.Context, dir, rev string) (string, error) {
	if remoteRef, ok := remoteTrackingRef(rev); ok {
		out, err := r.Run(ctx, dir, "rev-parse", remoteRef)
		if err == nil {
			return strings.TrimSpace(out), nil
		}
	}
	out, err := r.Run(ctx, dir, "rev-parse", rev)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) fetch(ctx context.Context, dir, ref string) error {
	if remoteRef, ok := remoteTrackingRef(ref); ok {
		if _, err := r.Run(ctx, dir, "fetch", "origin", fmt.Sprintf("+refs/heads/%s:%s", ref, remoteRef)); err == nil {
			return nil
		}
	}
	_, err := r.Run(ctx, dir, "fetch", "origin", ref)
	return err
}

func remoteTrackingRef(ref string) (string, bool) {
	if strings.HasPrefix(ref, "origin/") || strings.HasPrefix(ref, "refs/") {
		return "", false
	}
	return "refs/remotes/origin/" + ref, true
}
