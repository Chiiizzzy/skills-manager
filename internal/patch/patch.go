package patch

import (
	"context"
	"fmt"
)

type GitRunner interface {
	Run(ctx context.Context, dir string, args ...string) (string, error)
}

type Service struct {
	Git GitRunner
}

func (s Service) Apply(ctx context.Context, repoDir, patchFile string) error {
	if patchFile == "" {
		return nil
	}
	_, err := s.Git.Run(ctx, repoDir, "apply", "--3way", patchFile)
	if err != nil {
		return fmt.Errorf("apply patch %s: %w", patchFile, err)
	}
	return nil
}

func (s Service) Refresh(ctx context.Context, repoDir, outputPatch string) (string, error) {
	out, err := s.Git.Run(ctx, repoDir, "diff", "--binary")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("no local changes to refresh patch")
	}
	return out, nil
}
