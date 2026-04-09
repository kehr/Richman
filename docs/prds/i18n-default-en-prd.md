# Richman 前端国际化（默认英文 / 可切换中文）PRD

## 1. 背景

当前 Richman 前端存在中英文混杂的状态：

- 已经存在一套 50 行的自写 i18n provider（`domain/i18n/provider.tsx`）默认 zh、localStorage key `richman_locale`
- 仅 3 个业务文件真正消费了 `useLocale`（PreferencesTab、HelpPage、HelpPage.test）
- 同时 50 个 `.tsx` 文件里散落着 410 处硬编码中文字符串（onboarding UX overhaul + LLM settings 合入后新增约 180 处）
- AntD `ConfigProvider` 没设 `locale` prop，DatePicker / Pagination / Form 默认校验消息等全是 AntD 内置英文
- `domain/{money,datetime,ui}/format.ts` 里数字 / 日期 / 货币格式硬编码 `en-GB` / `en-US`
- Settings 页已有语言 Radio 但其自身 label 也是硬编码中文

这导致用户无论停留在哪种语言都会看到另一种语言的文案泄漏，且无法稳定满足「默认英文用户」的首访体验。

## 2. 目标

- 前端默认英文，用户可在 Settings 偏好面板一键切换中文
- 切换即时生效，无需刷新，所有 AntD 组件、业务文案、数字日期格式化同步变化
- 跨会话持久化语言偏好
- 对已有用户（localStorage 已存 `richman_locale`）零中断继承
- 建立面向后续多语言扩展的基础设施（库、目录结构、类型约束）

## 3. 非目标

- 不引入 i18n 管理平台、翻译供应商对接、在线热更新机制
- 不做 RTL（阿拉伯 / 希伯来）支持
- 不扩展到英文 / 中文以外的第三种语言（基础设施留口子，但本期不上）
- 后端不返回本地化字符串，不实现 Accept-Language 协商（backend 目前 0 中文）
- 不做运行时按需加载翻译资源（静态打包，理由见 §8）

## 4. 范围

### 4.1 迁移范围

所有用户可见的前端硬编码中文字符串都进入 i18n 体系，包括但不限于：

- 导航 / 布局（nav、layouts、面包屑）
- 认证页（登录、注册、找回密码、Auth split layout）
- Dashboard（DashboardPage、LLMStatusBanner、EmptyHoldingsHero、DecisionCardWall、ChangeAnchorList）
- Portfolio（PortfolioListPage、PortfolioTransactionsPage、HoldingTable、RecognizedHoldingTable、AddHoldingDrawer、AddTransactionDrawer、QuickHoldingForm、ScreenshotImportModal、TotalCapitalRow、ImagePreview、AssetTypeStep）
- Decision Cards（DecisionCardSummary、ExecutionPlanStrip、SourcePill、ChangeBadge、Conclusion、DimensionReasoning、ExecutionPlan、MainRisks、MetaSidebar）
- Settings（PreferencesTab、AccountTab、SubscriptionTab、ChannelsTab）
- Settings LLM（LLMSection、LLMConfigForm、LLMHealthyCard、LLMFailingCard、LLMProbeButton、LLMEmptyState）
- Onboarding（WelcomePage、FirstAnalysisPage、StepIndicator、OnboardingGuard）
- Notification channels、Help 页结构化内容
- Auth（LoginForm、RegisterForm、AuthSplitLayout、SampleDecisionCard）
- 类型定义中的中文（asset-catalog/types.ts、notification-channels/types.ts、settings-llm/api.ts）
- Form 校验 rule 的 `message`、AntD message/notification 文本、Tooltip、Popconfirm 文案
- 数字、日期、货币格式化输出（§5.3）
- 涉及数量的文案需使用复数形式（英文 `_one` / `_other` 后缀，中文可共用同一值）
- 翻译文本中嵌入 React 组件（链接、加粗、图标）的文案必须使用 `<Trans>` 组件而非 `t()`

基础设施改造：

- 引入 `react-i18next` + `i18next` + `i18next-browser-languagedetector`
- 删除自写 `domain/i18n/provider.tsx` 并迁移 3 个 consumer
- `src/i18n/locales/{en,zh}/{namespace}.json` 新结构
- `ui-kit/eat` barrel 导出 antd 的 `zhCN` / `enUS` locale
- `App.tsx` 的 `ConfigProvider.locale` prop 响应式绑定
- `test/utils.tsx` 测试 wrapper 加 i18next provider
- `layouts/MainLayout.tsx` 侧边栏底部新增语言快速切换入口（Globe 图标 + 当前语言文字），详见 §5.5.2

### 4.2 本期内完成的相关改造

- `domain/money/format.ts`、`domain/datetime/format.ts`、`domain/ui/format.ts` 的 Intl locale 参数从硬编码改为读取 i18next 当前语言
- Help page 结构化内容从 `frontend/src/i18n/help/` 搬到 `src/i18n/locales/{en,zh}/help-content.json`
- 所有 AntD `Form` 组件的自定义 `rules[].message` 改为走 `t()`

### 4.3 本期不做

- 任何后端改动
- 数字 / 货币符号（¥ vs $）的全局切换（格式只变 locale，货币仍是人民币）
- i18n key 的翻译审校工作流
- 自动化检查新增硬编码中文的 lint 规则（留到后续）

## 5. 用户可见行为

### 5.1 首次访问（无 localStorage 记录）

- 读取 `navigator.language`（或 `navigator.languages[0]`）
- 以 `zh-*` 开头（zh-CN / zh-TW / zh-HK 等）则界面初始语言为中文。i18next 必须配置 `load: 'languageOnly'` 将 `zh-CN` / `zh-TW` 等 strip 到 `zh`，否则找不到对应资源文件夹
- 其他所有值（包括空、undefined）落到英文。i18next 必须配置 `supportedLngs: ['en', 'zh']` 拒绝存入非法语言值
- 该次渲染即时生效，用户看不到加载态或英文闪烁。i18next 显式配置 `react: { useSuspense: false }`，因为资源同步加载不需要 Suspense，避免 Strict Mode 下误触发 Suspense boundary
- languagedetector 的 `detection.caches` 显式设为 `['localStorage']`，避免默认同时写入 cookie

### 5.2 已有偏好（localStorage 有 `richman_locale` = `zh` 或 `en`）

- 忽略 `navigator.language`
- 按存储值渲染
- 若存储值为未知字符串（老数据损坏、手动篡改）→ 回退 `en` 并覆盖写入合法值

### 5.3 切换语言

两个入口，任选其一即可触发切换，行为相同：

1. **Settings 入口**：用户进 Settings → Preferences → 语言 Radio 选另一项（详见 §5.5.1）
2. **侧边栏快切入口**：用户点击 MainLayout 侧边栏底部的 Globe 图标，弹出 dropdown，选中目标语言（详见 §5.5.2）

任一入口触发后，共同的切换流程：

- 触发 `i18n.changeLanguage(newLocale)`
- 所有使用 `useTranslation()` 的组件即时重渲染（包括两个入口自身，保证两处 UI 的选中态 / 显示语言同步）
- `ConfigProvider.locale` 同步切换，AntD 内置组件文案（DatePicker 月份、Pagination 翻页、Modal 按钮等）同步
- dayjs locale 同步切换（`dayjs.locale('zh-cn')` / `dayjs.locale('en')`），确保 DatePicker / Calendar 的月份名、星期名跟随语言。dayjs 的 locale 切换必须在 `i18n.on('languageChanged')` 回调中执行
- 数字 / 日期 / 货币格式化输出同步变化
- localStorage 的 `richman_locale` 被 i18next-browser-languagedetector 自动持久化
- `<html lang>` 属性同步更新为 BCP 47 tag（`en` / `zh`）

### 5.4 跨会话

- 下次访问直接读 localStorage，§5.2 路径
- 同一浏览器多标签同时打开：切换一个标签语言不影响其他标签的即时渲染（localStorage storage 事件不订阅，各标签独立）；下次打开新标签才读最新偏好。本 MVP 不做跨标签同步

### 5.5 语言切换入口 UI 规格

#### 5.5.1 Settings → Preferences Tab

Preferences tab 内部所有 label 都走 i18n：

- 顶部 label「语言 / Language」走 `t('settings.preferences.languageLabel')`
- Radio 的两个选项 label 保持各自语言硬写：`中文` / `English`（不走 t()，因为选项本身就是要展示目标语言）
- 时区 label / 时区下拉提示 / 主题 label / 亮色 label / 数字格式折叠 label / 折叠内部说明文字 全部走 i18n
- Radio value 绑定 `i18n.language`（或其标准化到 `zh` / `en` 的映射）
- Radio `onChange` 直接调用 `i18n.changeLanguage(newValue)`，不走任何本地中间 state

#### 5.5.2 MainLayout 侧边栏快切入口

- 位置：`MainLayout.tsx` 的 `menuFooterRender` 里，当前 flex row 从「avatar (left) | help (right)」改为「avatar (left) | language (center) | help (right)」三子布局
- 触发元素：AntD `Dropdown`，trigger 为 `GlobalOutlined` 图标 + 当前语言的母语自称短标签
  - 英文状态下显示 `EN`
  - 中文状态下显示 `中文`
  - 短标签宽度限制在 2 个字符以内，避免挤压相邻元素
- Dropdown 展开项固定为两项，顺序固定为 `English` → `中文`：
  - 选项 label 保持各自母语写法（与 Settings Radio 一致）
  - 当前语言对应项带 checkmark 或高亮，由 AntD Menu 的 `selectedKeys` 控制
- 选中某项 → 调用 `i18n.changeLanguage(newValue)` → dropdown 自动关闭
- Hover trigger 时显示 tooltip，文案走 `t('nav.switchLanguage')`（英文 "Switch language" / 中文 "切换语言"）
- 键盘可达：trigger 必须可 Tab 聚焦，按 Enter 打开 dropdown，方向键在菜单项间移动
- 移动端 / 窄屏：本期 ProLayout `collapsed=false` 硬写、不支持窄屏，本入口跟随整个侧边栏一同在移动端不可见，不做额外适配

### 5.6 错误与空状态

- 翻译 key 缺失：react-i18next 按 `fallbackLng: 'en'` 回退；en 也缺则显示 key 名。开发模式 key 缺失应被 key 名暴露以便发现
- 组件加载前的 suspense 窗口：禁止出现，因为 resources 静态 import，初始化同步完成

## 6. 状态空间

三个维度：`stored ∈ {empty, en, zh, corrupt}`、`detected ∈ {en-*, zh-*, other}`、`locale_result ∈ {en, zh}`

| stored | detected | locale result | 分类 | 说明 |
|--|--|--|--|--|
| empty | en-US / en-GB / ... | en | Valid | 英文用户首访 |
| empty | zh-CN / zh-TW / ... | zh | Valid | 中文用户首访 |
| empty | ja / ko / de / ... | en | Valid | 非中非英回退 |
| empty | undefined | en | Valid | navigator.language 不可用回退 |
| en | zh-CN | en | Valid | 用户偏好覆盖检测 |
| zh | en-US | zh | Valid | 同上反向 |
| en | en-US | en | Valid | 偏好与检测一致 |
| zh | zh-CN | zh | Valid | 偏好与检测一致 |
| corrupt ("fr" / "" / "xyz") | any | en | Valid | 未知偏好回退并修正存储 |
| any | any + AntD locale ≠ i18n locale | - | **Forbidden** | ConfigProvider 必须原子响应 i18n.language，不允许两者异步 |
| init-in-flight | any | undefined | **Transient** | 禁止存在；静态 import 确保 init 同步完成于首帧之前 |

## 7. 替代路径验证

每一条都必须通过审阅：

| 路径 | 处理 |
|--|--|
| Back 导航（SPA 内） | 不 remount App，i18next 单例状态持久，locale 不变 |
| 刷新 / 重新打开标签 | 从 localStorage 读上次偏好，§5.2 |
| 切换语言后立刻导航 | changeLanguage 同步更新 i18next state，React batch update，到达下一页面时 locale 已切完 |
| 连续快速切换（疯狂点 Radio / Dropdown） | i18next.changeLanguage 幂等，最后一次生效，ConfigProvider 响应最后状态 |
| 两入口混用（一边 Radio 一边 Dropdown） | 两者都订阅 useTranslation hook，都读 i18n.language 单一数据源，选中态永远一致 |
| 用户在 Settings 页点 Sidebar Dropdown | Sidebar Dropdown 在 MainLayout（父布局）里，Settings 是子路由，两者共存。两处都即时更新 |
| Form 校验 message 切换时机 | 校验规则在 form submit / blur 时重新执行，切换语言后下一次触发校验显示新语言 message。已在 rule message 的组件内 useMemo / 重新计算 |
| Help 页切换语言 | Section ids 保持跨语言稳定 → IntersectionObserver 不需要重建（或重建也安全因为 id 一致） |
| Missing key | fallback 到 en，仍缺则显示 key 名 |
| localStorage 被第三方脚本清空 | 回到首访流程（navigator.language 检测） |
| StrictMode 双渲染 | i18next init 幂等，ConfigProvider 绑定 state 幂等 |
| 浏览器禁用 localStorage（隐私模式） | languagedetector 降级到 navigator.language 检测，无法持久化，每次刷新回到默认。可接受 |
| 用户同时切换语言和主题 | i18n 与主题使用独立 state，互不影响 |

## 8. Pre-mortem

| 潜在 bug | 根因 | 设计防御 |
|--|--|--|
| AntD DatePicker / Pagination 切 zh 后仍是英文 | `ConfigProvider.locale` 未绑定 i18n.language | App.tsx 用 `useTranslation()` 取 i18n.language，map 到对应 antd locale 对象，传入 ConfigProvider.locale prop，作为 reactive 依赖 |
| DatePicker 日历月份名切 zh 后仍是 January | dayjs locale 独立于 AntD ConfigProvider | 在 `i18n.on('languageChanged')` 回调中执行 `dayjs.locale()`，初始化时也要同步 |
| 浏览器检测到 zh-CN 但资源只有 zh，整个中文检测失效 | i18next 默认精确匹配 locale tag | 配置 `load: 'languageOnly'` strip 地区后缀 + `supportedLngs: ['en', 'zh']` 白名单 |
| 英文复数形式不正确（1 items 而非 1 item） | 翻译 key 没有 `_one`/`_other` 后缀 | TRD 规定所有涉及数量的 key 必须提供复数形式 |
| 自定义 Form rule message 不切语言 | `rules=[{ message: '请输入' }]` 是组件 body 外的常量 | 所有自定义 rule message 改为走 t() 并放在组件内部 useMemo 或直接在 body 里计算 |
| 数字 / 日期 / 货币不切换 | `Intl.NumberFormat('en-GB', ...)` 硬编码 | format helpers 接受 locale 参数或直接读 i18next.language |
| 首帧闪 key 名（FOUC-key） | i18next 用 HTTP backend 异步加载 | 全部用静态 import 注入 resources，init 同步完成 |
| 老用户 localStorage 偏好丢失 | i18next-browser-languagedetector 默认 key `i18nextLng` | 配置 `detection.lookupLocalStorage = 'richman_locale'` 沿用老 key |
| Help page 切语言后 deep link / section 滚动坏 | zh 和 en 的 help-content.json sections 数组 id 不一致 | TRD 强制约束：两个 content 文件的 sections 必须 id 序列严格一致，类型系统校验 |
| Biome / dep-cruiser 阻止直接 import `antd/locale` | 项目强制 ui-kit/eat barrel | 先改 barrel 新增导出，再用 |
| 测试全挂 | `test/utils.tsx` 没 I18nextProvider wrapper | barrel 改造同 PR 更新测试 wrapper，预加载全部 resources |
| 模板字符串里的中文漏迁 | 扫描只看 `"..."`、`'...'`，`\`...\${}...\`` 里的中文逃逸 | 人工审查阶段两遍扫描：一遍通配 `[\u4e00-\u9fff]`，一遍专扫 template literal |
| changeLanguage 后 useTranslation 没重渲染 | 组件层 memo 边界没 subscribe locale 变化 | react-i18next 自身处理：useTranslation 内部订阅语言变化事件，触发 rerender |
| 中文下空间溢出（UI 挤爆） | 英文的按钮 / 标签到中文后膨胀或相反 | 本期不做专项布局走查；依赖开发人工 smoke test；极端样式冲突记为后续 ticket |
| Sidebar Dropdown 显示的当前语言与真实不符 | Dropdown 组件从 localStorage 直读而非 i18n.language | 强制从 useTranslation hook 读语言，禁止组件直读 localStorage |
| Settings Radio 和 Sidebar Dropdown 不同步 | 两组件用不同 state 源 | 两者必须都订阅 useTranslation，选中态由 i18n.language 单一源驱动 |
| Sidebar 底部 3 子布局在默认宽度下挤爆 | 新加的 language dropdown 占用空间 | menuFooterRender 用 `justifyContent: 'space-between'` + language dropdown 的 trigger 限制为图标 + 2 字母短标签；必要时隐藏 help 文字 |
| Globe 图标被误解为「切换地区」 | 视觉符号歧义 | trigger 同时显示语言短标签；hover tooltip 明确「Switch language / 切换语言」 |

## 9. 设计决策记录

### 9.1 库选型：react-i18next

替代：保留自写 provider（太简陋）、react-intl（ICU 语法对 MVP 过重）、lingui（生态窄）

选 react-i18next：生态最成熟、`useTranslation` API 标准、TS 类型插件能从 JSON 生成 key 字面量类型、插值/复数/namespace/fallback 全部开箱、bundle ~10KB gzipped 可接受。

### 9.2 文件布局：每 locale 一个文件夹 + namespace 拆分

```
frontend/src/i18n/
  config.ts                   # i18next 初始化
  locales/
    en/
      common.json             # nav + 跨页复用 UI（modal、hero、button、空状态）
      auth.json               # 登录 / 注册 / onboarding / guard
      app.json                # dashboard + portfolio + decision-card（核心业务）
      settings.json           # settings tabs + notification channels + LLM 配置
    zh/
      （同上 4 个 namespace）
  help/
    en.json                   # Help 页结构化内容（HelpContent 类型）
    zh.json
    types.ts
    index.ts                  # getHelpContent loader（非 i18next namespace）
```

namespace 合并为 4 个（common / auth / app / settings），每个约 100 key，降低 `useTranslation` 多 namespace 的心智负担。原 9 拆 namespace 对 410 key 的规模偏碎（平均 45 key/ns），合并后边界更清晰。

help-content 保持为独立 TS 模块，不注册为 i18next namespace，因为 i18next 的扁平 KV 机制不适合数组/对象结构化内容。help 目录从原 `i18n/help/` 位置保留不变（不搬入 locales/）。

### 9.3 默认检测：navigator.language + en 回退

替代：硬 en（首访中文用户体验差）、硬弹窗（UX 重）

采用浏览器检测：覆盖了「中文用户第一眼看中文」和「英文用户第一眼看英文」两个主路径，且不阻塞首访。用户偏好一旦存储就覆盖检测。

### 9.4 迁移范围：一次性全量

替代：分两期、分三期

选一次性：全量迁移后不存在「一半页面变一半不变」的中间态；review 负担用 Plan 的 step 粒度细化对冲（基础设施 → 按 namespace 批量迁移 → AntD ConfigProvider 联动 → 格式化层 → Form rule message → 测试更新 → Help 搬迁 → polish）。

### 9.5 Format helpers 同步改造

替代：只迁字符串、format 留 en-GB

选同步：切中文后数字日期货币仍是英式会产生明显的不一致感（"Total: ¥1,234.56" 混在一堆中文 label 中），没必要留半成品。代价是 Plan 多 2-3 步。

### 9.6 Help 内容保持独立模块

替代：搬入 `src/i18n/locales/{en,zh}/help-content.json` 统一收纳

选保留 `frontend/src/i18n/help/`：i18next 的 namespace 机制期望扁平或嵌套的 KV 结构，不适合存储 Help 页的数组/对象结构化内容（HelpContent 类型含 sections 数组）。滥用 i18next 做内容管理会引入类型不安全。help-content 继续通过独立的 `getHelpContent(locale)` 加载器提供，从 `useTranslation().i18n.language` 读取当前语言。

### 9.7 双入口（Settings + Sidebar Globe）

替代：A 只 Settings、B Settings + avatar dropdown 子菜单、C Settings + sidebar 独立一行 Language 标签

选 Settings + Sidebar Globe 图标：
- 与 sidebar 既有的 help 图标对称，视觉协调
- 发现性比 avatar 子菜单好（不需要先点 avatar 才能看到）
- 比独立一行轻量，不抢 sidebar 垂直空间
- `GlobalOutlined` 已在 eat barrel 里导出，零 barrel 改动
- Dropdown trigger 的「Globe + 当前语言短标签」组合降低图标歧义

## 10. 成功标准

上线（合入 main）判定条件：

1. 默认进入前端看到英文界面（navigator.language 为 en-* 或未设置）
2. navigator.language 为 zh-* 的首访看到中文
3. Settings Preferences → 语言 Radio 切换立即生效，所有组件（包括 AntD 内置文案）同步变化
4. MainLayout 侧边栏底部的 Globe Dropdown 切换立即生效，效果与 Settings Radio 完全一致
5. 两入口的选中态永远同步：在 Settings 切换后立即回看 Sidebar Dropdown 的 trigger label 和 selected state 也变了；反之亦然
6. 刷新后语言偏好持久
7. 老用户的 localStorage `richman_locale` 值被读取并生效
8. `grep -rn '[\u4e00-\u9fff]' frontend/src --include='*.tsx'` 的结果只剩 i18n 资源文件（JSON）、注释、以及语言选项 label（`中文`） —— 不再有业务文案硬编码
9. `pnpm lint:all` 通过
10. `pnpm test` 通过
11. Help 页切换语言、deep link、IntersectionObserver 行为不变
12. AntD DatePicker 月份名、Pagination 翻页文字、Form 默认校验消息、Modal 按钮在切 zh 后全部为中文
13. 数字 / 日期 / 货币格式化输出随语言变化

## 11. 不影响的事项

- 后端 API 合约（零变更）
- 数据库 schema（零变更）
- 路由结构（零变更）
- 认证 / 权限逻辑（零变更）
- 样式 / 主题 token（零变更）

## 12. TRD 必须覆盖的行业最佳实践 gap

以下 gap 在 PRD 层面不展开，但 TRD 必须给出具体方案：

1. **TypeScript 类型安全**：用 i18next CLI 从 JSON 生成 `resources.d.ts` + module augmentation，让 `t()` key 有字面量类型
2. **Format helpers 保持纯函数**：`formatAmount(amount, locale)` 接受 locale 参数，调用方从 React 层传入，不在 helper 内读全局 i18next 单例
3. **Key 命名约定**：dot notation + camelCase（`auth.loginButton`、`portfolio.addHolding.title`）
4. **`interpolation.escapeValue: false`**：React 自动 escape，i18next 不需要重复
5. **测试两层策略**：单元测试 mock `t()` 返回 key 本身（快），集成测试用真实翻译
6. **i18next-cli key 提取**：编码完成后跑一遍检测孤立 key 和缺失 key；后续可加入 CI
7. **Intl.NumberFormat 实例缓存**：切语言后需新 formatter 实例，用 locale 做 key 的 Map 缓存
8. **`<Trans>` 组件使用规范**：明确哪些文案需要嵌入 React 组件，规定 `<Trans>` 与 `t()` 的选择边界

## 13. 时序与依赖

- 本 PRD 与 onboarding UX overhaul 在时间线上互斥
- onboarding 合入 main 前：本 feature 只写文档（PRD / TRD / Plan），不动代码
- onboarding 合入 main 后：`feat/i18n-default-en` 分支 rebase onto main，然后进入编码
- 编码阶段会一并迁移 onboarding 在 main 阶段新加的所有中文字符串，因此 onboarding 无需为 i18n 预留任何接口
