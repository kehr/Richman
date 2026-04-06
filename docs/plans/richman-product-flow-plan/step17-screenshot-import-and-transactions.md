# Step 17 截图批量导入 Modal 与交易记录子页

## 任务目标

实现 PRD §4.3 截图批量导入全屏 Modal（双栏对照、低置信度高亮）和 PRD §4.4 交易记录子页两个收尾功能。

## 涉及文件

创建：
- `frontend/src/features/portfolio/components/ScreenshotImportModal.tsx`
- `frontend/src/features/portfolio/components/ScreenshotImportModal.test.tsx`
- `frontend/src/features/portfolio/components/RecognizedHoldingTable.tsx`（双栏右侧的可编辑表）
- `frontend/src/features/portfolio/components/ImagePreview.tsx`（双栏左侧的原图展示，含放大）
- `frontend/src/features/portfolio/api/screenshot.ts`（POST /api/v1/portfolio/import-screenshot）
- `frontend/src/features/portfolio/use-screenshot-import.ts`（mutation hook）
- `frontend/src/pages/portfolio/PortfolioTransactionsPage.tsx`
- `frontend/src/pages/portfolio/PortfolioTransactionsPage.test.tsx`
- `frontend/src/features/portfolio/components/TransactionTable.tsx`
- `frontend/src/features/portfolio/components/AddTransactionDrawer.tsx`

修改：
- `frontend/src/pages/portfolio/PortfolioListPage.tsx`（接入 ScreenshotImportModal 的"📷 截图批量导入"按钮）
- `frontend/src/features/portfolio/components/ScreenshotModeForm.tsx`（用 RecognizedHoldingTable 替换 step16 的简化版）
- `frontend/src/routes.tsx`（注册 /portfolio/:id/transactions 路由）

## 设计依据

- PRD §4.3 截图批量导入全屏 Modal 完整规格
- PRD §4.4 交易记录子页
- TRD §4 截图 OCR 管线（前端只消费 API，不重做识别逻辑）
- TRD §4.4 置信度阈值（前端按 confidence < 0.6 / 0.6-0.85 / >= 0.85 三档高亮）

## 实施要点

- ScreenshotImportModal：
  - antd Modal full-screen，header 暗色 #1F1F1F
  - 文件上传输入支持拖放和点选
  - 上传后 useScreenshotImport mutation 调 API
  - 识别结果用 useState 管理，每行可编辑
  - 左右双栏布局：左 38% 原图（可点击放大到全屏 viewer）+ 右 62% RecognizedHoldingTable
  - 表格行字段验证规则按 confidence：
    - >= 0.85：白底
    - 0.6-0.85：黄底（FEF3C7）+ 输入框边框 #F59E0B
    - < 0.6：清空字段并强制要求用户填
  - 底部说明 "将覆盖当前 N 个持仓 (仓位总和 X%)" 或 "将新增 N 个持仓"
  - 确认导入按钮：分别调用现有 POST /api/v1/holdings 接口逐个创建（或批量接口如已有），全部成功后关闭 Modal 并刷新列表
  - 错误处理：超过 5 上限的检测在确认前完成
- ImagePreview：
  - 缩略 + 点击全屏（用 antd Image preview）
- RecognizedHoldingTable：
  - 每行字段：标的名 + 代码 / 成本 / 仓位 / 删除按钮
  - 用 antd Form.List 管理行
- ScreenshotModeForm（修改 step16 的简化版）：
  - 用户在 Drawer 里上传截图后，启动一个内联的简化校对（单个标的，不开 Modal）
  - 这是单个录入路径；批量录入路径走 PortfolioListPage 顶部的 Modal
- PortfolioTransactionsPage：
  - 顶部面包屑 ← Portfolio / 标的名 / 交易记录
  - TransactionTable 显示该 holding 的所有 trades
  - 右上"+ 添加交易记录"按钮打开 AddTransactionDrawer
  - 底部汇总条：综合成本 / 综合仓位 / 总买入金额 / 总卖出金额
- 所有 antd 组件经 ui-kit/eat barrel

## 验证标准

1. `pnpm test src/features/portfolio src/pages/portfolio` 通过
2. 浏览器手动测：
   - 点"📷 截图批量导入"打开 Modal，上传一张测试图
   - mock 的识别结果按 confidence 正确高亮
   - 修改后确认导入，列表正确更新
   - 进入 /portfolio/:id/transactions 看到交易列表
3. 5 上限保护：截图识别出超量时确认前警告
4. `pnpm lint:all` 通过

## 依赖说明

- 前置：step16 Portfolio 列表与 Drawer、step09 后端 import-screenshot API、step07 后端 screenshot service

## 预估提交

- commit 1: `feat(portfolio): add screenshot import modal with dual pane`
- commit 2: `feat(portfolio): wire screenshot mode form to recognition flow`
- commit 3: `feat(portfolio): add transactions sub page`
