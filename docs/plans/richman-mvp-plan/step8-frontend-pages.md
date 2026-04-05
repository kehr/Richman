# Step 8: Frontend Shell + All Pages

## 任务目标

实现完整的前端应用：domain 层基础设施（HTTP client、auth、i18n）、layouts、config、所有 features 模块、所有 pages。包含亮暗主题、中英文国际化、风险声明。

## 涉及文件路径

### 创建

**Domain 层：**
- `frontend/src/domain/http/client.ts` -- API client（request 函数、拦截器、错误处理）
- `frontend/src/domain/http/types.ts` -- 通用响应类型
- `frontend/src/domain/auth/storage.ts` -- Token 存储（localStorage）
- `frontend/src/domain/auth/use-current-user.ts` -- 当前用户 hook
- `frontend/src/domain/auth/auth-guard.tsx` -- 路由守卫组件
- `frontend/src/domain/i18n/config.ts` -- next-intl 配置
- `frontend/src/domain/i18n/zh.json` -- 中文翻译
- `frontend/src/domain/i18n/en.json` -- 英文翻译
- `frontend/src/domain/ui/format.ts` -- 数据格式化（金额、百分比、日期）
- `frontend/src/domain/ui/use-theme.ts` -- 主题切换 hook

**Config 层：**
- `frontend/src/config/routes.tsx` -- 路由配置
- `frontend/src/config/theme.ts` -- Ant Design ThemeConfig（亮色/暗色）
- `frontend/src/config/query-client.ts` -- TanStack Query 全局配置

**Layouts：**
- `frontend/src/layouts/MainLayout.tsx` -- ProLayout 侧边栏布局

**Features：**
- `frontend/src/features/auth/api.ts`
- `frontend/src/features/auth/useAuth.ts`
- `frontend/src/features/auth/LoginForm.tsx`
- `frontend/src/features/auth/RegisterForm.tsx`
- `frontend/src/features/auth/index.ts`
- `frontend/src/features/dashboard/api.ts`
- `frontend/src/features/dashboard/useStats.ts`
- `frontend/src/features/dashboard/StatsOverview.tsx`
- `frontend/src/features/dashboard/index.ts`
- `frontend/src/features/portfolio/api.ts`
- `frontend/src/features/portfolio/usePortfolio.ts`
- `frontend/src/features/portfolio/HoldingForm.tsx`
- `frontend/src/features/portfolio/TradeRecordList.tsx`
- `frontend/src/features/portfolio/index.ts`
- `frontend/src/features/asset-catalog/api.ts`
- `frontend/src/features/asset-catalog/useAssetCatalog.ts`
- `frontend/src/features/asset-catalog/AssetPicker.tsx`
- `frontend/src/features/asset-catalog/index.ts`
- `frontend/src/features/decision-card/api.ts`
- `frontend/src/features/decision-card/useDecisionCard.ts`
- `frontend/src/features/decision-card/DecisionCardView.tsx`
- `frontend/src/features/decision-card/ThreeDimensionChart.tsx`
- `frontend/src/features/decision-card/ConfidenceBadge.tsx`
- `frontend/src/features/decision-card/index.ts`
- `frontend/src/features/analysis/api.ts`
- `frontend/src/features/analysis/useAnalysis.ts`
- `frontend/src/features/analysis/AnalysisProgress.tsx`
- `frontend/src/features/analysis/index.ts`
- `frontend/src/features/notification/api.ts`
- `frontend/src/features/notification/useNotification.ts`
- `frontend/src/features/notification/ChannelConfigForm.tsx`
- `frontend/src/features/notification/index.ts`

**Pages（App Router）：**
- `frontend/src/app/(auth)/login/page.tsx`
- `frontend/src/app/(auth)/register/page.tsx`
- `frontend/src/app/(main)/layout.tsx`
- `frontend/src/app/(main)/dashboard/page.tsx`
- `frontend/src/app/(main)/portfolio/page.tsx`
- `frontend/src/app/(main)/portfolio/[id]/page.tsx`
- `frontend/src/app/(main)/portfolio/new/page.tsx`
- `frontend/src/app/(main)/analysis/page.tsx`
- `frontend/src/app/(main)/decision-cards/page.tsx`
- `frontend/src/app/(main)/decision-cards/[id]/page.tsx`
- `frontend/src/app/(main)/notifications/page.tsx`
- `frontend/src/app/(main)/settings/page.tsx`

### 修改

- `frontend/src/app/layout.tsx` -- 接入 Providers（QueryClient、ConfigProvider、i18n）
- `frontend/src/app/page.tsx` -- Root redirect to /dashboard

## PRD/TRD 章节引用

- PRD 3.1 持仓管理（快速模式 + 明细模式、分类浏览搜索）
- PRD 3.3 决策卡（内容、两层结构、简洁/详细切换）
- PRD 3.4 推送通知设置
- PRD 4.1 账户系统（邮箱注册 + 密码登录 + 邀请码）
- PRD 5.2 前端技术栈
- PRD 5.2.1 前端架构
- PRD 1.6 国际化（中文 + 英文）
- `docs/standards/frontend.md` 完整前端规范

## 验证标准

- [ ] `pnpm dev` 启动成功
- [ ] `pnpm lint` 通过（Biome，含 noRestrictedImports 规则）
- [ ] `pnpm type-check` 通过
- [ ] dependency-cruiser 架构检查通过（层间依赖无违规）
- [ ] 注册页面：输入邮箱 + 密码 + 邀请码可注册
- [ ] 登录页面：输入邮箱 + 密码可登录
- [ ] 未登录访问主页面被重定向到登录页
- [ ] Dashboard：显示持仓概览统计
- [ ] 持仓列表：显示当前持仓，支持新增/编辑/删除
- [ ] 新增持仓：AssetPicker 分类浏览 + 搜索选择标的
- [ ] 新增持仓：快速模式填写成本和仓位
- [ ] 交易记录：明细模式添加买卖记录
- [ ] 决策卡列表：显示最新分析结果
- [ ] 决策卡详情：三维摘要、信心度、操作建议（两层展开）、风险提示
- [ ] 决策卡：简洁/详细模式切换
- [ ] 分析页面：手动触发分析 + 进度显示
- [ ] 推送设置：添加/编辑/删除推送渠道
- [ ] 主题切换：亮色/暗色正常切换
- [ ] 语言切换：中文/英文切换，所有文案正确
- [ ] 风险声明在页面底部显著展示
- [ ] 所有 UI 组件通过 ui-kit/eat 导入（无直接 antd 导入）

## 依赖说明

- Step 3 完成（持仓 API 就绪）
- Step 6 完成（决策卡 API 就绪）
- Step 7 完成（推送 API 就绪）
- Step 1 完成（前端项目骨架就绪）
