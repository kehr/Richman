# i18n Default English 执行报告

## 基本信息

- 执行方式：superpowers:subagent-driven-development，每个 step 派发独立 implementer subagent + 两阶段 review
- 分支：`feat/i18n-default-en`（worktree `.claude/worktrees/i18n-default-en`）
- 三层文档闭环：
  - PRD：`docs/prds/i18n-default-en-prd.md` (commit 343193b)
  - TRD：`docs/trds/i18n-default-en-trd.md` (commit 343193b)
  - Plan：`docs/plans/i18n-default-en-plan.md` + 12 step 文件 (commit 343193b)

## 全局规则

- commit 粒度：每个 step 一个 commit
- lint 要求：每个 step 必须 `pnpm lint:all` 通过
- 冲突处理：worktree 基于 origin/main (85971dc)，执行期间不 rebase
- 零 AI 痕迹：commit message / 注释 / 文件内容不含 AI 相关信息

## Step 执行记录

### Step 1: i18n 基础设施
- 状态：已完成
- commit：88bc223
- 关键决策：直接静态 import 8 个 JSON 文件，不用动态加载（skeleton 文件暂空）
- 偏差说明：spec reviewer 确认 i18n/help/ 文件为 main 分支预存，非本 step 新增

### Step 2: App.tsx 集成 + 测试工具更新
- 状态：已完成
- commit：95a8d26
- 关键决策：test/utils.tsx 使用 I18nextProvider（react-i18next 正确导出），非旧的自定义 I18nProvider
- 偏差说明：OnboardingLayout 测试 pre-existing 失败（StepIndicator hardcode TOTAL_STEPS=5 与测试断言不符），与本 step 无关，不在修复范围内

### Step 3: JSON 资源文件
- 状态：已完成
- commit：b428e0d
- 关键决策：部分旧键语义升级（如 "Login"→"Sign In"），零引用旧键不迁移；count 字符串用 (s) 括号写法替代 _one/_other（MVP 简化）
- 偏差说明：旧 en/zh.json 中 29 个键无源码引用，未迁移

### Step 4: MainLayout i18n
- 状态：已完成
- commit：c44b1d8（amended）
- 关键决策：Globe Dropdown trigger 用原生 `<button>` 替代 `<Space role="button">`（Biome useSemanticElements 规则）；`languageMenu` 用 useMemo 避免每次 render 重建；Tooltip 置于 Dropdown 外层确保键盘可达
- 偏差说明：初版 code quality review 发现 3 个 Important 问题，修复后 re-review 通过

### Step 5: 格式化工具重构
- 状态：已完成
- commit：46d1908
- 关键决策：intl-cache 用 JSON.stringify 生成 key（当前调用点字面量稳定，无实际问题）；locale 映射二元化（"zh"→"zh-CN", else→"en-US"，load:languageOnly 保证当前不触发 zh-TW 场景）
- 偏差说明：TransactionTable.tsx / PortfolioTransactionsPage.tsx 通过 useMoney 消费，无需直接更新

### Step 6: auth namespace 迁移
- 状态：已完成
- commit：3ae8e32
- 关键决策：LoginPage / RegisterPage / ForgotPasswordPage 全部迁移至 auth namespace

### Step 7: app namespace -- dashboard + decision-card
- 状态：已完成
- commit：b703de3
- 关键决策：DashboardPage、DecisionCard 等迁移至 app namespace

### Step 8: app namespace -- portfolio
- 状态：已完成
- commit：f723fc1
- 关键决策：
  - HoldingTable.tsx 的 `computeAmount` 移入 `useMemo` 工厂函数内部，解决 Biome `useExhaustiveDependencies` 误报
  - Python 批量正则修复：全部 portfolio 组件遗漏 `portfolio.` 前缀（TypeScript strict 类型检查捕获）
  - `summarizeConfig` 作为组件内闭包（不传参 `t`），绕开 TFunction 类型不兼容
  - cancel/save 统一走 `t("action.cancel/save", { ns: "common" })`
- 偏差说明：初版遗漏 `portfolio.` 前缀，通过 tsc 类型报错发现后批量修复，lint 通过后提交

### Step 9: settings namespace 迁移
- 状态：已完成
- commit：67a9135
- 关键决策：
  - `中文` / `English` 语言名称保留原文（属于内容数据，不属于 UI 字符串）
  - provider label（"Claude (Anthropic)" 等）保留英文（品牌名称）
  - LLMProbeButton 的 `label` prop 默认值改为 `t("llm.probeButton.default")`，保留 prop 供调用方覆盖
  - RISK_OPTIONS 移入 `useMemo([t])` 使其随 locale 变化重新计算
  - 全量中文扫描：剩余 15 处均为合法例外（ETF 名称内容数据、`中文` 语言标签、贵州茅台示例数据）
- 偏差说明：无

### Step 10: HelpPage 迁移
- 状态：已完成
- commit：b3b8a28 + 25e50a3
- 关键决策：顺带修复 HelpPage.test.tsx（移除 vi.mock provider，改用 testI18n.changeLanguage）；test/utils.tsx 导出 testI18n 供测试直接控制语言
- 偏差说明：提前修复了 HelpPage.test.tsx（原属 Step 11 范围），Step 11 可跳过此文件

### Step 11: 测试文件迁移
- 状态：待执行

### Step 12: 清理 + 验证
- 状态：待执行

## 已修复问题

（执行过程中记录）

## 未修复观察项

（执行过程中记录）

## 无法决策项

（等待用户验收时决策）
