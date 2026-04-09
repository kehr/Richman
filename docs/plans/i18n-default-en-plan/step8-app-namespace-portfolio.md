# Step 8: app namespace 迁移 -- portfolio

## 任务目标

将 portfolio 相关页面和组件中的所有硬编码中文替换为 `t("app:...")` 调用。包括持仓列表、交易记录、添加/编辑抽屉、截图导入等。

## 涉及文件

- 修改: `frontend/src/pages/portfolio/PortfolioListPage.tsx`
- 修改: `frontend/src/pages/portfolio/PortfolioTransactionsPage.tsx`
- 修改: `frontend/src/pages/portfolio/components/HoldingTable.tsx`
- 修改: `frontend/src/pages/portfolio/components/RecognizedHoldingTable.tsx`
- 修改: `frontend/src/pages/portfolio/components/AddHoldingDrawer.tsx`
- 修改: `frontend/src/pages/portfolio/components/AddTransactionDrawer.tsx`
- 修改: `frontend/src/pages/portfolio/components/QuickHoldingForm.tsx`
- 修改: `frontend/src/pages/portfolio/components/ScreenshotImportModal.tsx`
- 修改: `frontend/src/pages/portfolio/components/TotalCapitalRow.tsx`
- 修改: `frontend/src/pages/portfolio/components/ImagePreview.tsx`
- 修改: `frontend/src/pages/portfolio/components/TransactionTable.tsx`
- 修改: `frontend/src/pages/portfolio/components/AssetTypeStep.tsx`
- 修改: `frontend/src/features/portfolio/TradeRecordList.tsx`

## PRD/TRD 引用

- PRD §4.1（portfolio 迁移范围）
- TRD §12（字符串迁移约定）
- TRD §12.3（Form rule message useMemo 约定）
- TRD §6（format helpers 传 locale）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过
- [ ] `rg '[\u4e00-\u9fff]' frontend/src/pages/portfolio frontend/src/features/portfolio --type tsx` 结果为零（测试文件除外）
- [ ] `pnpm dev` 启动后 Portfolio 页面默认英文
- [ ] 切中文后持仓表格列头、交易记录、抽屉表单全部中文
- [ ] 数字格式（千分位、日期）随语言切换
- [ ] 添加持仓 / 添加交易的 Form validation message 随语言切换

## 依赖

- Step 3（app namespace JSON 已就绪）
- Step 5（format helpers 已重构）

## 实施注意

- ScreenshotImportModal 有 21 处中文，是 portfolio 最密集的组件，包含上传提示、识别结果表格列头、状态文案
- HoldingTable 有 15 处，包含表格列 title + 操作按钮 + 弹窗确认文案
- AddHoldingDrawer / AddTransactionDrawer 有 13 处，大量 Form.Item label + placeholder + rules message
- Form rules message 必须在组件 body 内 useMemo 包裹
- PortfolioTransactionsPage 同时有中文文案和 formatAmount 调用，需要两头改
- TradeRecordList 在 features/ 下（不是 pages/），注意路径
- AssetTypeStep 中有资产类型选项（股票、基金等），这些是枚举 label，key 用 `app:portfolio.assetType.stock` 等
