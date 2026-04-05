# Richman PRD

## 1. 产品概述

### 1.1 产品名称

Richman

### 1.2 产品定位

AI 驱动的个人投研决策助手。

### 1.3 核心价值

把专业基金经理的思维方式装进普通人口袋 -- 不做信息聚合，不做交易执行，专注"基于你的持仓，告诉你该怎么做、以及为什么"。

### 1.4 目标用户

有钱、有逻辑、但缺少投资经验的城市白领。

用户特征：
- 月收入较高，持有一定规模投资资产
- 逻辑思维强但不懂专业投研方法论
- 希望得到有依据的决策建议而非模糊推荐

### 1.5 商业化模式

- MVP 阶段：邀请制，需邀请码注册
- 后续阶段：免费 + 付费订阅（freemium）

### 1.6 国际化

- MVP：中文 + 英文
- 后续：按需扩展更多语言


## 2. 支持标的类型

### 2.1 黄金 / 黄金 ETF

| 维度 | 量化数据 | 实时信息 |
|------|---------|---------|
| 趋势 | 金价均线、动量指标 | - |
| 位置 | 实际利率、美元指数 | - |
| 催化剂 | - | 央行购金、Fed 表态、地缘事件 |

### 2.2 A 股宽基 ETF（沪深 300、中证 500 等）

| 维度 | 量化数据 | 实时信息 |
|------|---------|---------|
| 趋势 | 均线信号、RSI、MACD | - |
| 位置 | PE/PB 历史分位、北向资金 | - |
| 催化剂 | - | 政策信号、宏观定调 |

### 2.3 行业 ETF

| 维度 | 量化数据 | 实时信息 |
|------|---------|---------|
| 趋势 | 行业相对强弱、动量 | - |
| 位置 | 景气度、拥挤度 | - |
| 催化剂 | - | 产业政策、龙头财报 |

### 2.4 美股

| 维度 | 量化数据 | 实时信息 |
|------|---------|---------|
| 趋势 | 均线信号、动量指标 | - |
| 位置 | 席勒 PE（CAPE）、VIX | - |
| 催化剂 | - | 财报季、Fed 利率路径 |


## 3. 核心功能模块

### 3.1 持仓管理

**录入方式：** 纯手动录入（MVP 不支持导入或券商 API 对接），支持两种录入模式：

- 快速模式：直接填入当前均价成本和仓位比例，适合快速开始
- 明细模式：录入历史买卖交易记录，系统自动计算成本和仓位

两种模式可以混合使用。用户首次可通过快速模式录入当前成本，后续加减仓通过明细模式记录，系统基于全部记录重新计算综合成本。

**标的选择交互：** 先按类型分类浏览（黄金 / A 股宽基 / 行业 / 美股），再在分类内搜索选择

**标的库范围：**
- MVP 阶段：每类标的精选主流品种（如沪深 300 ETF、中证 500 ETF、纳斯达克 100 ETF、SPDR 黄金 ETF 等），总计约 30-50 个
- 后续阶段：逐步扩展覆盖范围
- 初始化方式：通过数据库 seed 脚本（`backend/db/seed/asset_catalog.sql`）导入，后续可通过管理后台维护

**快速模式录入字段：**
- 标的名称（从预设标的库选择）
- 当前均价成本
- 当前仓位比例（占总资金 %）

**明细模式录入字段（加减仓记录）：**
- 时间
- 价格
- 数量
- 方向（买入 / 卖出）

**自动计算：**
- 综合持仓成本（加权平均，基于快速录入成本 + 明细记录综合计算）
- 当前浮盈亏
- 历史操作记录

**限制：** MVP 每用户最多 5 个持仓标的

### 3.2 三维分析引擎（Plan B+）

#### 3.2.1 架构设计

采用"量化底座 + LLM 增强"的双层架构：

**Layer 1 - 量化底座（确定性）：** 三个维度全部有量化基础计算，保证分析的稳定性和可复现性。

**Layer 2 - LLM 增强（催化剂维度）：** 在催化剂维度的量化底座之上，叠加 LLM 联网搜索能力，发现非结构化事件并做语义解读。LLM 增强是叠加而非替代。

#### 3.2.2 三个维度

**趋势维度 -- 当前价格走势方向**
- 量化信号：均线交叉（MA5/MA20/MA60）、RSI、MACD、动量评分
- 输出：趋势方向（上行 / 震荡 / 下行）+ 强度评分

**位置维度 -- 当前估值/价格是否合理**
- 量化信号：PE/PB 历史分位数、CAPE（美股）、实际利率（黄金）、历史价格区间
- 输出：位置评估（低估 / 合理 / 高估）+ 分位数值

**催化剂维度 -- 近期是否有明确驱动事件**
- 量化底座：Polymarket 事件概率、系统预设关键词库匹配（维护一组按标的类型分类的关键词列表，用于在 LLM 搜索结果中识别相关事件，具体关键词列表和匹配规则在 TRD 中定义）
- LLM 增强：联网搜索实时事件、语义解读事件影响、叙事背景分析
- 输出：催化剂方向（利多 / 中性 / 利空）+ 关键事件摘要

#### 3.2.3 权重机制

产品为每类标的预设基础权重和允许微调范围：

| 标的类型 | 趋势基础权重 | 位置基础权重 | 催化剂基础权重 | 微调幅度 |
|---------|------------|------------|--------------|---------|
| 黄金 ETF | 25% | 30% | 45% | +/-10% |
| A 股宽基 ETF | 30% | 40% | 30% | +/-10% |
| 行业 ETF | 35% | 30% | 35% | +/-10% |
| 美股 | 30% | 35% | 35% | +/-10% |

- LLM 在基础权重 +/-10% 范围内根据当前市场环境微调（如基础 30% 的维度，可调至 20%-40%）
- 三维权重之和始终为 100%
- 权重调整需在决策卡中透明展示（展示当前实际权重及调整理由）

#### 3.2.4 决策矩阵

三维交叉输出五级建议：
- 积极加仓
- 小幅加仓
- 持有等待
- 分批减仓
- 控制仓位

#### 3.2.5 信心度

每张决策卡包含信心度评分（0-100%），反映三维一致性和数据完整性程度。

计算逻辑：
- 三维方向完全一致：基础信心度 80-100%
- 两维一致、一维冲突：基础信心度 50-70%
- 三维方向各异：基础信心度 20-40%
- 数据不完整扣减：缺少一个维度数据扣 20%，LLM 催化剂增强不可用扣 10%

具体计算公式在 TRD 中定义。

#### 3.2.6 数据更新频率

| 数据源 | 更新频率 | 说明 |
|--------|---------|------|
| AKShare | 日级 | 每日收盘后拉取，用于趋势和位置计算 |
| Yahoo Finance | 日级 | 每日拉取美股/黄金收盘数据 |
| Polymarket API | 日级 | 每日分析前拉取最新事件概率 |
| LLM 联网搜索 | 每次分析时 | 分析触发时实时搜索 |

#### 3.2.7 降级策略

- LLM 不可用时：催化剂维度仅使用量化底座（Polymarket 概率），信心度相应下调
- 行情数据源故障时（AKShare / Yahoo Finance）：受影响维度标记为"数据不足"，不输出该维度评分，使用最近一次有效数据标注数据时间
- Polymarket 不可用时：催化剂量化底座降级为仅 LLM 搜索，信心度下调

### 3.3 决策卡

每个持仓生成一张结构化决策卡。

**卡片内容：**
- 标的基本信息（成本、仓位、浮盈亏）
- 三维判断摘要（趋势 / 位置 / 催化剂各一句话结论）
- 信心度评分
- Polymarket 相关市场概率（如适用）
- 当前操作建议（持有 / 加仓 / 减仓）
- 具体操作点（两层结构）：
  - 默认层：方向 + 逻辑说明
  - 展开层：具体价格区间 + 触发条件 + 推理过程
- 主要风险提示（1-2 条）
- 今日要点（基于最新一次分析的简短状态更新，说明与上次分析相比有什么变化）

**展示模式：** 简洁模式 / 详细模式可切换

**交互方式：** MVP 纯卡片展示，不支持 AI 对话追问

**风险声明：** 页面显著位置标注"仅供参考，不构成投资建议"

### 3.4 每日推送通知

**推送频率和时区：**

所有推送时间以北京时间（CST, UTC+8）为准：

| 推送 | 时间 | 覆盖标的 | 内容 |
|------|------|---------|------|
| 早盘前 | 08:30 | A 股相关标的（宽基 ETF、行业 ETF） | 开盘前决策参考 |
| 收盘后 | 15:30 | A 股相关标的 + 黄金 ETF | 收盘后分析总结 |
| 美股收盘后 | 06:00（次日） | 美股标的 | 美股收盘后分析总结 |

如果用户同时持有 A 股和美股标的，每日最多收到 3 次推送。只持有 A 股标的的用户每日 2 次，只持有美股标的的用户每日 1 次。黄金 ETF 跟随 A 股推送时段。

**推送内容：** 对应时段相关标的的决策卡摘要

**MVP 推送渠道：**
- 微信公众号（模板消息）
- 飞书机器人（卡片消息）
- 邮件（HTML 邮件）

**后续扩展渠道：**
- 钉钉机器人（Webhook）
- Slack（Webhook）
- Telegram（Bot API）
- iOS APNs
- 更多渠道

**架构设计：** 可插拔适配器模式
- 统一接口：send(userId, message, channels[])
- 每个渠道实现一个 adapter
- 新增渠道只需添加一个 adapter 文件，零改动核心逻辑


## 4. 账户与权限体系

### 4.1 账户系统

- 独立账户体系：邮箱注册 + 密码登录
- MVP 需要邀请码才能注册
- 后续打通第三方 OAuth（微信、Google 等）
- 后续增加手机号注册和验证码登录

### 4.2 权限模型（MVP 搭骨架）

**用户角色：**
- 普通用户（MVP）
- 管理员（预留）

**订阅计划：**
- invite（MVP）-- 邀请用户享受全部功能
- free / pro / premium（后续）

**功能额度（按计划绑定）：**
- 持仓上限
- 每日分析次数
- 可用推送渠道数
- LLM 模型选择

**权限检查：** API 层统一中间件，根据用户当前 plan 判断是否有权调用对应接口


## 5. 技术架构

### 5.1 整体架构

前后端分离，Go 后端提供纯 RESTful JSON API，所有客户端消费同一套 API。前端架构借鉴 Orbiter 项目的 Pages + Features 双层模式。

### 5.2 前端

| 项 | 选型 |
|---|---|
| 框架 | Next.js 15 (App Router) |
| UI 组件库 | Ant Design 6 + @ant-design/pro-components |
| 样式方案 | Ant Design CSS-in-JS + CSS Variables（不用 Tailwind） |
| 数据获取 | TanStack Query v5（服务端状态管理） |
| 客户端状态 | React hooks（useState / useReducer / Context） |
| 语言 | TypeScript (strict mode) |
| i18n | next-intl（中文 + 英文） |
| 主题 | 亮色/暗色（Ant Design ConfigProvider 默认配置） |
| 包管理 | pnpm |
| Lint/Format | Biome（tab 缩进、100 字符行宽、双引号、始终分号） |
| 架构检查 | dependency-cruiser（层间依赖约束） |
| 部署 | Vercel |

#### 5.2.1 前端架构（Pages + Features 双层模式）

借鉴 Orbiter 项目的成熟架构模式：

**依赖流向：** App -> config -> pages -> features -> domain -> ui-kit/eat

| 层 | 职责 | 可依赖 | 不可依赖 |
|---|------|--------|---------|
| config/ | 路由和主题配置 | pages（引用）、ui-kit/eat | features、domain |
| pages/ | 页面组装（纯组合） | features/*/index（barrel）、domain、ui-kit/eat | feature 内部文件 |
| features/ | 自包含业务模块 | domain、ui-kit/eat | 其他 features、pages |
| domain/ | 跨模块基础设施 | ui-kit/eat、第三方库 | features、pages |
| layouts/ | 页面布局 | config、ui-kit/eat | features、domain、pages |
| ui-kit/ | Ant Design 封装 | antd 包 | 业务代码 |

**Feature 模块结构：**
每个 feature 包含 api.ts（API 函数 + DTO 类型）、useXxx.ts（TanStack Query hooks）、index.ts（barrel 导出，仅公开 API）。

**UI 组件导入规则：**
所有 Ant Design 组件通过 ui-kit/eat barrel 导入，禁止直接从 antd / @ant-design/pro-components / @ant-design/icons 导入。由 Biome noRestrictedImports 规则强制执行。

**Pro 组件优先规则：**

| 场景 | 使用 |
|------|-----|
| 卡片容器 | Card（eat，默认 borderless） |
| 统计指标 | StatisticCard |
| 描述列表 | ProDescriptions |
| 数据表格 | ProTable |
| 页面布局 | ProLayout |

### 5.3 后端

| 项 | 选型 |
|---|---|
| 语言 | Go |
| Web 框架 | Gin |
| 数据库操作 | sqlc（类型安全 SQL 生成） |
| 数据库 | PostgreSQL（Supabase 或自建） |
| LLM | 多模型抽象层（Claude API / OpenAI API） |
| 定时任务 | Cron 调度器（早盘 + 收盘触发分析） |
| 部署 | Docker + VPS |

#### 5.3.1 后端三层架构

| 层 | 职责 | 可依赖 | 不可依赖 |
|---|------|--------|---------|
| API handlers | 参数校验、路由、响应格式 | service、config、middleware | repo、db |
| Service | 业务逻辑编排 | repo、config、外部服务 | API handlers |
| Repo | 数据访问（sqlc 生成） | db/query | service、API handlers |

### 5.4 数据源（MVP 免费数据优先）

| 数据源 | 用途 | 更新频率 |
|--------|------|---------|
| AKShare | A 股行情 / 估值数据 | 日级（收盘后拉取） |
| Yahoo Finance | 美股 / 黄金行情 | 日级（每日拉取） |
| Polymarket API | 宏观事件市场隐含概率 | 日级（分析前拉取） |
| LLM 联网搜索 | 实时叙事补充（催化剂增强层） | 每次分析时实时 |

### 5.5 API 设计原则

- 纯 RESTful JSON，客户端无关，camelCase 字段命名
- URL 路径 kebab-case 复数形式（如 /api/v1/decision-cards）
- 分页参数：page（默认 1）+ pageSize（默认 20，范围 1-100）
- 统一错误格式：{ error: { code, message, details } }
- Next.js Web App 是 API 的一个消费者，不与 SSR 耦合
- Go 后端已是独立服务，后续可根据流量拆分为微服务

### 5.6 代码质量保障（Lint 系统）

前后端各自配置完善的 lint 工具链，每次代码修改后必须通过 lint 检查才能进入下一步工作。

#### 5.6.1 前端 Lint 工具链

| 工具 | 用途 | 命令 |
|------|------|------|
| Biome | 代码格式化 + lint（替代 ESLint + Prettier） | `pnpm lint` / `pnpm format` |
| TypeScript | 静态类型检查（strict mode） | `pnpm type-check` |
| dependency-cruiser | 架构层间依赖约束检查 | `pnpm lint:deps` |

**Biome 规则重点：**
- 格式：tab 缩进、100 字符行宽、双引号、始终分号
- noRestrictedImports：禁止直接从 antd / @ant-design/pro-components / @ant-design/icons 导入，必须通过 ui-kit/eat barrel
- 推荐规则集 + 自定义规则
- import 自动排序

**dependency-cruiser 规则重点：**
- features 之间互相隔离
- domain 不依赖 features/pages
- ui-kit 不依赖业务代码
- layouts 限制依赖范围

**前端全量检查命令：** `pnpm lint:all`（Biome + type-check + dependency-cruiser，全部通过才算通过）

#### 5.6.2 后端 Lint 工具链

| 工具 | 用途 | 命令 |
|------|------|------|
| golangci-lint | Go 综合 lint（集成多个 linter） | `golangci-lint run ./...` |
| go vet | Go 官方静态分析 | `go vet ./...` |
| sqlc vet | SQL 查询静态检查 | `sqlc vet` |

**golangci-lint 启用的 linter：**
- govet -- 官方静态分析
- errcheck -- 未处理的 error 返回值
- staticcheck -- 高级静态分析
- unused -- 未使用的代码
- gosimple -- 代码简化建议
- gocritic -- 代码风格和性能
- gofmt / goimports -- 格式化和 import 排序
- misspell -- 拼写检查
- revive -- 可配置的 lint 规则

**golangci-lint 配置文件：** `backend/.golangci.yml`

**后端全量检查命令：** `cd backend && golangci-lint run ./... && go vet ./...`

#### 5.6.3 构建系统

前后端各自提供完善的一键命令，降低构建、运行和验证的操作成本。

**前端 scripts（package.json）：**

| 命令 | 用途 |
|------|------|
| `pnpm dev` | 启动开发服务器 |
| `pnpm build` | 生产构建 |
| `pnpm lint` | Biome lint |
| `pnpm format` | Biome format |
| `pnpm type-check` | TypeScript 类型检查 |
| `pnpm lint:deps` | dependency-cruiser 架构边界检查 |
| `pnpm lint:all` | 全部检查合并（lint + format + type-check + lint:deps） |
| `pnpm test` | 运行 Vitest 测试 |
| `pnpm test:watch` | Vitest watch 模式 |
| `pnpm test:coverage` | 测试覆盖率报告 |

**后端 Makefile：**

| 命令 | 用途 |
|------|------|
| `make dev` | 启动开发服务器（热重载，使用 air） |
| `make build` | 编译生产二进制 |
| `make lint` | golangci-lint + go vet |
| `make test` | 运行全部测试 |
| `make test-race` | 运行测试（含竞态检测） |
| `make test-cover` | 测试覆盖率报告 |
| `make sqlc` | 生成 sqlc 代码 |
| `make migrate-up` | 执行数据库迁移 |
| `make migrate-down` | 回滚数据库迁移 |
| `make docker-build` | 构建 Docker 镜像 |
| `make check` | 全部检查合并（lint + test + build） |

#### 5.6.4 配置管理

前后端通过 env 配置文件管理不同环境的配置，敏感信息不入版本库。

**配置文件结构：**

```
frontend/
  .env.example            # 模板（入库，包含所有变量名和说明）
  .env.local              # 本地开发覆盖（不入库）
  .env.dev                # 开发环境默认值（入库）
  .env.prod               # 生产环境默认值（入库，不含敏感值）

backend/
  .env.example            # 模板（入库）
  .env                    # 实际配置（不入库）
  configs/
    config.dev.yaml           # 开发环境配置（入库，非敏感）
    config.prod.yaml          # 生产环境配置（入库，非敏感）
```

**前端配置项（Next.js .env）：**

| 变量 | 说明 | 示例 |
|------|------|------|
| `NEXT_PUBLIC_API_BASE` | 后端 API 地址 | `http://localhost:8080/api/v1` |
| `NEXT_PUBLIC_APP_ENV` | 运行环境标识 | `dev` / `prod` |

前端仅通过 `NEXT_PUBLIC_` 前缀暴露给客户端的变量，敏感信息不放前端。

**后端配置项（Go .env）：**

| 变量 | 说明 | 敏感 |
|------|------|------|
| `APP_ENV` | 运行环境 | 否 |
| `SERVER_PORT` | HTTP 服务端口 | 否 |
| `DATABASE_URL` | PostgreSQL 连接串 | 是 |
| `JWT_SECRET` | JWT 签名密钥 | 是 |
| `LLM_PROVIDER` | 默认 LLM 提供商 | 否 |
| `CLAUDE_API_KEY` | Claude API Key | 是 |
| `OPENAI_API_KEY` | OpenAI API Key | 是 |
| `WECHAT_APP_ID` | 微信公众号 AppID | 是 |
| `WECHAT_APP_SECRET` | 微信公众号 Secret | 是 |
| `FEISHU_WEBHOOK_URL` | 飞书机器人 Webhook | 是 |
| `SMTP_HOST` | 邮件 SMTP 服务器 | 否 |
| `SMTP_PORT` | SMTP 端口 | 否 |
| `SMTP_USER` | SMTP 用户名 | 是 |
| `SMTP_PASSWORD` | SMTP 密码 | 是 |
| `LOG_LEVEL` | 日志级别 | 否 |
| `LOG_DIR` | 日志文件目录 | 否 |

**配置管理规则：**
- 所有配置通过 `internal/config/config.go` 集中加载，业务代码不直接读 `os.Getenv`
- `.env.example` 入库，包含全部变量名、说明和非敏感默认值
- `.env` 不入库（.gitignore），包含实际敏感值
- Docker 部署通过 `docker-compose.yml` 的 `env_file` 或环境变量注入
- Vercel 部署通过 Vercel Dashboard 的 Environment Variables 配置

#### 5.6.5 Lint 执行规则

- 每次修改代码后立即执行对应端的 lint，修复所有问题后才能继续
- 提交前必须前后端 lint 全部通过
- CI/CD 中 lint 作为必须通过的 gate

### 5.7 工程规范文档

工程规范文档放在 docs/standards/ 目录，借鉴 Orbiter 项目模式并适配 Go 后端：

- naming.md -- 文件、目录、标识符、数据库、API 命名约定
- frontend.md -- Pages + Features 架构、依赖规则、组件使用规范
- backend.md -- Go 四层架构、service/repo 模式、错误处理
- database.md -- PostgreSQL schema 约定、审计字段、索引策略
- api.md -- RESTful API 设计、版本管理、分页、错误格式
- testing.md -- 测试结构、命名、mock 策略
- logging.md -- 日志系统规范（zap、轮转、脱敏、采样）


## 6. 多平台路线

| 阶段 | 客户端 | 说明 |
|------|--------|------|
| MVP | Web App (Next.js) | Vercel 部署 |
| Phase 2 | iOS App + macOS App | 消费同一套 Go API |
| Phase 3+ | Android App + Windows App | 消费同一套 Go API |

架构原则：Go 后端提供纯 RESTful JSON API，所有客户端消费同一套 API，新增客户端不需要后端改动。


## 7. MVP 范围

### 7.1 MVP 包含

- 独立账户体系 + 邀请码注册
- 手动持仓录入（最多 5 个标的）
- 分类浏览 + 搜索选择标的
- 三维分析引擎（量化底座 + LLM 催化剂增强）
- 决策卡输出（简洁/详细切换，两层操作建议）
- 每日推送（微信公众号 / 飞书 / 邮件，A 股两次 + 美股一次，最多三次）
- 权限骨架（plan + quota 模型）
- 亮色/暗色主题
- 中文 + 英文国际化
- 风险免责声明

### 7.2 MVP 不包含（后续迭代）

- Excel/CSV 导入或券商 API 对接
- AI 对话 / 追问能力
- 异动驱动实时警报
- 历史决策胜率回测
- 社交分享
- iOS / macOS / Android / Windows 原生客户端
- 钉钉 / Slack / Telegram 推送
- 支付集成（Stripe 等）
- 第三方 OAuth 登录
