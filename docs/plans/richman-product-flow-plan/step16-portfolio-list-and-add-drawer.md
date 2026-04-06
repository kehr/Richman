# Step 16 Portfolio 列表与 Add Holding Drawer

## 任务目标

改造 PortfolioListPage 列表，新增右侧滑入的 Add Holding Drawer（两步流程，三 tab 录入）。是 PRD §4.1 §4.2 的实现。

## 涉及文件

修改：
- `frontend/src/pages/portfolio/PortfolioListPage.tsx`

创建：
- `frontend/src/features/portfolio/components/AddHoldingDrawer.tsx`
- `frontend/src/features/portfolio/components/AddHoldingDrawer.test.tsx`
- `frontend/src/features/portfolio/components/AssetPicker.tsx`（Step 1 选标的子组件）
- `frontend/src/features/portfolio/components/AssetPicker.test.tsx`
- `frontend/src/features/portfolio/components/QuickModeForm.tsx`
- `frontend/src/features/portfolio/components/DetailModeForm.tsx`
- `frontend/src/features/portfolio/components/ScreenshotModeForm.tsx`（包裹 step17 的截图组件）
- `frontend/src/features/portfolio/components/HoldingTable.tsx`（如果当前在 page 内联，抽出来）
- `frontend/src/features/portfolio/components/TotalCapitalRow.tsx`

修改：
- `frontend/src/features/portfolio/api.ts`（如已存在，确保支持新增 / 更新 / 删除 holding 接口）
- `frontend/src/features/portfolio/use-holdings.ts`

## 设计依据

- PRD §4.1 列表页结构与字段
- PRD §4.2 Add Holding Drawer 两步流程
- PRD §8.3 总资金 Portfolio 入口
- TRD §7.3 features/portfolio 改造

## 实施要点

- PortfolioListPage：
  - 顶部标题行：左侧"我的持仓"+ 副标题"3 / 5 个 · MVP 每用户最多 5 个标的"；右侧 2 按钮（截图导入、添加持仓）
  - 第二行 TotalCapitalRow：根据 useUserSettings 显示总资金或"设置以查看"链接
  - HoldingTable：列含 标的 / 类型 / 成本 / 现价 / 仓位（百分比 + 金额）/ 浮盈亏（百分比 + 金额）/ 操作
  - 行点击（非操作列）跳到该标的最近的决策卡详情页
  - 上限 5：达到时主按钮置灰 + tooltip
- AddHoldingDrawer：
  - antd Drawer，宽度 520-720px（响应式）
  - 顶部步骤指示 "✓ 选择标的 — ② 填写信息"，可点击回退
  - 内部 useState 管理 currentStep / selectedAsset / formData
  - Step 1 渲染 AssetPicker（按类型 tab + 搜索 + 网格列表）
  - Step 2 渲染 3 个 tab：QuickModeForm / DetailModeForm / ScreenshotModeForm
  - 三个 tab 共享同一 holding 表单 state；切换 tab 不丢已填字段（统一存到 form ref）
  - 取消 / 保存按钮，保存调 useCreateHolding mutation 后关闭并 invalidate holdings query
- QuickModeForm：均价成本 + 仓位比例两个字段
- DetailModeForm：可增减行的交易记录表，自动计算综合成本与仓位
- ScreenshotModeForm：在 Drawer 内嵌入截图上传 + 识别结果（识别后跳到 QuickModeForm 预填）。本 step 实现"调用 import-screenshot API 后把第一个识别结果填进 QuickModeForm"的简化路径；完整双栏校对 modal 留给 step17
- 所有 antd 组件经 ui-kit/eat barrel
- AssetPicker 的标的库通过 GET /api/v1/asset-catalog?type=xxx 拉，按 onboarding 选择的 categories 优先排序

## 验证标准

1. `pnpm test src/features/portfolio` 通过
2. 浏览器手动测：
   - 点"+ 添加持仓"打开 Drawer
   - Step 1 切换 tab 搜索选标的
   - Step 2 三个 tab 切换不丢数据
   - 快速模式保存后看到列表新增一行
   - 5 个上限触发主按钮置灰
3. `pnpm lint:all` 通过

## 依赖说明

- 前置：step10 路由、step11 useMoney、step09 后端 DTO 含 category
- step17 会接续完成 ScreenshotModeForm 的完整双栏体验

## 预估提交

- commit 1: `refactor(portfolio): extract holding table and total capital row`
- commit 2: `feat(portfolio): add holding drawer with two-step flow`
- commit 3: `feat(portfolio): add quick / detail / screenshot mode forms`
