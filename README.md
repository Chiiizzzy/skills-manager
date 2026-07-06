# Skills Manager

Skills Manager 是一个用于管理 agent skills 的本地优先 CLI。它把开源 skills 的上游更新、个人定制 patch、合并产物和 agent 生效目录拆开管理，避免直接在 `.trae/skills` 等目录里修改导致来源不可追踪、更新不可审查。

当前实现提供 `skillctl` Go CLI，覆盖配置解析、Git source 拉取、patch 刷新、doctor 校验、sync 和 rollback 的第一版本地流程。

## 目标

- 定期拉取 GitHub 上游 skills 的最新改动。
- 保留个人定制，并在上游更新时显式暴露冲突。
- 将合并后的 skills 同步到 agent 的生效目录，例如 `/cloudide/workspace/.trae/skills`。
- 支持 dry-run、rollback、doctor 校验，降低误覆盖风险。

## 仓库结构

```text
skills-manager/
  README.md
  skillctl.yaml              # 本机实际配置，后续由用户维护
  skillctl.lock              # 锁定每个 skill 的 upstream commit
  examples/
    skillctl.yaml.example    # 配置样例
  scripts/
    install-skillctl.sh      # 一次性注册 skillctl 命令
  sources/                   # 上游快照，由 CLI 生成，不手工改
  patches/                   # 本地定制 patch，按 skill 管理
  dist/                      # 合并后的可安装产物，由 CLI 生成
  backups/                   # sync 前备份，由 CLI 生成
  bin/                       # 本地构建产物，由用户生成
  docs/
    specs/
      2026-07-01-skills-manager-design.md
    plans/
      2026-07-01-skills-manager-implementation-plan.md
```

## Install Command

第一次使用时执行一次安装脚本，将 `skillctl` 注册为 shell 命令：

```bash
scripts/install-skillctl.sh
```

脚本会构建 `bin/skillctl`，并默认创建 `~/.local/bin/skillctl` 链接。之后在任意 shell 中可直接使用 `skillctl`；如果脚本提示 `~/.local/bin` 不在 `PATH` 中，按提示把它加入 shell profile。

如需安装到其他已在 `PATH` 中的目录：

```bash
SKILLCTL_BIN_DIR=/usr/local/bin scripts/install-skillctl.sh
```

## Build

开发调试时也可以只构建本地二进制：

```bash
go build -o bin/skillctl ./cmd/skillctl
```

## Local Check

```bash
go test ./...
skillctl --root . status
```

如果当前仓库根目录还没有 `skillctl.yaml`，`status` 会报告配置文件缺失。可以先复制并编辑 `examples/skillctl.yaml.example`。

## 核心命令

```bash
skillctl --root . status
skillctl --root . update --all
skillctl --root . diff brainstorming
skillctl --root . patch refresh brainstorming
skillctl --root . doctor
skillctl --root . sync --profile trae-workspace --dry-run
skillctl --root . sync --profile trae-workspace
skillctl --root . rollback --profile trae-workspace
```

## 推荐工作流

1. 将常用开源 skills 注册到 `skillctl.yaml`。第一版 `source.ref` 只支持普通分支名，例如 `main` 或 `feature/foo`。
2. 执行 `skillctl update --all` 拉取上游更新并重新生成 `dist/`。
3. 如果出现冲突，在 `dist/<skill>` 中解决 conflict marker。
4. 执行 `skillctl patch refresh <skill>` 将已解决的差异刷新回 `patches/`。
5. 执行 `skillctl doctor` 检查 `SKILL.md`、引用文件和冲突标记。
6. 执行 `skillctl sync --profile trae-workspace --dry-run` 预览安装变更。
7. 执行 `skillctl sync --profile trae-workspace` 同步到 agent 生效目录。

## 文档

- 设计方案：[docs/specs/2026-07-01-skills-manager-design.md](docs/specs/2026-07-01-skills-manager-design.md)
- 实施计划：[docs/plans/2026-07-01-skills-manager-implementation-plan.md](docs/plans/2026-07-01-skills-manager-implementation-plan.md)
