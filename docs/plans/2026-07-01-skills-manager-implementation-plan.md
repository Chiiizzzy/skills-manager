# Skills Manager 实施方案

> **面向代理执行者：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务执行本计划。步骤使用复选框语法 `- [ ]` 跟踪。

**目标：** 交付一个本地优先的 `skillctl` CLI，用于拉取上游 skills、维护本地 patch、生成可安装 dist，并同步到 agent skills 目录。

**方案概述：** 第一版采用 Go 单二进制实现，配置和 lock 文件使用 YAML，git 操作先通过本机 `git` 命令完成。实现按模块拆分：配置解析、source 拉取、patch 应用、doctor 校验、sync/rollback 安装和 CLI 命令。

**技术栈：** Go、cobra、yaml.v3、本机 git、标准库文件系统 API、table-driven tests

---

## 文件结构

```text
skills-manager/
  go.mod
  cmd/skillctl/main.go
  internal/config/config.go
  internal/config/config_test.go
  internal/gitx/git.go
  internal/gitx/git_test.go
  internal/workspace/layout.go
  internal/workspace/layout_test.go
  internal/patch/patch.go
  internal/patch/patch_test.go
  internal/doctor/doctor.go
  internal/doctor/doctor_test.go
  internal/syncer/syncer.go
  internal/syncer/syncer_test.go
  internal/commands/root.go
  internal/commands/status.go
  internal/commands/update.go
  internal/commands/diff.go
  internal/commands/patch.go
  internal/commands/doctor.go
  internal/commands/sync.go
  internal/commands/rollback.go
  examples/skillctl.yaml.example
  README.md
  docs/specs/2026-07-01-skills-manager-design.md
  docs/plans/2026-07-01-skills-manager-implementation-plan.md
```

## Task 1: 初始化 Go CLI 工程

**Files:**
- Create: `go.mod`
- Create: `cmd/skillctl/main.go`
- Create: `internal/commands/root.go`

- [ ] **Step 1: 初始化 module**

Run:

```bash
go mod init github.com/your-org/skills-manager
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
```

Expected:

```text
go.mod created
```

- [ ] **Step 2: 创建 CLI 入口**

Create `cmd/skillctl/main.go`:

```go
package main

import (
	"os"

	"github.com/your-org/skills-manager/internal/commands"
)

func main() {
	if err := commands.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: 创建 root command**

Create `internal/commands/root.go`:

```go
package commands

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skillctl",
		Short: "Manage upstream skills, local patches, and agent skill installs",
	}

	cmd.AddCommand(newStatusCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDiffCommand())
	cmd.AddCommand(newPatchCommand())
	cmd.AddCommand(newDoctorCommand())
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newRollbackCommand())

	return cmd
}
```

- [ ] **Step 4: 添加空命令骨架**

Create each file under `internal/commands/` with this shape, replacing command names:

```go
package commands

import "github.com/spf13/cobra"

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show managed skills status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
```

For `update`, use `Use: "update [skill]"`; for `diff`, use `Use: "diff [skill]"`; for `patch`, create subcommand `refresh`; for `sync` and `rollback`, add a `--profile` flag.

- [ ] **Step 5: 验证 CLI 可运行**

Run:

```bash
go run ./cmd/skillctl --help
```

Expected: help output contains `status`, `update`, `diff`, `patch`, `doctor`, `sync`, `rollback`.

## Task 2: 配置和 lock 文件模型

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `examples/skillctl.yaml.example`

- [ ] **Step 1: 实现配置结构**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources  map[string]Source  `yaml:"sources"`
	Profiles map[string]Profile `yaml:"profiles"`
}

type Source struct {
	Repo   string           `yaml:"repo"`
	Ref    string           `yaml:"ref"`
	Skills map[string]Skill `yaml:"skills"`
}

type Skill struct {
	Path string `yaml:"path"`
}

type Profile struct {
	Target string   `yaml:"target"`
	Skills []string `yaml:"skills"`
}

type Lock struct {
	Skills map[string]LockedSkill `yaml:"skills"`
}

type LockedSkill struct {
	Source         string `yaml:"source"`
	UpstreamCommit string `yaml:"upstream_commit"`
	UpstreamPath   string `yaml:"upstream_path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("sources must not be empty")
	}
	for name, source := range c.Sources {
		if source.Repo == "" {
			return fmt.Errorf("source %q repo must not be empty", name)
		}
		if source.Ref == "" {
			return fmt.Errorf("source %q ref must not be empty", name)
		}
		if len(source.Skills) == 0 {
			return fmt.Errorf("source %q skills must not be empty", name)
		}
		for skillName, skill := range source.Skills {
			if skill.Path == "" {
				return fmt.Errorf("skill %q path must not be empty", skillName)
			}
		}
	}
	for name, profile := range c.Profiles {
		if profile.Target == "" {
			return fmt.Errorf("profile %q target must not be empty", name)
		}
		if len(profile.Skills) == 0 {
			return fmt.Errorf("profile %q skills must not be empty", name)
		}
	}
	return nil
}
```

- [ ] **Step 2: 编写配置测试**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skillctl.yaml")
	data := []byte(`
sources:
  superpowers:
    repo: https://github.com/example/skills.git
    ref: main
    skills:
      brainstorming:
        path: skills/brainstorming
profiles:
  trae-workspace:
    target: /cloudide/workspace/.trae/skills
    skills:
      - brainstorming
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Sources["superpowers"].Skills["brainstorming"].Path != "skills/brainstorming" {
		t.Fatalf("unexpected skill path")
	}
}
```

- [ ] **Step 3: 写入配置样例**

Update `examples/skillctl.yaml.example`:

```yaml
sources:
  superpowers:
    repo: https://github.com/example/skills.git
    ref: main
    skills:
      brainstorming:
        path: skills/brainstorming
      writing-plans:
        path: skills/writing-plans

profiles:
  trae-workspace:
    target: /cloudide/workspace/.trae/skills
    skills:
      - brainstorming
      - writing-plans
```

- [ ] **Step 4: 验证**

Run:

```bash
go test ./internal/config
```

Expected:

```text
ok  	github.com/your-org/skills-manager/internal/config
```

## Task 3: workspace 路径约定

**Files:**
- Create: `internal/workspace/layout.go`
- Create: `internal/workspace/layout_test.go`

- [ ] **Step 1: 实现布局模型**

Create `internal/workspace/layout.go`:

```go
package workspace

import "path/filepath"

type Layout struct {
	Root string
}

func New(root string) Layout {
	return Layout{Root: root}
}

func (l Layout) ConfigPath() string {
	return filepath.Join(l.Root, "skillctl.yaml")
}

func (l Layout) LockPath() string {
	return filepath.Join(l.Root, "skillctl.lock")
}

func (l Layout) SourceDir(source string) string {
	return filepath.Join(l.Root, "sources", source)
}

func (l Layout) PatchFile(skill string) string {
	return filepath.Join(l.Root, "patches", skill, "local.patch")
}

func (l Layout) DistSkillDir(skill string) string {
	return filepath.Join(l.Root, "dist", skill)
}

func (l Layout) BackupDir(profile string) string {
	return filepath.Join(l.Root, "backups", profile)
}
```

- [ ] **Step 2: 编写测试**

Create `internal/workspace/layout_test.go`:

```go
package workspace

import "testing"

func TestLayoutPaths(t *testing.T) {
	layout := New("/repo")
	cases := map[string]string{
		"config": layout.ConfigPath(),
		"lock":   layout.LockPath(),
		"source": layout.SourceDir("superpowers"),
		"patch":  layout.PatchFile("brainstorming"),
		"dist":   layout.DistSkillDir("brainstorming"),
		"backup": layout.BackupDir("trae-workspace"),
	}

	want := map[string]string{
		"config": "/repo/skillctl.yaml",
		"lock":   "/repo/skillctl.lock",
		"source": "/repo/sources/superpowers",
		"patch":  "/repo/patches/brainstorming/local.patch",
		"dist":   "/repo/dist/brainstorming",
		"backup": "/repo/backups/trae-workspace",
	}

	for key, got := range cases {
		if got != want[key] {
			t.Fatalf("%s got %q want %q", key, got, want[key])
		}
	}
}
```

- [ ] **Step 3: 验证**

Run:

```bash
go test ./internal/workspace
```

Expected: tests pass.

## Task 4: git source 操作

**Files:**
- Create: `internal/gitx/git.go`
- Create: `internal/gitx/git_test.go`

- [ ] **Step 1: 实现 git 命令封装**

Create `internal/gitx/git.go`:

```go
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
```

- [ ] **Step 2: 编写可跳过的集成测试**

Create `internal/gitx/git_test.go`:

```go
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
```

- [ ] **Step 3: 验证**

Run:

```bash
go test ./internal/gitx
```

Expected: tests pass or skip only when git is unavailable.

## Task 5: patch 应用和刷新

**Files:**
- Create: `internal/patch/patch.go`
- Create: `internal/patch/patch_test.go`

- [ ] **Step 1: 实现 patch 接口**

Create `internal/patch/patch.go`:

```go
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
```

- [ ] **Step 2: 编写最小测试**

Create `internal/patch/patch_test.go`:

```go
package patch

import (
	"context"
	"testing"
)

type fakeGit struct {
	out string
	err error
}

func (f fakeGit) Run(ctx context.Context, dir string, args ...string) (string, error) {
	return f.out, f.err
}

func TestRefreshRequiresChanges(t *testing.T) {
	service := Service{Git: fakeGit{}}
	_, err := service.Refresh(context.Background(), t.TempDir(), "local.patch")
	if err == nil {
		t.Fatalf("expected error when git repo has no changes")
	}
}
```

- [ ] **Step 3: 验证**

Run:

```bash
go test ./internal/patch
```

Expected: tests pass.

## Task 6: doctor 校验

**Files:**
- Create: `internal/doctor/doctor.go`
- Create: `internal/doctor/doctor_test.go`

- [ ] **Step 1: 实现校验器**

Create `internal/doctor/doctor.go`:

```go
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Issue struct {
	Path    string
	Message string
}

func CheckSkillDir(skillDir string) []Issue {
	var issues []Issue
	skillFile := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		issues = append(issues, Issue{Path: skillFile, Message: "SKILL.md is missing"})
		return issues
	}

	_ = filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, Issue{Path: path, Message: readErr.Error()})
			return nil
		}
		text := string(data)
		if strings.Contains(text, "<<<<<<<") || strings.Contains(text, "=======") || strings.Contains(text, ">>>>>>>") {
			issues = append(issues, Issue{Path: path, Message: "git conflict marker found"})
		}
		return nil
	})

	return issues
}

func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return "doctor passed"
	}
	var b strings.Builder
	for _, issue := range issues {
		_, _ = fmt.Fprintf(&b, "%s: %s\n", issue.Path, issue.Message)
	}
	return b.String()
}
```

- [ ] **Step 2: 编写测试**

Create `internal/doctor/doctor_test.go`:

```go
package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSkillDirMissingSkillFile(t *testing.T) {
	issues := CheckSkillDir(t.TempDir())
	if len(issues) != 1 {
		t.Fatalf("got %d issues want 1", len(issues))
	}
}

func TestCheckSkillDirConflictMarker(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("<<<<<<< HEAD\n"), 0644); err != nil {
		t.Fatal(err)
	}
	issues := CheckSkillDir(dir)
	if len(issues) != 1 || issues[0].Message != "git conflict marker found" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}
```

- [ ] **Step 3: 验证**

Run:

```bash
go test ./internal/doctor
```

Expected: tests pass.

## Task 7: sync 和 rollback

**Files:**
- Create: `internal/syncer/syncer.go`
- Create: `internal/syncer/syncer_test.go`

- [ ] **Step 1: 实现目录复制和备份**

Create `internal/syncer/syncer.go`:

```go
package syncer

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

type Options struct {
	DryRun bool
}

type Plan struct {
	Files []string
}

func CopyDir(src, dst string) error {
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

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func Timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
```

- [ ] **Step 2: 编写测试**

Create `internal/syncer/syncer_test.go`:

```go
package syncer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nested", "SKILL.md"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDir(src, filepath.Join(dst, "skill")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "skill", "nested", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content")
	}
}
```

- [ ] **Step 3: 验证**

Run:

```bash
go test ./internal/syncer
```

Expected: tests pass.

## Task 8: 接通命令行为

**Files:**
- Modify: `internal/commands/status.go`
- Modify: `internal/commands/doctor.go`
- Modify: `internal/commands/sync.go`
- Modify: `internal/commands/rollback.go`
- Modify: `internal/commands/update.go`
- Modify: `internal/commands/diff.go`
- Modify: `internal/commands/patch.go`

- [ ] **Step 1: 增加 root 参数**

Add persistent flag in `internal/commands/root.go`:

```go
var rootPath string

cmd.PersistentFlags().StringVar(&rootPath, "root", ".", "skills-manager repository root")
```

- [ ] **Step 2: 实现 doctor 命令**

Use `workspace.New(rootPath)` and `doctor.CheckSkillDir(layout.DistSkillDir(skill))` for every skill declared in config. Exit with error if issues exist.

- [ ] **Step 3: 实现 sync 命令**

Load config, find selected profile, run doctor for each selected skill, copy `dist/<skill>` to `<profile.target>/<skill>`, and create backup before overwrite.

- [ ] **Step 4: 实现 rollback 命令**

Find the latest timestamped backup under `backups/<profile>` and copy it back to the profile target.

- [ ] **Step 5: 实现 status 命令**

Print source names, skill names, profile names, and whether each `dist/<skill>/SKILL.md` exists.

- [ ] **Step 6: 实现 update/diff/patch refresh**

Wire commands to git and patch services:

- `update` clones/fetches source, checks out upstream path into dist, applies patch if present.
- `diff` prints git diff between dist and source snapshot.
- `patch refresh` writes `git diff --binary` output to `patches/<skill>/local.patch`.

- [ ] **Step 7: 端到端验证**

Run:

```bash
go test ./...
go run ./cmd/skillctl --root . status
go run ./cmd/skillctl --root . doctor
```

Expected: unit tests pass; status and doctor return useful output without panic.

## Task 9: 文档和发布准备

**Files:**
- Modify: `README.md`
- Modify: `examples/skillctl.yaml.example`
- Create: `.gitignore`

- [ ] **Step 1: 添加 .gitignore**

Create `.gitignore`:

```gitignore
/sources/
/dist/
/backups/
/skillctl.lock.tmp
```

- [ ] **Step 2: 补充 README 安装说明**

Add:

````markdown
## Build

```bash
go build -o bin/skillctl ./cmd/skillctl
```

## Local Check

```bash
go test ./...
bin/skillctl --root . status
```
````

- [ ] **Step 3: 提交**

Run:

```bash
git add .
git commit -m "docs: add skills manager design and implementation plan"
```

Expected: commit succeeds.

## 自检清单

- [ ] spec 中的 source、patch、dist、profile、doctor、sync、rollback 均有对应任务。
- [ ] README 描述了目标、结构、核心命令和推荐工作流。
- [ ] plan 没有占位实现或含糊步骤。
- [ ] 第一版不引入服务端、UI 或 marketplace。
- [ ] sync 在 doctor 失败时不会写入 agent 生效目录。
- [ ] rollback 只影响目标安装目录，不修改 source、patch、dist 和 lock。
