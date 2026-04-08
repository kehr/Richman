# LLM 降级契约与用户自选 Provider 实施计划（总述）

本 plan 将 `docs/prds/llm-degraded-contract-prd.md` 和 `docs/trds/llm-degraded-contract-trd.md` 的设计落地为可执行的编码任务，拆分成 5 个 phase，每个 phase 一个独立 step 文件，放在同名目录 `llm-degraded-contract-plan/` 下。

## Phase 依赖关系

```
Phase 1: Backend Foundation        (DB + models + crypto + ssrf)
                |
                v
Phase 2: Backend Core              (repo + resolver)
                |
                v
Phase 3: Backend Integration       (synthesizer + service + handlers + wire-up)
                |
                v
Phase 4: Frontend                  (types + settings + dashboard + card pill)
                |
                v
Phase 5: Verification              (make check + e2e smoke + execution report)
```

Phase 1-3 必须严格顺序。Phase 4 可在 Phase 3 的 API handler 完成后开始（handler 稳定后前端可实现）。Phase 5 在 Phase 3 + Phase 4 都完成后才能执行。

## Step 文件清单

| Phase | 文件 | 主题 |
|---|---|---|
| 1 | `step1-backend-foundation.md` | DB 迁移 + model + crypto + SSRF |
| 2 | `step2-backend-core.md` | LLMConfigRepo + Resolver |
| 3 | `step3-backend-integration.md` | Synthesizer + service + handlers + wire-up |
| 4 | `step4-frontend.md` | API types + settings + dashboard + pill + onboarding |
| 5 | `step5-verification.md` | make check + 前端 lint + e2e smoke + 执行报告 |

## 全局执行规则

### 分支与 worktree

本 plan 在独立 worktree `feat/llm-degraded-contract` 执行，该 worktree 从 `main` 拉出。Phase 完成后在同一 worktree 分主题 commit（严格按 `docs/standards/commit-hygiene.md` 的规则）。全部完成后由主会话统一合并到 main。

### 提交粒度

每个 phase 至少 1 个 commit，复杂 phase 允许拆多个 commit。禁止 `git add -A`，所有 stage 必须显式 file path。每个 commit message 严格遵循 `type(scope): subject` 格式。

### lint 纪律

每次修改 backend 文件后必须在 `backend/` 目录下运行 `make check`；前端同理跑 `pnpm lint:all`。零 warning 才能进入下一个 step。

### 零 AI 痕迹

所有代码、注释、commit message、分支名、文档都不得出现 AI/Claude/Anthropic/OpenAI 工具相关字样（Claude API / OpenAI API 作为 **产品名** 是被允许的）。

### 失败处理

每个 step 的验收条件写在该 step 文件末尾。验收不通过时：
- 属于本 step scope 的 bug 立即修
- 跨 step 的设计问题在执行报告里记录，由主会话决定是否回到 PRD/TRD 调整
- 无法决策的项保留到 Phase 5 统一呈现

## 交叉引用

每个 step 的"设计依据"段必须同时引用：
- PRD 章节（做什么、为什么）
- TRD 章节（怎么做、具体签名）

禁止在 step 文件里写代码实现，实现细节在 TRD 里；step 只负责任务编排、验收标准、依赖说明。

## 验收标准（总）

- 所有 5 个 phase 完成后：
  - backend `make check` 全绿
  - frontend `pnpm lint:all` + `pnpm test` 全绿
  - 本地手动烟雾测试：创建用户 → 配置 LLM → 触发分析 → 看到 AI 角标 → 删除 LLM → 再分析 → 看到 Rules 角标 → 重新配置 → 点击 banner "重新分析所有持仓" → 历史卡片变回 AI
- 执行报告 `docs/reports/llm-degraded-contract-execution-report.md` 完整更新
- 所有决策记录与 PRD 的 "决策记录" 段落对齐
- 未决策的观察项在执行报告里明示

## 执行报告位置

`docs/reports/llm-degraded-contract-execution-report.md`

执行 agent 必须在每个 step 完成后追加章节，记录：commit SHA、关键决策、偏差说明、验收结果、已记录但未修的观察项。
