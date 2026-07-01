# Skills Manager

Skills Manager 是一个用于管理 agent skills 的本地优先工具方案。它把开源 skills 的上游更新、个人定制 patch、合并产物和 agent 生效目录拆开管理，避免直接在 `.trae/skills` 等目录里修改导致来源不可追踪、更新不可审查。

当前仓库处于设计阶段，已包含设计方案、实施计划和配置样例；CLI 代码将在后续按实施计划落地。

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
  sources/                   # 上游快照，由 CLI 生成，不手工改
  patches/                   # 本地定制 patch，按 skill 管理
  dist/                      # 合并后的可安装产物，由 CLI 生成
  backups/                   # sync 前备份，由 CLI 生成
  docs/
    specs/
      2026-07-01-skills-manager-design.md
    plans/
      2026-07-01-skills-manager-implementation-plan.md
```

## 核心命令设计

```bash
skillctl status
skillctl update --all
skillctl diff brainstorming
skillctl patch refresh brainstorming
skillctl doctor
skillctl sync --profile trae-workspace --dry-run
skillctl sync --profile trae-workspace
skillctl rollback --profile trae-workspace
```

## 推荐工作流

1. 将常用开源 skills 注册到 `skillctl.yaml`。
2. 执行 `skillctl update --all` 拉取上游更新并重新生成 `dist/`。
3. 如果出现冲突，在 `dist/<skill>` 中解决 conflict marker。
4. 执行 `skillctl patch refresh <skill>` 将已解决的差异刷新回 `patches/`。
5. 执行 `skillctl doctor` 检查 `SKILL.md`、引用文件和冲突标记。
6. 执行 `skillctl sync --profile trae-workspace --dry-run` 预览安装变更。
7. 执行 `skillctl sync --profile trae-workspace` 同步到 agent 生效目录。

## 文档

- 设计方案：[docs/specs/2026-07-01-skills-manager-design.md](docs/specs/2026-07-01-skills-manager-design.md)
- 实施计划：[docs/plans/2026-07-01-skills-manager-implementation-plan.md](docs/plans/2026-07-01-skills-manager-implementation-plan.md)
