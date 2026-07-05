package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Runner struct{}

var bareVersionTagRefRE = regexp.MustCompile(`^v?[0-9]+[.][0-9]+[.][0-9]+([-+][0-9A-Za-z][0-9A-Za-z.-]*)?$`)

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
	if _, remoteRef, ok := remoteBranchRef(rev); ok {
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
	if branch, remoteRef, ok := remoteBranchRef(ref); ok {
		if _, err := r.Run(ctx, dir, "fetch", "origin", fmt.Sprintf("+refs/heads/%s:%s", branch, remoteRef)); err == nil {
			return nil
		}
	}
	if _, fullRef, ok := tagRef(ref); ok {
		if _, err := r.Run(ctx, dir, "fetch", "origin", fmt.Sprintf("+%s:%s", fullRef, fullRef)); err == nil {
			return nil
		}
	}
	_, err := r.Run(ctx, dir, "fetch", "origin", ref)
	return err
}

func remoteBranchRef(ref string) (branch string, remoteRef string, ok bool) {
	if ref == "" || isHexSHA(ref) {
		return "", "", false
	}
	if _, _, ok := tagRef(ref); ok {
		return "", "", false
	}
	if strings.ContainsAny(ref, "~^:") || strings.IndexFunc(ref, unicode.IsSpace) >= 0 {
		return "", "", false
	}

	switch {
	case strings.HasPrefix(ref, "refs/remotes/origin/"):
		branch = strings.TrimPrefix(ref, "refs/remotes/origin/")
	case strings.HasPrefix(ref, "refs/heads/"):
		branch = strings.TrimPrefix(ref, "refs/heads/")
	case strings.HasPrefix(ref, "origin/"):
		branch = strings.TrimPrefix(ref, "origin/")
	case strings.HasPrefix(ref, "refs/"):
		return "", "", false
	default:
		branch = ref
	}
	if branch == "" {
		return "", "", false
	}
	return branch, "refs/remotes/origin/" + branch, true
}

func tagRef(ref string) (tag string, fullRef string, ok bool) {
	if ref == "" || isHexSHA(ref) || strings.ContainsAny(ref, "~^:") || strings.IndexFunc(ref, unicode.IsSpace) >= 0 {
		return "", "", false
	}
	if strings.HasPrefix(ref, "refs/tags/") {
		tag = strings.TrimPrefix(ref, "refs/tags/")
		if tag == "" || strings.HasPrefix(tag, "/") {
			return "", "", false
		}
		return tag, "refs/tags/" + tag, true
	}
	if bareVersionTagRefRE.MatchString(ref) {
		return ref, "refs/tags/" + ref, true
	}
	return "", "", false
}

func isHexSHA(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, r := range ref {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
