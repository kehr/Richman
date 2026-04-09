# Step 4: MainLayout i18n（语言 Dropdown + nav 菜单）

## 任务目标

在 MainLayout 侧边栏底部添加语言快速切换 Dropdown（Globe 图标 + 当前语言标签），同时将 sidebar nav 菜单项名称和 footer 文案迁移到 t()。

## 涉及文件

- 修改: `frontend/src/layouts/MainLayout.tsx`

## PRD/TRD 引用

- PRD §5.5.2（Sidebar Globe Dropdown 完整 UI 规格：位置、trigger、展开项、选中态、tooltip、键盘可达）
- TRD §7.1（menuFooterRender 改造 JSX 结构）
- TRD §7.2（约束：Dropdown label 硬写母语、selectedKeys 绑 i18n.language）
- TRD §7.3（menuRoutes i18n：从模块顶层常量移入组件 body useMemo）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm dev` 启动后侧边栏底部可见 Globe 图标 + 当前语言短标签（默认 EN）
- [ ] 点击 Globe → dropdown 展开两项（English / 中文），当前项有高亮
- [ ] 选中「中文」→ sidebar nav 菜单项名称切为中文、Globe 短标签切为「中文」、所有已接 i18n 的文案同步切换
- [ ] sidebar nav 的 Dashboard / Portfolio / Settings 文案随语言切换
- [ ] Footer 的 Logout / Help 文案随语言切换
- [ ] Hover Globe 显示 tooltip「Switch language / 切换语言」
- [ ] `pnpm test` 通过

## 依赖

- Step 2（App.tsx 已接入 i18n，useTranslation 可用）
- Step 3（common namespace JSON 中有 nav.* 和 nav.switchLanguage key）

## 实施注意

- menuRoutes 从模块顶层常量移入组件函数体内 useMemo，依赖 [t]
- menuFooterRender 内 flex row 从 2 子变 3 子（avatar | language | help），保持 space-between 布局
- Dropdown 的 `menu.items` label 硬写 "English" / "中文"，不走 t()
- GlobalOutlined 已在 eat barrel 导出，直接用
- 本 step 完成后 MainLayout 不再有硬编码中文（nav 项 + footer 文案全部 i18n）
