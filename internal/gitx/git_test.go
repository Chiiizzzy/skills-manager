package gitx

import (
	"context"
	"os/exec"
	"testing"
)

func TestGitAvailable(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	runner := Runner{}
	if _, err := runner.Run(context.Background(), "", "--version"); err != nil {
		t.Fatal(err)
	}
}
