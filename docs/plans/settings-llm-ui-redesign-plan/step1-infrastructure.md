# Step 1: 基础设施 — eat barrel + formatRelativeTime

**依赖：** 无（Phase 1，可与 Step 2 并行）

## 任务目标

1. 在 eat barrel 中新增 `EllipsisOutlined` 导出，供 ProviderCardLayout Dropdown 触发按钮使用
2. 新建 `formatRelativeTime` 工具函数，基于 `Intl.RelativeTimeFormat` 将 date 转换为相对时间字符串（"2 分钟前" / "2 minutes ago"）
3. 新建对应的纯函数单元测试

## 涉及文件

- 修改：`frontend/src/ui-kit/eat/index.ts`
- 创建：`frontend/src/features/settings-llm/utils/formatRelativeTime.ts`
- 创建：`frontend/src/features/settings-llm/utils/formatRelativeTime.test.ts`

## 设计依据

- PRD §eat barrel 变更：新增 `EllipsisOutlined` 导出
- TRD §formatRelativeTime 函数：函数签名、null 处理、阈值表、`Intl.RelativeTimeFormat` 用法
- TRD §eat barrel 变更：在 `@ant-design/icons` 导出块新增一行

## 实施步骤

- [ ] **1.1** 在 `frontend/src/ui-kit/eat/index.ts` 的 `@ant-design/icons` 导出块末尾添加 `EllipsisOutlined,`
  - 验证：grep `EllipsisOutlined` eat barrel 可找到该导出

- [ ] **1.2** 创建 `frontend/src/features/settings-llm/utils/formatRelativeTime.ts`
  - 实现 `formatRelativeTime(date, lang)` 函数
  - 参照 TRD §formatRelativeTime 函数章节：null/undefined 返回 `"—"`，阈值表，负数 format 调用

- [ ] **1.3** 创建 `frontend/src/features/settings-llm/utils/formatRelativeTime.test.ts`
  - 覆盖：null 输入、秒级、分钟级、小时级、天级、"zh" 和 "en" 输出格式
  - 注意：前端测试规范只允许纯函数 `.test.ts`，不允许 UI 测试（CLAUDE.md）

- [ ] **1.4** 运行 `cd frontend && pnpm lint:all`，修复全部 lint / type 错误

- [ ] **1.5** 提交
  - `git add frontend/src/ui-kit/eat/index.ts frontend/src/features/settings-llm/utils/`
  - commit message: `feat(settings-llm): add EllipsisOutlined barrel export and formatRelativeTime util`

## 验证标准

- `EllipsisOutlined` 在 eat barrel 可导入，不报 TS 错误
- `formatRelativeTime(null, "zh")` 返回 `"—"`
- `formatRelativeTime(new Date(Date.now() - 120000), "zh")` 返回 `"2 分钟前"`（约值）
- `formatRelativeTime(new Date(Date.now() - 120000), "en")` 返回 `"2 minutes ago"`（约值）
- `pnpm lint:all` 全部通过
