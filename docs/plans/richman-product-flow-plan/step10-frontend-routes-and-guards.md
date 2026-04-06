# Step 10 前端路由与守卫重构

## 任务目标

把前端路由和左侧菜单从 6 项精简为 3 项 + 帮助底部入口，新增 onboarding 路由分支和 OnboardingGuard，删除取消的页面文件。

## 涉及文件

修改：
- `frontend/src/routes.tsx`（重构 routes 树，删除 AnalysisPage / DecisionCardListPage / NotificationsPage 的导入和路由）
- `frontend/src/layouts/MainLayout.tsx`（菜单从 6 项改为 3 项，添加底部帮助入口）
- `frontend/src/domain/auth/auth-guard.tsx`（追加 OnboardingGuard 链）

创建：
- `frontend/src/domain/auth/onboarding-guard.tsx`

删除：
- `frontend/src/pages/analysis/AnalysisPage.tsx`
- `frontend/src/pages/decision-cards/DecisionCardListPage.tsx`（保留 DecisionCardDetailPage）
- `frontend/src/pages/notifications/NotificationsPage.tsx`（内容迁到 Settings tab，由 step18 负责）
- `frontend/src/features/analysis/`（整个 feature 文件夹）
- `frontend/src/features/notification/`（整个 feature 文件夹，注意只删与渠道配置相关的，与"通知历史"无关的代码留给 step18）

## 设计依据

- PRD §1.2 §1.3 信息架构与 sitemap
- PRD §9 菜单最终形态
- TRD §6.1 OnboardingGuard 行为
- TRD §7.1 §7.2 路由重构与 MainLayout 改造

## 实施要点

- 新路由清单：
  - 公开：/login /register
  - onboarding（OnboardingGuard 内）：/onboarding/welcome /onboarding/categories /onboarding/first-holding /onboarding/first-analysis
  - 主应用：/dashboard /portfolio /portfolio/:id/transactions /decision-cards/:id /settings /help
  - 兜底：* → /dashboard
- OnboardingGuard 调 useOnboardingStatus hook（step11 前置或本 step 内顺手新建一个临时 hook 占位，step11 替换为正式实现）
- 守卫逻辑：
  - 已登录 + onboarding 未完成 + 当前路径不在 /onboarding/* → 跳 /onboarding/welcome
  - 已登录 + onboarding 已完成 + 当前路径在 /onboarding/* → 跳 /dashboard
- MainLayout menuRoutes 数组只保留 Dashboard / Portfolio / Settings 三项
- 用 ProLayout menuFooterRender 渲染帮助入口（不放在 menuRoutes 里以免占顶级位）
- 所有 antd 图标继续从 @/ui-kit/eat barrel 导入
- 删除文件前确认无其他模块引用（grep 一下）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过（包括 dependency-cruiser 架构检查）
2. `pnpm dev` 启动后访问 /dashboard 不报错
3. 浏览器查看左侧菜单只有 3 项 + 底部帮助
4. 模拟未完成 onboarding 的用户访问 /dashboard 自动跳 /onboarding/welcome
5. 已完成 onboarding 的用户访问 /onboarding/welcome 自动跳 /dashboard

## 依赖说明

- 前置：step08（onboarding API 必须先存在）
- 后续步骤建立在新的路由结构上

## 预估提交

- commit 1: `refactor(routes): collapse menu to dashboard, portfolio, settings`
- commit 2: `feat(auth): add onboarding guard`
- commit 3: `chore(pages): remove deprecated analysis, decision-cards-list, notifications pages`
