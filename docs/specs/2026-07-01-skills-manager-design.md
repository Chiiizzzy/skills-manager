# Skills Manager 设计方案

## 背景

日常 vibe coding 会使用 GitHub 上开源的 skills 引导 agent 做方案设计、编码、代码审查等任务。由于工作场景存在本地约束，开源 skills 往往需要做定制修改。直接修改 agent 的生效目录，例如 `/cloudide/workspace/.trae/skills`，会带来三个问题：

- 上游持续更新后，本地无法稳定追踪哪些改动来自 upstream。
- 本地定制和 upstream 新改动冲突时，缺少清晰的三方合并和审查流程。
- 更新完成后，还需要可靠同步到各 agent 的 `.skills` / `.trae/skills` / `.claude/skills` 等目录。

Skills Manager 的目标是建立一个独立仓库，统一保存上游来源、个人 patch、合并产物和安装配置，并通过轻量 CLI 完成更新、冲突处理、校验、同步和回滚。

## 设计目标

- 上游版本可追踪：每个 skill 记录来源 repo、ref、路径和当前锁定 commit。
- 本地定制可审查：个人修改以 patch 形式保存，不直接混入上游快照。
- 冲突显式暴露：上游更新与本地 patch 冲突时，保留 conflict marker 并阻止同步。
- 同步过程可回滚：写入 agent 生效目录前先备份，安装完成后记录 manifest。
- 多 agent 目录可配置：支持不同 profile 同步到不同目标目录。
- 本地优先：第一版不依赖服务端，不做账号体系，不要求 GitHub App。

## 非目标

- 不做 skill marketplace。
- 不做复杂图形界面。
- 不自动理解或改写 skill 语义。
- 不直接管理 agent 的运行时行为，只负责文件层面的 skills 安装。
- 不默认使用 symlink 安装，因为不同 agent 对 symlink 的扫描和权限处理不完全一致。

## 核心模型

### Source

Source 表示一个上游 Git 仓库。

字段：

- `repo`：GitHub 或其他 Git URL。
- `ref`：默认跟踪分支或 tag。
- `skills`：该 source 下可管理的 skill 列表，每个 skill 指向仓库内路径。

### Skill

Skill 是最小管理单元。一个 skill 对应一个目录，目录内必须包含 `SKILL.md`，并可以包含 prompt、脚本、参考资料等关联文件。

### Patch

Patch 表示用户对某个 skill 的本地定制。Patch 不修改 `sources/`，而是保存在 `patches/<skill>/` 中。第一版推荐使用 git patch 文件，后续可以扩展为 overlay 目录。

### Dist

Dist 是将上游快照和本地 patch 合并后的可安装结果。`dist/` 由 CLI 生成，不建议手工长期维护。冲突解决时允许临时编辑 `dist/<skill>`，随后通过 `skillctl patch refresh <skill>` 刷新 patch。

### Profile

Profile 描述安装目标。例如：

- `trae-workspace`：同步到 `/cloudide/workspace/.trae/skills`
- `claude-trade-crm`：同步到 `/cloudide/workspace/trade_crm_platform/.claude/skills`

每个 profile 可以选择安装的 skill 子集。

## 仓库结构

```text
skills-manager/
  README.md
  skillctl.yaml
  skillctl.lock
  examples/
    skillctl.yaml.example
  sources/
    <source-name>/
  patches/
    <skill-name>/
      local.patch
  dist/
    <skill-name>/
      SKILL.md
  backups/
    <profile-name>/
  docs/
    specs/
    plans/
```

## 配置文件

`skillctl.yaml` 描述来源和安装目标：

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

`skillctl.lock` 记录精确 upstream commit：

```yaml
skills:
  brainstorming:
    source: superpowers
    upstream_commit: 1111111111111111111111111111111111111111
    upstream_path: skills/brainstorming
  writing-plans:
    source: superpowers
    upstream_commit: 2222222222222222222222222222222222222222
    upstream_path: skills/writing-plans
```

## 更新流程

`skillctl update --all` 执行流程：

1. 拉取所有 source 的最新 ref。
2. 对每个 skill 读取 `skillctl.lock` 中的旧 upstream commit。
3. 取出旧 upstream、最新 upstream 和本地 patch。
4. 生成新的 `dist/<skill>`。
5. 如果 patch 可干净应用，更新 `dist/` 和 `skillctl.lock`。
6. 如果产生冲突，保留冲突文件，标记该 skill 为 conflicted，不更新安装目录。

## 冲突处理流程

当上游更新和本地 patch 冲突时：

1. 用户在 `dist/<skill>` 中解决 conflict marker。
2. 执行 `skillctl doctor` 确认没有 `<<<<<<<`、`=======`、`>>>>>>>`。
3. 执行 `skillctl patch refresh <skill>`，根据旧 upstream 和已解决的 `dist/<skill>` 重新生成 `patches/<skill>/local.patch`。
4. 再次执行 `skillctl update <skill>` 或 `skillctl doctor` 验证 patch 能稳定重放。

## 同步流程

`skillctl sync --profile <name>` 执行流程：

1. 校验 profile 存在，目标目录存在或可创建。
2. 对待同步 skills 执行 `doctor` 校验。
3. 生成 dry-run diff，展示新增、修改、删除文件。
4. 非 dry-run 模式下，先备份目标目录中将被覆盖的 skill。
5. 将 `dist/<skill>` 复制到目标目录。
6. 写入 `.skillctl-install.yaml`，记录 profile、安装时间、manager commit、每个 skill 的 upstream commit 和文件 hash。

## 回滚流程

`skillctl rollback --profile <name>` 将最近一次备份恢复到目标目录。回滚不会修改 `sources/`、`patches/`、`dist/` 和 lock，只影响 agent 生效目录。

## 校验规则

`skillctl doctor` 至少检查：

- 每个 dist skill 存在 `SKILL.md`。
- `SKILL.md` 引用的本地相对文件存在。
- dist 中不存在 git conflict marker。
- profile 中声明的 skill 都存在于 dist。
- sync 目标目录可写。
- `skillctl.lock` 中记录的 commit 在 source 仓库中可解析。

## 技术选型

第一版推荐使用 Go 编写 CLI：

- 单二进制，适合 CloudIDE 和本机环境。
- 标准库即可覆盖文件复制、hash、yaml 外少量依赖。
- git 操作可以先调用本机 `git` 命令，降低实现复杂度。

依赖建议：

- `github.com/spf13/cobra`：CLI 命令组织。
- `gopkg.in/yaml.v3`：配置文件读写。
- `github.com/stretchr/testify`：测试断言。

## 第一版验收标准

- 可以注册至少一个 GitHub skills source。
- 可以从 source 提取指定 skill 到 `sources/`。
- 可以生成并应用本地 patch。
- 可以执行 update 并识别冲突。
- 可以执行 doctor 阻止带冲突的 dist 被同步。
- 可以 dry-run 展示同步变更。
- 可以同步到 `/cloudide/workspace/.trae/skills`。
- 可以从最近一次备份回滚。

