# Step 2: App.tsx 集成 + 测试工具更新

## 任务目标

将 i18n 接入 React 树：App.tsx 移除旧 I18nProvider、添加 i18n/config side-effect import、ConfigProvider.locale 响应式绑定。同步更新 test/utils.tsx 测试 wrapper。

## 涉及文件

- 修改: `frontend/src/App.tsx`
- 修改: `frontend/src/test/utils.tsx`

## PRD/TRD 引用

- TRD §4.1（App.tsx 改造后结构）
- TRD §4.3（设计约束：I18nProvider 删除、side-effect import 位置、ConfigProvider locale 驱动源）
- TRD §10（测试工具更新：隔离 i18n 实例）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm dev` 启动后浏览器能看到页面（功能不回归）
- [ ] AntD 组件默认显示英文文案（ConfigProvider.locale 生效的初步信号）
- [ ] `pnpm test` 通过（测试 wrapper 更新后现有测试不挂）

## 依赖

- Step 1（i18n/config.ts 存在才能 import）

## 实施注意

- App.tsx 删除 `import { I18nProvider } from "@/domain/i18n/provider"` 和对应 JSX 标签
- `import "./i18n/config"` 必须放在 App.tsx 所有其他 import 之前（TRD §4.3 约束）
- test/utils.tsx 创建隔离的 i18n 实例（`i18n.createInstance()`），不复用 app 的全局单例
- 本 step 之后旧的 `useLocale` 仍在被 PreferencesTab、HelpPage 等使用（provider.tsx 还没删），这些组件会在后续 step 迁移
