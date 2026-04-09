# Step 1: i18n 基础设施

## 任务目标

安装 react-i18next 及相关依赖，创建 i18n 初始化配置、TypeScript 类型声明、AntD locale 映射模块。本 step 完成后 i18n 单例已可用但尚未接入 React 树。

## 涉及文件

- 修改: `frontend/package.json`（新增依赖）
- 新建: `frontend/src/i18n/config.ts`
- 新建: `frontend/src/i18n/@types/i18next.d.ts`
- 新建: `frontend/src/i18n/antd-locale.ts`
- 修改: `frontend/tsconfig.json`（如需将 `@types` 目录纳入编译）

## PRD/TRD 引用

- TRD §1（依赖变更）
- TRD §2（i18next 初始化 config + 6 个约束）
- TRD §3（TypeScript 类型安全 module augmentation）
- TRD §4.2（AntD locale 映射）

## 验证标准

- [ ] `pnpm install` 成功，`i18next`、`react-i18next`、`i18next-browser-languagedetector` 出现在 dependencies
- [ ] `pnpm lint:all` 通过
- [ ] `frontend/src/i18n/config.ts` 可被 TypeScript 编译（`pnpm type-check`）
- [ ] 手动验证：在 config.ts 中 `t("nav.dashboard")` 有类型提示（需要 Step 3 的 JSON 才完整，本 step 先用占位 JSON 验证类型管道通畅）

## 依赖

无前置依赖。

## 实施注意

- config.ts 的 static import 路径指向 Step 3 将要创建的 JSON 文件。本 step 先创建最小骨架 JSON（每个 namespace 至少一个 key）以通过编译，Step 3 填充完整内容
- `@types/i18next.d.ts` 的 module augmentation 严格按 TRD §3.1 写法，resources 引用 `en` locale 作为 source of truth
- antd-locale.ts 的导入路径 `antd/locale/en_US` 和 `antd/locale/zh_CN` 不走 eat barrel（TRD §5 说明）
