# Commit Scope 纪律

本规范定义 Richman 项目所有 git 提交必须遵守的 scope 约束。目的是让 commit history 成为可回溯的工程档案，而不是无主题的文件堆。

## 背景

2026-04-09 调试 `analysis/synthesis` 的 nil-panic 时发现，commit `0ad2ce3 docs(trd): add onboarding UX overhaul technical design` 的 commit message 写的是「新增一份 TRD 文档」，但实际 stat 包含：

- `backend/cmd/server/main.go` +12 行
- `backend/internal/analysis/synthesis/synthesizer.go` +12 行
- `backend/internal/analysis/synthesis/synthesizer_test.go` +23 行
- `docs/trds/onboarding-ux-overhaul-trd.md` +817 行
- `frontend/src/features/auth/LoginForm.tsx` +40 行
- `frontend/src/features/auth/RegisterForm.tsx` +39 行
- `frontend/src/pages/auth/components/AuthSplitLayout.tsx` +257 行

七个文件跨 backend/frontend/docs 三个独立 scope，但在 commit message 里只体现了「TRD 文档」一件事。最直接的后果是：

1. 将来用 `git log -- backend/internal/analysis/synthesis/` 追溯 nil-panic 修复时会指向一个看起来完全无关的 docs commit，误导后续维护者
2. `git bisect` 定位 bug 引入点时拿到的是一个巨型混合 commit，无法缩小范围
3. PR review 阶段 reviewer 看到「docs(trd)」的标题会按 docs 标准审，漏掉代码改动的实际风险
4. revert 变得不可能——revert 一个 docs commit 会意外撤销 backend 修复和前端重构

根因是使用了 `git add -A` / `git commit -a` 这类宽 scope 暂存命令，没有按主题 stage 文件。这个反模式在 `~/.claude/CLAUDE.md` 里已经有明确警告，但项目层面没有机制拦住它。本规范把反模式的防御固化为可执行的规则。

## 核心原则

1. **一个 commit 一个主题**：每个 commit 只应该做一件逻辑上连贯的事情
2. **commit message 必须和实际文件列表对应**：message 描述的范围必须完全覆盖实际修改的所有文件，不能是其中一部分
3. **按文件名 stage，不要用通配**：`git add <file1> <file2>` 而不是 `git add -A` / `git add .` / `git commit -a`
4. **混合 scope 必须拆**：如果改动涉及多个主题，必须拆成多个 commit；如果主题之间有依赖，安排先后顺序

## Commit Message 规范

### 格式

```
<type>(<scope>): <subject>

<body，可选>

<footer，可选>
```

### type 允许的值

| type | 含义 |
|------|------|
| `feat` | 新增功能（产品层可见的新能力） |
| `fix` | 修 bug |
| `refactor` | 重构，不改变外部行为 |
| `perf` | 性能优化 |
| `test` | 只动测试 |
| `docs` | 只动文档（`docs/` 目录、README、代码注释） |
| `style` | 纯样式调整（前端 CSS、格式化） |
| `chore` | 工具链、配置、依赖升级、lint 清理等维护类 |
| `build` | 构建系统或外部依赖的变化 |
| `ci` | CI 配置 |

### scope 约束

- scope 必须具体：`fix(analysis/synthesis)` 而不是 `fix(backend)`
- 跨多个 scope 时必须拆 commit，不允许写 `fix(backend,frontend)` 或 `fix(*)`
- 纯 docs commit 的 scope 填文档类别：`docs(prd)`、`docs(trd)`、`docs(standards)`

### subject 约束

- 祈使句，英文，首字母小写，句末无句号
- 不超过 70 个字符
- 不允许 emoji
- 不允许 AI 相关字样（遵循零 AI 痕迹原则）

## Stage 纪律

### 禁止的命令

```bash
git add -A               # 禁止
git add .                # 禁止
git commit -a            # 禁止
git commit --all         # 禁止
```

这些命令会把工作树里所有变动不分 scope 地扫进暂存区，包括别的会话在并行修改的文件、未完成的实验代码、临时产物等。

### 允许的命令

```bash
git add backend/internal/analysis/synthesis/synthesizer.go
git add backend/internal/analysis/synthesis/synthesizer_test.go
git status                          # 确认 stage 状态符合预期
git diff --cached                   # review 准备提交的内容
git commit -m "fix(analysis/synthesis): guard Synthesize against nil provider"
```

### 交互式 stage

对于同一个文件里有多个主题的变更，用 `git add -p` 分块暂存：

```bash
git add -p path/to/file.go         # 按 hunk 选择
```

禁止为了方便就把整个文件一起 commit 进「主 commit」，剩下的无关 hunk 回头再处理。

## Commit Message 与文件列表对齐检查

在每次 `git commit` 前必须做一次对齐检查：

1. `git status` 确认暂存的文件列表
2. 对每个暂存文件自问：**这个文件的改动是否在 commit message 的主题范围内？**
3. 如果有任何一个文件不属于当前主题，从暂存区撤回（`git restore --staged <file>`），等下一个 commit 处理

## 违规案例与正确姿势

### 反模式：一次性大扫除

```bash
git add -A
git commit -m "docs(trd): add onboarding UX overhaul technical design"
```

实际 stat：

```
 backend/cmd/server/main.go                         |  12 +-
 backend/internal/analysis/synthesis/synthesizer.go |  12 +-
 docs/trds/onboarding-ux-overhaul-trd.md            | 817 +++++++++++++++++++++
 frontend/src/pages/auth/components/AuthSplitLayout.tsx | 257 ++++++-
```

问题：message 只描述 TRD 文档，但实际 commit 包含 backend 修复和前端重构。

### 正确姿势：按主题拆 commit

```bash
# Commit 1: TRD 文档，只暂存 docs
git add docs/trds/onboarding-ux-overhaul-trd.md
git commit -m "docs(trd): add onboarding UX overhaul technical design"

# Commit 2: backend 修复
git add backend/internal/analysis/synthesis/synthesizer.go
git add backend/internal/analysis/synthesis/synthesizer_test.go
git add backend/cmd/server/main.go
git commit -m "fix(analysis/synthesis): guard Synthesize against nil provider"

# Commit 3: 前端布局重构
git add frontend/src/pages/auth/components/AuthSplitLayout.tsx
git add frontend/src/features/auth/LoginForm.tsx
git add frontend/src/features/auth/RegisterForm.tsx
git commit -m "refactor(auth): restructure login/register form layout"
```

三个 commit 各自可独立 review、可独立 revert、可独立 cherry-pick。

## 修复已经混合的 commit

如果发现已经提交的 commit 包含多个主题（且尚未 push）：

1. `git reset --mixed HEAD^` 把最近一个 commit 的文件退回暂存区
2. 按上面的「正确姿势」重新分批 commit
3. **严格禁止** 对已经推到 main 的 commit 执行这个操作；如果已 push，只能用「后续 commit 说明混合情况」的方式补救，不允许 force push 重写 history

## 并行会话场景的特殊约束

如果你在多个会话/终端同时修改同一个 worktree，并行会话的文件变动会互相可见但不应互相交叉 commit：

1. 使用 `git worktree add` 创建独立工作树隔离并行会话
2. 禁止在一个会话里 `git add -A`，因为它会扫到另一个会话正在编辑的文件
3. 即使确信「另一个会话的改动和我的无关」，也必须显式按文件名 stage，拒绝任何形式的通配

## CI 层拦截

CI 必须配置提交消息格式校验（commitlint 或等价工具）：

1. 拦截不符合 `<type>(<scope>): <subject>` 格式的 commit message
2. 拦截不在允许列表里的 type
3. 拦截超过 70 字符的 subject
4. 拦截 commit message 里出现的 AI 相关关键词（零 AI 痕迹原则）

## 例外情况

以下情况允许跳过本规范：

- `merge` commit：message 由 git 自动生成
- 初次仓库导入：`chore: initial commit`
- 合并到归档分支前的 squash commit：允许总结多个历史 commit，但必须在 body 里列明合并了哪些 commit

其它任何情况都必须严格遵守。
