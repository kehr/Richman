# Step 6: auth namespace 迁移

## 任务目标

将 auth + onboarding 相关组件中的所有硬编码中文替换为 `t("auth:...")` 调用。包括登录/注册表单、AuthSplitLayout、SampleDecisionCard、全部 onboarding 页面、OnboardingGuard。

## 涉及文件

- 修改: `frontend/src/features/auth/LoginForm.tsx`
- 修改: `frontend/src/features/auth/RegisterForm.tsx`
- 修改: `frontend/src/pages/auth/components/AuthSplitLayout.tsx`
- 修改: `frontend/src/pages/auth/components/SampleDecisionCard.tsx`
- 修改: `frontend/src/pages/onboarding/WelcomePage.tsx`
- 修改: `frontend/src/pages/onboarding/FirstHoldingPage.tsx`
- 修改: `frontend/src/pages/onboarding/CategoriesPage.tsx`
- 修改: `frontend/src/pages/onboarding/FirstAnalysisPage.tsx`
- 修改: `frontend/src/pages/onboarding/LLMConsentPage.tsx`
- 修改: `frontend/src/pages/onboarding/components/StepIndicator.tsx`
- 修改: `frontend/src/pages/onboarding/components/OnboardingLayout.tsx`
- 修改: `frontend/src/domain/auth/onboarding-guard.tsx`

## PRD/TRD 引用

- PRD §4.1（auth + onboarding 迁移范围）
- TRD §12（字符串迁移约定：每组件 6 步）
- TRD §11（key 命名约定）
- TRD §12.3（Form rule message 约定：useMemo 包裹）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过
- [ ] `rg '[\u4e00-\u9fff]' frontend/src/features/auth frontend/src/pages/auth frontend/src/pages/onboarding frontend/src/domain/auth --type tsx` 结果为零（测试文件除外，留到 Step 11）
- [ ] `pnpm dev` 启动后登录/注册页默认显示英文，切中文后全部中文
- [ ] Onboarding 全流程可走通且文案随语言切换

## 依赖

- Step 3（auth namespace JSON 已就绪）

## 实施注意

- LoginForm / RegisterForm 中的 Form rules message 必须在组件 body 内用 useMemo 包裹（TRD §12.3）
- SampleDecisionCard 中的中文是展示用文案（推荐、趋势等标签），对应 key 进 auth namespace（因为只在 auth 页面使用）
- OnboardingGuard 中的中文可能是 console 日志或条件判断中的字符串，需要甄别哪些是用户可见文案、哪些是内部逻辑
- FirstHoldingPage 是 33 处中文最密集的单文件，注意 Form item label / placeholder / validation message 全覆盖
- LLMConsentPage 有 18 处中文，包含较长的说明段落，可能需要用 Trans 组件处理内嵌链接
