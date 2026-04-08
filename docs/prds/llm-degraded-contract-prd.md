# LLM 降级契约与用户自选 Provider 产品需求文档

## 背景与驱动

2026-04-09 在调试 `analysis/synthesis.Synthesizer` 的 nil-panic 时发现，Richman 的分析流水线在 LLM provider 不可用时会崩溃。临时修复（P0 commit 0ad2ce3）加上了 nil 短路守卫，让单次调用不再 panic，但暴露出更深层的产品问题：

第一，Richman 的分析能力被设计成"量化层 + LLM 文案增强"两级结构，量化层（trend、position、catalyst、confidence、recommendation）完全不依赖 LLM，LLM 只负责把结构化结果翻译成自然语言文案。但目前的数据模型没有把这两级能力分离开——卡片没有任何字段记录"本次合成是 LLM 还是规则引擎产出的"，用户和后端都没有办法区分。

第二，当前的 LLM provider 只有一个系统级单例，由 `backend/cmd/server/main.go` 从 cfg 加载。没有用户级别的配置能力，这意味着要么所有用户共享同一个 API 额度，要么一个都不用。对付费用户或隐私敏感用户来说这个限制都不可接受。

第三，降级状态没有用户可见的反馈。用户看到的卡片可能是 LLM 解读的，也可能是模板生成的，两者在 UI 上完全一样，用户无法分辨为什么有时候建议很精致有时候只是一句套话。

本 PRD 一次性解决这三个问题：明确三态合成契约、引入用户级配置、给用户可见的降级反馈和回归路径。

## 产品目标

1. Richman 在任何 LLM 可用性场景下都能产出决策卡片，不会因为 LLM 缺席就拒绝分析
2. 用户可以在 /settings/llm 配置自己的 LLM provider（Claude / OpenAI / OpenAI 兼容第三方）
3. 用户可以一眼看出某张卡片的解读是 AI 生成还是规则生成，并知道如何升级
4. 后端可观测到 LLM 各层（user / system default / template）的使用占比和失败率，方便运维判断降级事件

## 非目标

- 不支持多 provider 同时 active（MVP 每个用户只能有一个活跃配置）
- 不支持主密钥轮换（密钥丢失需要用户重新填 key）
- 不支持模型级别的 A/B 测试或费用账单
- 不支持用户在卡片粒度上选择 provider（比如这只股用 Claude、另一只用 OpenAI）

## 用户故事

### US-1 新用户开箱即用
新用户注册 Richman，没有任何 LLM 配置经验。他希望：
- 不配置 LLM 也能看到分析结果（走 template fallback）
- 卡片上清晰标注"基于规则引擎"，不假装是 AI 解读
- Onboarding 有一步引导他了解 AI 解读的差异以及配置路径
- 如果他同意使用系统默认 LLM，可以在 onboarding 一步勾选

### US-2 配置自己的 Provider
熟悉 API 的用户想用自己的 Anthropic 账户获得更好的模型质量和额度控制。他希望：
- 在 /settings/llm 填 API key 和模型名
- 保存时系统帮他做一次连通性测试，失败立即反馈
- 保存成功后，他的后续分析用他的 key，不走系统默认
- 他的 key 被加密存储，不会在 log 或 API response 里泄露

### US-3 Key 失效时的降级
用户之前配好了自己的 OpenAI key，但某天 key 被管理员取消了。他不希望：
- 分析流水线直接崩溃
- 卡片状态没有任何提示
他希望：
- 分析继续跑，卡片基于规则引擎生成
- 设置页面能看到"key 失效"的红色警告
- 可选择"允许我的 key 失效时使用系统默认 LLM"，此时自动 fallback 到系统默认

### US-4 从降级态回归
用户配好了 LLM，但 dashboard 上的卡片还是昨天生成的 template 版本。他希望：
- Dashboard 顶部 banner 提示他有历史卡片可以升级为 AI 解读
- 点击按钮后一次性重新分析所有持仓
- 分析完成后 banner 消失，卡片刷新

### US-5 运维观察降级率
运维团队想知道系统级 LLM 的稳定性。他希望：
- Prometheus 指标区分三层 provider 的成功率和延迟
- log 里每个降级事件都有结构化记录
- 可以统计"过去 24 小时有多少卡片是 template"

## 三态 LLM 合成契约

### 核心字段

每张决策卡片新增两个字段，存储本次合成的真实来源：

`synthesis_source` 三取一的枚举：
- `llm`：文案字段和 recommendation 子对象全部由 LLM 生成并正常解析
- `mixed`：LLM 调用成功且文案 JSON 解析通过，但 recommendation 子对象缺失或非法，该部分回退到规则模板
- `template`：LLM 调用失败（超时、鉴权、网络、provider 完全不可用、JSON 整体解析失败），全部字段走 templateFallback 路径

`provider_used` 四取一的枚举：
- `user`：本次调用使用了用户配置的 provider
- `system_default`：本次调用使用了系统默认 provider
- `none`：本次没有任何 provider 参与，走 template
- `unknown`：历史数据（迁移前）或状态未知

两个字段正交。`synthesis_source` 回答"内容质量降级了吗"，供用户和前端消费；`provider_used` 回答"数据流向哪去了"，供审计和监控消费。

### 触发规则与现有代码路径的对应

合成路径完全复用当前 `synthesis.Synthesize` 的分支结构，不引入新分支：

- Provider 返回 ok + 文案 JSON 解析通过 + recommendation 子对象解析通过：`synthesis_source=llm`
- Provider 返回 ok + 文案 JSON 解析通过 + recommendation 子对象缺失/非法：`synthesis_source=mixed`
- Provider 返回 error 或 panic 或 provider 本身为 nil 或文案 JSON 解析失败：`synthesis_source=template`

### 与现有 BadgeState 的关系

`synthesis_source` 与 `diff.BadgeState` 完全正交。BadgeState 描述"本次卡片相对于前一次发生了什么变化"，是关于 delta 的；`synthesis_source` 描述"本次卡片的内容质量是什么水平"，是关于 state 的。两个字段的取值组合之间没有任何冲突约束。

`BadgeDataDegraded`（已有）专门代表数据源降级，与 LLM 降级是两个独立维度。一张卡片可以同时是 `BadgeDataDegraded + synthesis_source=llm`（数据有问题但 AI 还在解读）或 `BadgeNone + synthesis_source=template`（数据没变但 LLM 不可用）。

## 配置分层模型

### 两层配置

Richman 支持两层 LLM provider 配置，优先级从上到下：

层级 `user`：用户在 /settings/llm 自配置的 provider，仅对该用户生效
层级 `system_default`：管理员在 `.env` 配置的系统级 provider，作为所有用户的候选 fallback

### 支持的 provider_type

MVP 支持三种协议族：

`claude`：Anthropic Messages API，base_url 固定为 api.anthropic.com，用户只填 api_key 和 model
`openai`：OpenAI Chat Completions API，base_url 固定为 api.openai.com，用户只填 api_key 和 model
`openai_compatible`：OpenAI Chat Completions 协议，但 base_url 由用户自填，支持 DeepSeek、群星、自托管 vLLM、本地 llama.cpp 等所有兼容 OpenAI 协议的端点

后续若有新协议族（Gemini、Cohere 等），通过追加 enum 值的方式扩展，不改 schema。

### 隐私同意

用户自选 LLM 引入了隐私决策：当用户的 key 不可用时，他的数据是否可以走系统默认 provider？这个问题分两个场景，用两个独立的 consent 字段表达：

`use_system_default_when_unconfigured`：当用户完全没有配置自己的 provider 时，是否允许 Richman 用系统默认 provider 为他跑分析。默认 false。新用户 onboarding 时显式引导，勾选后才启用系统默认。

`fallback_to_system_default_on_user_failure`：当用户已配置自己的 provider 但调用失败时，是否自动 fallback 到系统默认。默认 false。Settings 页面上明示这个选项，勾选时附带隐私说明。

两个 consent 独立开关，用户可以选择"没配的时候用系统默认，但配了以后不降级"，或者"没配的时候 template，但配错了可以兜底"，各种组合都合法。

## Fallback 链

### 完整解析顺序

对于每次 `AnalyzeHolding` 调用，backend 按以下顺序解析 provider：

1. 查用户 `llm_configs`
   - 如果配置存在且 provider 被 resolver 成功实例化：尝试调用
     - 成功：`source ∈ {llm, mixed}`，`provider_used = user`
     - 失败：进入步骤 2（前提 `fallback_to_system_default_on_user_failure = true`）
     - 失败且 consent = false：进入步骤 3
   - 如果配置不存在：进入步骤 2（前提 `use_system_default_when_unconfigured = true`），否则进入步骤 3
2. 尝试系统默认 provider
   - 成功：`source ∈ {llm, mixed}`，`provider_used = system_default`
   - 失败或 system_default 未配置：进入步骤 3
3. `templateFallback`
   - 必然成功：`source = template`，`provider_used = none`

### 错误归一化

Provider 底层的各种错误被归一化为几类，每类的处置是"进下一级"：

`ErrProviderUnavailable`：连接失败、DNS 错、TLS 错、密钥解密失败
`ErrProviderAuthFailed`：HTTP 401/403，且标记该层级健康状态为 `failing`
`ErrProviderRateLimited`：HTTP 429
`ErrProviderBadResponse`：HTTP 5xx 或响应体 JSON 解析失败
`ErrProviderTimeout`：context deadline（允许一次重试）

降级事件不回传前端错误码，只更新卡片的 `synthesis_source` 和 `provider_used`，配合设置页面显示用户 key 的 health 状态。用户看到的直接信号是"卡片角标是 AI 还是 Rules"，不是 HTTP 错误弹窗。

## UX 表面

### Settings 页 LLM Section

位置：`/settings/llm`

三种状态对应三种布局：

未配置：单行说明 + 空状态图 + "添加 LLM Provider"按钮。下方 callout 提示如果系统默认可用且用户已同意 `use_system_default_when_unconfigured`，那么分析会走系统默认，否则走规则引擎。

健康：显示 provider 品牌徽标、模型名、api_key_hint（最后 4 位 + 前缀"..."）、health 状态（绿色"健康"+ 最后 probe 时间）、`fallback_to_system_default_on_user_failure` 开关。下方三个按钮：测试连通性 / 编辑 / 删除。

失效：显示 provider 品牌徽标、红色"失效"+ 失效原因简报 + 最后 probe 时间、当前的 fallback 行为说明。下方三个按钮：重新测试 / 编辑 / 删除。

### Dashboard Banner

出现条件：后端 `GET /api/v1/dashboard/summary` 返回 `llmStatus.needsReanalysis = true`。判定逻辑：用户 active holdings 里存在 `synthesis_source IN (template, mixed)` 的最新卡片，同时用户当前 LLM 配置处于 healthy 状态（或系统默认可用且用户已同意系统默认）。

banner 样式：dashboard 顶部一条不侵入的信息带，左侧图标 + 文案"AI 解读已配置，你有 N 个持仓的历史卡片仍基于规则引擎"，右侧"重新分析所有持仓"按钮 + 关闭 X。

关闭行为：仅写入 sessionStorage，下次登录仍会出现。这是有意设计——问题还在，用户不应该有"永久关闭"的逃生口。

点击"重新分析所有持仓"：调用 `POST /api/v1/analysis/reanalyze-all`，返回 task_id，前端轮询任务状态直到完成。

### 卡片角标

每张决策卡片右上角新增一个 pill，根据 `synthesis_source` 显示：

`llm`：蓝色实心圆角 pill，文案"AI"
`mixed`：蓝色虚边圆角 pill，文案"Mixed"
`template`：灰色实心圆角 pill，文案"Rules"
`unknown`：不显示 pill

pill hover 展示 tooltip，分别解释三种状态的含义和当前 provider。hover 是桌面端交互；移动端 pill 可点击打开一个简短解释 modal。

### Onboarding 引导

新用户 onboarding 新增一步"AI 解读设置"，展示两个选择：

选项 A：跳过，使用规则引擎分析。卡片会显示 Rules 角标。你可以之后在设置里配置 LLM。
选项 B：我想试试 AI 解读。此时分两条路：
- 如果系统默认 provider 可用：显示一个简短隐私说明（"你的持仓数据将以加密传输方式发给 Richman 的默认 AI provider 做分析"），用户勾选"我同意"后 `use_system_default_when_unconfigured = true`
- 如果系统默认不可用：直接引导到"配置自己的 LLM provider"表单

此步骤可跳过，跳过即选 A。

## 数据模型

### 新表 `llm_configs`

字段设计详见 TRD 的 DB Schema 段落。关键 invariant：每个用户最多一条 active 配置（`UNIQUE (user_id) WHERE is_deleted = false`）；`api_key_cipher` 与 `api_key_nonce` 必须同生同死；`use_system_default_when_unconfigured` 和 `fallback_to_system_default_on_user_failure` 默认 false。

### 扩展 `decision_cards`

新增两列：`synthesis_source VARCHAR(16)` 可空、`provider_used VARCHAR(32)` 可空。历史行初始 NULL，迁移脚本一次性回填为 `synthesis_source='llm', provider_used='user'`（乐观回填：历史部署被认为是 LLM 驱动的）。

## 可观测性

Prometheus 指标最小集合：

- `llm_request_total{layer, provider, status}`：分层的成功率
- `llm_fallback_total{from, to}`：降级事件计数
- `llm_request_duration_seconds{layer, provider}`：响应延迟直方图
- `decision_cards_by_source{source}`：近 24h 产出卡片按 source 分布

日志：每次 resolver 调用输出一条结构化日志，字段包括 user_id、asset_code、layer_attempted 列表、layer_used、synthesis_source、duration_ms、fallback_reason（如果发生降级）。

**所有日志点严格不输出 api_key 明文或 api_key_cipher**。TRD 会规定 `LLMConfig.Masked()` 方法为唯一允许的序列化入口，代码评审时重点检查。

## 安全

### 密钥存储

用户 API key 使用 AES-256-GCM 加密后存储。主密钥 `LLM_CONFIG_MASTER_KEY` 从环境变量加载，长度必须是 32 字节（64 字符 hex），服务启动时校验，启动后常驻内存。每次加密生成随机 12 字节 nonce，密文和 nonce 分列存储。

解密只发生在 provider 调用前的瞬间，调用结束后让 GC 回收。禁止把 plaintext key 放进 context、日志、错误信息、指标 label 或任何跨请求缓存。

### SSRF 防护

`openai_compatible` 允许用户填 base_url，必须在保存时做严格校验：

scheme 必须是 https（禁止 http、file、ftp、gopher 等）
DNS 解析后检查所有 A/AAAA 记录不在以下范围：
- 10.0.0.0/8
- 172.16.0.0/12
- 192.168.0.0/16
- 127.0.0.0/8
- 169.254.0.0/16（AWS metadata / link-local）
- fc00::/7
- ::1/128
- fe80::/10

hostname 黑名单：localhost、metadata.google.internal、169.254.169.254（EC2 metadata）、任何以 .local 结尾的域名

同样的校验必须在每次 probe 和每次实际 ChatCompletion 调用前都跑一遍，不能只在保存时校验——避免 DNS rebinding 攻击。

详见 TRD 的 "SSRF Hardening" 段落。

### Rate Limiting

| 端点 | 限额 | 理由 |
|---|---|---|
| `PUT /api/v1/settings/llm` | 10/min/user | 防止重复提交 |
| `POST /api/v1/settings/llm/probe` | 30/min/user | 用户可能反复测试不同 key |
| `POST /api/v1/analysis/reanalyze-all` | 1/10min/user | 重分析消耗大量 token，防误点触发连锁 |

复用现有的中间件实现。

## 状态空间完整枚举

三维组合：用户 provider 健康度 × 系统默认可用性 × 两个 consent 字段。

| # | user | sys_default | use_sd_unconfig | fb_on_fail | source | provider_used | 分类 |
|---|---|---|---|---|---|---|---|
| 1 | healthy | * | * | * | llm/mixed | user | Valid |
| 2 | failing | exists | * | true | llm/mixed | system_default | Valid (fallback 激活) |
| 3 | failing | exists | * | false | template | none | Valid (用户拒绝 fallback) |
| 4 | failing | absent | * | * | template | none | Valid |
| 5 | absent | exists | true | * | llm/mixed | system_default | Valid |
| 6 | absent | exists | false | * | template | none | Valid (新用户默认 consent=false) |
| 7 | absent | absent | * | * | template | none | Valid |

`*` 代表该字段在该行的结果里无影响。`use_sd_unconfig` 只在 user=absent 时起作用，`fb_on_fail` 只在 user=failing 时起作用，两个 consent 在语义上互不重叠。

## 用户体验的替代路径

| 路径 | 设计如何处理 |
|---|---|
| Back 导航中途放弃配置 | 前端表单 local state，未点 Save 不写库 |
| probe 失败后重试 | 表单保留用户输入，仅在 probe 通过 + Save 后写库 |
| Cross-session resumption | 配置落库，新 session 加载 DB |
| 双 tab 同时改配置 | last-write-wins（`updated_at` 时间戳） |
| banner 关闭后重现 | 关闭只写 sessionStorage，下次登录问题仍在则重现 |
| reanalyze-all 与 edit 并发 | reanalyze 使用触发时刻快照的 config，edit 仅影响后续 |
| 用户删除配置时正在分析 | 分析使用启动时快照的 resolver，删除仅影响新分析 |
| 主密钥轮换 | MVP 不支持；文档写清楚"密钥丢失 = 用户需要重新填 key" |

## 未来工作

- 多 provider 配置：用户可以保存多套配置并在分析时选择
- 按持仓/按资产类型选 provider
- 密钥轮换：在新老密钥并存期间解密旧密文、加密新密文
- Billing 和配额：用户可以看到自己的 token 消耗
- Provider 自动重连：exponential backoff + circuit breaker
- DeepSeek 等特定 provider 的 quirks 支持

## 决策记录

| # | 决策 | 日期 | 理由 |
|---|---|---|---|
| D1 | 始终出卡片，降级态只加标签 | 2026-04-09 | 量化层能独立工作，LLM 只做文案增强，不该阻断核心功能 |
| D2 | `synthesis_source` 三态 + `provider_used` 四态独立字段 | 2026-04-09 | 两个字段正交，用户维度和运维维度解耦 |
| D3 | UI 标签 AI / Rules / Mixed（英文） | 2026-04-09 | 清晰简洁，Rules 比 Function 更直观 |
| D4 | 支持 claude / openai / openai_compatible 三协议族 | 2026-04-09 | 覆盖 MVP 需求，`openai_compatible` 自带扩展性 |
| D5 | 系统默认 + 用户覆盖分层 | 2026-04-09 | 新用户开箱即用，高级用户可隔离 |
| D6 | 两个独立 consent 字段 | 2026-04-09 | 避免"未配置时走系统默认"和"失败时 fallback 系统默认"语义混淆 |
| D7 | fallback 三级链：user -> system_default -> template | 2026-04-09 | 保持分析永远能完成，降级逐级明确 |
| D8 | 历史卡片一次性回填 `source='llm'` | 2026-04-09 | 避免老卡全部触发 reanalysis banner |
| D9 | Synthesizer 接口改为 `(*Output, *Meta, error)` | 2026-04-09 | 显式胜过 ctx.Value 穿管 |
| D10 | Reanalyze 单次不限 holdings 数 | 2026-04-09 | 用户自己控制成本，后端只做 1/10min 节流 |
| D11 | 不支持主密钥轮换 | 2026-04-09 | MVP 妥协，接受运维限制 |
| D12 | SSRF 硬防护 base_url（https + IP block + hostname 黑名单 + 每次调用都重校） | 2026-04-09 | 防 DNS rebinding，防内网探测 |
