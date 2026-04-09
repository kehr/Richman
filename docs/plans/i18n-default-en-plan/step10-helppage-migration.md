# Step 10: HelpPage 迁移

## 任务目标

将 HelpPage 和 HelpSidebar 从旧 useLocale 迁移到 useTranslation。Help 结构化内容（`i18n/help/`）保持独立模块不变，只更换 locale 读取源。

## 涉及文件

- 修改: `frontend/src/pages/help/HelpPage.tsx`
- 修改: `frontend/src/pages/help/components/HelpSidebar.tsx`（如果有中文）
- 修改: `frontend/src/pages/help/components/HelpSection.tsx`（如果有中文）
- 修改: `frontend/src/i18n/help/index.ts`（如果需要调整 import 或 fallback 逻辑）

## PRD/TRD 引用

- TRD §9（HelpPage 迁移：useLocale → useTranslation，locale 类型 cast）
- TRD §9.2（Help 模块位置不变）
- TRD §9.3（Section ID 跨语言稳定性约束）
- PRD §9.6（Help 保持独立模块决策）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过（HelpPage.test.tsx 随后在 Step 11 更新）
- [ ] HelpPage 不再 import `useLocale`
- [ ] `pnpm dev` 启动后 Help 页面可正常加载
- [ ] 切中文后 Help 页面内容切换为中文
- [ ] Deep link `/help#badge` 在两种语言下都能正确滚动到对应 section
- [ ] IntersectionObserver 高亮在语言切换后仍然正常工作
- [ ] `rg '[\u4e00-\u9fff]' frontend/src/pages/help --type tsx` 结果为零（测试文件除外）

## 依赖

- Step 2（App.tsx 已接入 i18n）

## 实施注意

- HelpPage 是旧 useLocale 的最后 2 个消费者之一（HelpPage + HelpPage.test），本 step 完成后只剩 test 文件
- `getHelpContent(locale)` 的参数从 `useLocale().locale` 改为 `useTranslation().i18n.language as "en" | "zh"`
- Section ID 必须在 en.json 和 zh.json 之间完全一致（TRD §9.3），本 step 不改 JSON 内容，只改读取方式
- HelpSidebar 和 HelpSection 如果不直接使用 useLocale，则不需要改动（它们通过 props 接收数据）
- Help page 的 title / subtitle 在 HelpContent 结构里，不在 i18next namespace 里
