package patch

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type fakeGitRunner struct {
	output string
	err    error
	calls  []gitCall
}

type gitCall struct {
	dir  string
	args []string
}

func (f *fakeGitRunner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	f.calls = append(f.calls, gitCall{
		dir:  dir,
		args: append([]string(nil), args...),
	})
	return f.output, f.err
}

func TestApplyEmptyPatchFileDoesNotCallGit(t *testing.T) {
	git := &fakeGitRunner{}
	service := Service{Git: git}

	if err := service.Apply(context.Background(), "/repo", ""); err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}
	if len(git.calls) != 0 {
		t.Fatalf("git calls = %d, want 0", len(git.calls))
	}
}

func TestApplyPatchFileRunsGitApply3Way(t *testing.T) {
	git := &fakeGitRunner{}
	service := Service{Git: git}

	if err := service.Apply(context.Background(), "/repo", "/patches/local.patch"); err != nil {
		t.Fatalf("Apply() error = %v, want nil", err)
	}

	if len(git.calls) != 1 {
		t.Fatalf("git calls = %d, want 1", len(git.calls))
	}
	call := git.calls[0]
	if call.dir != "/repo" {
		t.Fatalf("git dir = %q, want %q", call.dir, "/repo")
	}
	wantArgs := []string{"apply", "--3way", "/patches/local.patch"}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Fatalf("git args = %#v, want %#v", call.args, wantArgs)
	}
}

func TestRefreshRequiresChanges(t *testing.T) {
	git := &fakeGitRunner{}
	service := Service{Git: git}

	_, err := service.Refresh(context.Background(), "/repo", "/patches/local.patch")
	if err == nil {
		t.Fatal("Refresh() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "no local changes") {
		t.Fatalf("Refresh() error = %q, want no local changes error", err)
	}
}

func TestRefreshReturnsDiffOutput(t *testing.T) {
	diff := "diff --git a/SKILL.md b/SKILL.md\n"
	git := &fakeGitRunner{output: diff}
	service := Service{Git: git}

	got, err := service.Refresh(context.Background(), "/repo", "/patches/local.patch")
	if err != nil {
		t.Fatalf("Refresh() error = %v, want nil", err)
	}
	if got != diff {
		t.Fatalf("Refresh() output = %q, want %q", got, diff)
	}

	if len(git.calls) != 1 {
		t.Fatalf("git calls = %d, want 1", len(git.calls))
	}
	call := git.calls[0]
	if call.dir != "/repo" {
		t.Fatalf("git dir = %q, want %q", call.dir, "/repo")
	}
	wantArgs := []string{"diff", "--binary"}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Fatalf("git args = %#v, want %#v", call.args, wantArgs)
	}
}
