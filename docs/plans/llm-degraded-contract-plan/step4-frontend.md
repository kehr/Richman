# Step 4: Frontend

## 任务目标

在 Step 3 的 backend API 稳定之后实现所有前端界面：LLM 配置设置区、dashboard 降级 banner、决策卡片 source pill、onboarding consent 步骤。完成后端到端闭环可见。

## 涉及文件

### 新增

- `frontend/src/features/settings-llm/`
  - `LLMSection.tsx`
  - `LLMEmptyState.tsx`
  - `LLMHealthyCard.tsx`
  - `LLMFailingCard.tsx`
  - `LLMConfigForm.tsx`
  - `LLMProbeButton.tsx`
  - `hooks.ts`
  - `types.ts`
  - `index.ts`
- `frontend/src/features/dashboard-llm-status/`
  - `LLMStatusBanner.tsx`
  - `useLLMStatusBanner.ts`
  - `index.ts`
- `frontend/src/features/decision-card/SourcePill.tsx`
- `frontend/src/features/onboarding/LLMConsentStep.tsx`（或扩展 existing onboarding step）

### 修改

- `frontend/src/api/decisionCards.ts`：DTO 类型追加 `synthesisSource` 和 `providerUsed`
- `frontend/src/api/dashboard.ts`：DashboardSummary 类型追加 `llmStatus`
- `frontend/src/api/settingsLLM.ts`（新）：GET/PUT/DELETE/probe 的 API hooks
- `frontend/src/api/analysis.ts`：追加 `useReanalyzeAll` hook
- `frontend/src/pages/settings/SettingsPage.tsx`：集成 `<LLMSection />`
- `frontend/src/pages/dashboard/DashboardPage.tsx`：集成 `<LLMStatusBanner />`
- `frontend/src/features/decision-card/DecisionCardHeader.tsx`：集成 `<SourcePill />`
- `frontend/src/pages/onboarding/OnboardingFlow.tsx`：新增 consent 步骤

## 设计依据

- PRD "UX 表面/Settings 页 LLM Section"：三种状态的布局
- PRD "UX 表面/Dashboard Banner"：触发条件、关闭行为、重新分析按钮
- PRD "UX 表面/卡片角标"：pill 样式和 tooltip
- PRD "UX 表面/Onboarding 引导"：选项 A/B 分支
- PRD "Fallback 链"：`fallbackToSystemDefaultOnFailure` 开关的显式文案
- TRD "前端架构"：feature 模块结构、hooks 命名、DTO 类型

## 验证标准

### LLMSection 组件测试

- 未配置：渲染 `LLMEmptyState`，显示"添加 LLM Provider"按钮
- 已配置 + 健康：渲染 `LLMHealthyCard`，显示 provider 品牌、模型、`..abcd` key hint、绿色 healthy 标签
- 已配置 + 失效：渲染 `LLMFailingCard`，显示红色 failing 标签 + 错误信息
- 点击"添加"：打开 `LLMConfigForm` modal
- 表单校验：
  - providerType 未选 → 不允许提交
  - openai_compatible + 未填 base_url → 不允许提交
  - openai_compatible + http base_url → 前端立即报错
  - api_key 未填（创建模式）→ 不允许提交
- probe 按钮：触发 probe mutation，成功显示绿色 toast，失败显示红色带错误详情
- save 成功后 invalidate `llm-config` 和 `dashboard-summary` 两个 query

### LLMStatusBanner 组件测试

- `needsReanalysis=false` → 不渲染
- `needsReanalysis=true` → 渲染 banner，显示 holdings 数 + 重新分析按钮
- 点击 X：sessionStorage 写入 `llm-status-banner-dismissed=1`，刷新页面前不再出现
- 点击"重新分析所有持仓"：触发 `useReanalyzeAll` mutation，成功后显示任务进度
- 重新分析完成 → banner 自动消失（依赖 dashboard-summary query 重新计算）

### SourcePill 组件测试

- `synthesisSource=llm` → 蓝色 pill 文案"AI"
- `synthesisSource=mixed` → 蓝色虚边 pill 文案"Mixed"
- `synthesisSource=template` → 灰色 pill 文案"Rules"
- `synthesisSource=unknown` → 不渲染
- hover 显示 tooltip，三种状态 tooltip 文案不同

### Onboarding Consent 组件测试

- 显示两个选项：跳过 / 我想试试 AI
- 选"跳过"：`use_system_default_when_unconfigured` 写 false
- 选"我想试试"：
  - 如果 dashboardSummary 表明系统默认可用：直接勾选并写 true
  - 如果系统默认不可用：跳转到 `/settings/llm` 并传 `from=onboarding` query
- API 调用：`POST /api/v1/onboarding/llm-consent` 带 boolean

### 集成验证

- `pnpm lint:all` 0 warnings
- `pnpm test` 所有 component test 和 unit test 通过
- `pnpm build` 成功
- 手动端到端：
  - 登录 → 进 settings → 配置 claude (用测试 key)
  - 保存成功 → 看到 healthy 状态
  - 触发分析 → 卡片上显示"AI" pill
  - 删除配置 → 再次触发分析 → 卡片显示"Rules" pill
  - Dashboard 顶部出现 banner
  - 点击重新分析 → 所有卡片回到"AI"

## 依赖

- Step 3 已完成且 backend 所有 API 可调

## 偏差处理

- 如果现有 settings page 已有 section 容器模式，复用该模式，不要新起一套
- SourcePill 的视觉设计如果与 Ant Design Badge 冲突，优先用 Badge + custom className
- hooks.ts 内的 query key 必须和 dashboard 的 `dashboard-summary` key 保持一致，否则失效不同步
- 如果前端已有 `features/decision-card/` 目录，`SourcePill.tsx` 直接放进去，不要新建 sub-feature
- onboarding 的 consent step 如果现有 onboarding flow 是线性 wizard，插入在合适位置（可能在 "风险偏好" step 之后）

## 预期产出

- 5-6 个新前端 feature 模块 + hooks + DTO 类型
- 多个分主题 commit：
  - `feat(api): add llm settings and reanalyze api hooks`
  - `feat(settings): add llm provider configuration ui`
  - `feat(dashboard): add llm status banner with reanalyze cta`
  - `feat(decision-card): add source provenance pill`
  - `feat(onboarding): add llm consent step`
