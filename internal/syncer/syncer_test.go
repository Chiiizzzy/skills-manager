package syncer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "nested", "file.txt")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}
	if err := os.WriteFile(srcFile, []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() error = %v, want nil", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}
	if string(got) != "content" {
		t.Fatalf("copied file content = %q, want %q", got, "content")
	}
}

func TestCopyDirOverwritesExistingFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "file.txt")
	dstFile := filepath.Join(dst, "file.txt")
	if err := os.WriteFile(srcFile, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile(src) error = %v, want nil", err)
	}
	if err := os.WriteFile(dstFile, []byte("old"), 0644); err != nil {
		t.Fatalf("WriteFile(dst) error = %v, want nil", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() error = %v, want nil", err)
	}

	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil", err)
	}
	if string(got) != "new" {
		t.Fatalf("copied file content = %q, want %q", got, "new")
	}
}

func TestTimestampFormat(t *testing.T) {
	got := Timestamp()

	if len(got) != len("20060102T150405Z") {
		t.Fatalf("Timestamp() length = %d, want %d", len(got), len("20060102T150405Z"))
	}
	if !strings.HasSuffix(got, "Z") {
		t.Fatalf("Timestamp() = %q, want suffix Z", got)
	}
	if got[8] != 'T' {
		t.Fatalf("Timestamp() = %q, want T separator at index 8", got)
	}
}
