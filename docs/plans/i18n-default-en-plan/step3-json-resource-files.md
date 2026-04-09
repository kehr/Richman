# Step 3: JSON 资源文件

## 任务目标

创建 4 个 namespace 的完整 en/zh JSON 翻译文件。将现有 `domain/i18n/{en,zh}.json` 的 key 迁入新结构，并预先为 Step 6-9 需要迁移的所有硬编码中文字符串创建对应 key。

## 涉及文件

- 新建: `frontend/src/i18n/locales/en/common.json`
- 新建: `frontend/src/i18n/locales/en/auth.json`
- 新建: `frontend/src/i18n/locales/en/app.json`
- 新建: `frontend/src/i18n/locales/en/settings.json`
- 新建: `frontend/src/i18n/locales/zh/common.json`
- 新建: `frontend/src/i18n/locales/zh/auth.json`
- 新建: `frontend/src/i18n/locales/zh/app.json`
- 新建: `frontend/src/i18n/locales/zh/settings.json`
- 修改: `frontend/src/i18n/config.ts`（Step 1 的骨架 JSON import 替换为完整文件）

## PRD/TRD 引用

- PRD §9.2（namespace 合并为 4 个 + 文件布局）
- TRD §11（key 命名约定：dot notation + camelCase）
- TRD §11.3（复数形式：_one / _other 后缀）
- PRD §4.1（迁移范围：完整组件清单）

## 验证标准

- [ ] 8 个 JSON 文件都是合法 JSON，`pnpm lint:all` 通过
- [ ] en 和 zh 的每个 namespace 文件 key 结构完全一致（同一个 key 在两边都存在）
- [ ] `pnpm type-check` 通过（config.ts 的 static import 指向新文件）
- [ ] key 覆盖现有 `domain/i18n/{en,zh}.json` 的全部 key（映射关系清晰）
- [ ] key 覆盖 Step 6-9 需要迁移的全部组件中的硬编码字符串（通过 `rg '[\u4e00-\u9fff]' frontend/src --type tsx` 的结果做交叉验证）

## 依赖

- Step 1（config.ts 存在）

## 实施注意

- namespace 边界按 TRD 说明：common=nav+跨页复用、auth=登录注册+onboarding+guard、app=dashboard+portfolio+decision-card、settings=设置tabs+LLM配置+通知渠道
- 现有 `domain/i18n/{en,zh}.json` 的 key 需要按新 namespace 边界重新分配（比如原 `nav.*` 进 common，原 `auth.*` 进 auth，原 `portfolio.*` 进 app 等）
- 不要遗漏测试文件中的中文断言字符串对应的 key（测试文件迁移在 Step 11，但 key 在这一步就要准备好）
- 每个涉及数量的 key 提供 `_one` / `_other` 后缀（TRD §11.3）
- 本 step 产出的 JSON 是「完整的翻译字典」，后续 step 的迁移工作只做「组件代码中的字符串替换 → t() 调用」，不再增删 JSON key
