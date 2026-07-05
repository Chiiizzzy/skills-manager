package syncer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if err := os.WriteFile(srcFile, []byte("new"), 0755); err != nil {
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
	info, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Stat(dst) error = %v, want nil", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0755); got != want {
		t.Fatalf("copied file mode = %v, want %v", got, want)
	}
}

func TestCopyDirPreservesExecutableBit(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "bin", "run")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v, want nil", err)
	}
	if err := os.WriteFile(srcFile, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v, want nil", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir() error = %v, want nil", err)
	}

	info, err := os.Stat(filepath.Join(dst, "bin", "run"))
	if err != nil {
		t.Fatalf("Stat() error = %v, want nil", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0755); got != want {
		t.Fatalf("copied file mode = %v, want %v", got, want)
	}
}

func TestCopyDirRejectsSelfCopy(t *testing.T) {
	src := t.TempDir()

	if err := CopyDir(src, src); err == nil {
		t.Fatal("CopyDir(src, src) error = nil, want error")
	}
}

func TestCopyDirRejectsNestedDestination(t *testing.T) {
	src := t.TempDir()

	if err := CopyDir(src, filepath.Join(src, "nested-dst")); err == nil {
		t.Fatal("CopyDir(src, nested dst) error = nil, want error")
	}
}

func TestTimestampFormat(t *testing.T) {
	got := Timestamp()

	if _, err := time.Parse("20060102T150405Z", got); err != nil {
		t.Fatalf("Timestamp() = %q, parse error = %v", got, err)
	}
}
