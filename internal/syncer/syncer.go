package syncer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	DryRun bool
}

type Plan struct {
	Files []string
}

func CopyDir(src, dst string) error {
	if err := validateCopyDirPaths(src, dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func validateCopyDirPaths(src, dst string) error {
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	srcAbs = filepath.Clean(srcAbs)
	dstAbs = filepath.Clean(dstAbs)
	if dstAbs == srcAbs {
		return fmt.Errorf("copy destination %q must differ from source %q", dstAbs, srcAbs)
	}
	rel, err := filepath.Rel(srcAbs, dstAbs)
	if err != nil {
		return err
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("copy destination %q must be outside source %q", dstAbs, srcAbs)
	}
	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	perm := info.Mode().Perm()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, perm)
}

func Timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
