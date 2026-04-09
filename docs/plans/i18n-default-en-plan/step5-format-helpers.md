# Step 5: 格式化工具重构

## 任务目标

重构 `domain/money/format.ts` 和 `domain/ui/format.ts`，所有公开函数新增 `locale` 参数，内部使用 Intl 实例缓存。保持纯函数特性，不引入对 i18next 全局单例的依赖。

## 涉及文件

- 新建: `frontend/src/domain/money/intl-cache.ts`
- 修改: `frontend/src/domain/money/format.ts`
- 修改: `frontend/src/domain/ui/format.ts`
- 修改: 所有调用 formatAmount / formatCurrency / formatDate / formatPercent(ui) 的组件（约 16 个文件，添加 locale 参数传递）

## PRD/TRD 引用

- PRD §4.2（format helpers 同步改造）
- PRD §12 T2（format helpers 保持纯函数）
- TRD §6（格式化工具重构完整设计：纯函数签名、Intl 缓存、locale 映射、约束）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过（现有 format 相关测试更新后仍 pass）
- [ ] formatAmount / formatCurrency / formatDate 接受 locale 参数，默认值为 "en"
- [ ] `pnpm dev` 启动后数字/日期格式显示正常
- [ ] 切换语言后数字千分位分隔符和日期格式随之变化（zh 用 zh-CN 格式，en 用 en-US 格式）
- [ ] intl-cache.ts 的 Map 缓存工作正常（不会每次渲染 new Intl.NumberFormat）

## 依赖

- Step 1（i18n 基础设施可用）

## 实施注意

- 先扫描所有调用点：`rg "formatAmount|formatCurrency|formatDate" frontend/src --type tsx -l` 获取完整文件列表
- 每个调用点需要在 React 组件中通过 `useTranslation().i18n.language` 获取 locale 传入
- `¥` 货币符号保持硬编码不变（PRD §4.3）
- `domain/money/format.ts` 的 `formatPercent` 不涉及 locale（百分比格式固定），不改
- `domain/ui/format.ts` 的 `formatPercent` 和 `formatConfidence` 也不涉及 locale，不改
- intl-cache.ts 可被 money/format.ts 和 ui/format.ts 共享引用
- locale 参数类型为 `string`，映射逻辑（"zh" → "zh-CN"）在函数内部完成（TRD §6.5）
