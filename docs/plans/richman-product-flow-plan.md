# Richman 产品动线 Plan 总述

## 目标

把当前"6 个空菜单项"的 MVP 骨架，按 `docs/prds/richman-product-flow-prd.md` 和 `docs/trds/richman-product-flow-trd.md` 的设计，重构为有完整动线的产品。

## 设计依据

- PRD: `docs/prds/richman-product-flow-prd.md`
- TRD: `docs/trds/richman-product-flow-trd.md`
- 工程规范: `docs/standards/frontend.md` `docs/standards/backend.md` `docs/standards/database.md` `docs/standards/api.md` `docs/standards/testing.md`

## Phase 拆分

| Phase | 目标 | Step 范围 | 阶段产物 |
|---|---|---|---|
| Phase 1 后端基础 | 数据模型 + 算法 | step01 - step04 | 新数据库字段、recommendation 类型、徽章 diff、权重 bias |
| Phase 2 后端服务 | 4 个新 service | step05 - step08 | user_settings / vision / screenshot / onboarding |
| Phase 3 后端 API | DTO 对齐 | step09 | 全部新接口与现有接口字段更新 |
| Phase 4 前端基础 | 路由 / 守卫 / 组件库 | step10 - step12 | 路由重构、菜单精简、money hook、决策卡组件 |
| Phase 5 前端 Onboarding | 4 步引导 | step13 | 4 个 onboarding 页 |
| Phase 6 前端 Dashboard / 详情页 | 核心首屏 | step14 - step15 | DashboardPage 三区结构 + DecisionCardDetailPage 5 区块 |
| Phase 7 前端 Portfolio | 持仓管理 | step16 - step17 | 列表 + Drawer + 截图 modal + 交易记录 |
| Phase 8 前端 Settings / Help / Login | 辅助页 | step18 - step20 | Settings 4 tab + Help 内容 + Login 双栏 |
| Phase 9 验证 | 烟雾 + 全量 lint | step21 | 端到端走通、所有 lint/test/build 通过 |

## 依赖关系

```
Phase 1 (基础) ──┬─→ Phase 2 (服务) ──→ Phase 3 (API)
                  │                              │
                  └────────────────────┐         │
                                       ↓         ↓
Phase 4 (前端基础) ←─────────────────────────────┘
        │
        ├─→ Phase 5 (Onboarding)
        ├─→ Phase 6 (Dashboard / 详情页)
        ├─→ Phase 7 (Portfolio)
        └─→ Phase 8 (Settings / Help / Login)
                            │
                            ↓
                    Phase 9 (验证)
```

Phase 1 必须先做。Phase 2-3 完成后才能解锁 Phase 4。Phase 4 完成后 Phase 5-8 可并行。Phase 9 收尾。

## Step 列表

| Step | 文件 | 目标 |
|---|---|---|
| step01 | `step01-db-migrations.md` | 3 个迁移：decision_cards 结构化、users 字段、holdings category |
| step02 | `step02-recommendation-types-and-diff.md` | recommendation 包 + diff 包 + 单元测试 |
| step03 | `step03-synthesis-and-analysis-integration.md` | synthesis 扩展生成结构化建议，analysis service 集成 diff |
| step04 | `step04-weight-risk-preference-bias.md` | weight manager 接受 risk_preference bias |
| step05 | `step05-user-settings-service.md` | user_settings service + privacy guard + money attach 工具 |
| step06 | `step06-llm-vision-provider.md` | LLM Vision 抽象接口 + Claude 实现 |
| step07 | `step07-screenshot-service.md` | screenshot service + import-screenshot API |
| step08 | `step08-onboarding-service.md` | onboarding service + onboarding API |
| step09 | `step09-api-dto-alignment.md` | decision_card / portfolio / user / onboarding DTO 全部对齐 |
| step10 | `step10-frontend-routes-and-guards.md` | 路由重构、MainLayout 菜单精简、OnboardingGuard |
| step11 | `step11-money-hook-and-user-settings-feature.md` | domain/money hook + features/user-settings barrel |
| step12 | `step12-decision-card-component-library.md` | features/decision-card 4 个组件 |
| step13 | `step13-onboarding-pages.md` | 4 个 onboarding 页面 |
| step14 | `step14-dashboard-page.md` | DashboardPage 三区结构重写 |
| step15 | `step15-decision-card-detail-page.md` | DecisionCardDetailPage 5 区块 + 右侧 meta 栏 |
| step16 | `step16-portfolio-list-and-add-drawer.md` | PortfolioListPage 改造 + AddHoldingDrawer |
| step17 | `step17-screenshot-import-and-transactions.md` | ScreenshotImportModal + 交易记录子页 |
| step18 | `step18-settings-page.md` | SettingsPage 4 tab |
| step19 | `step19-help-page-and-i18n-content.md` | HelpPage + i18n 帮助内容 |
| step20 | `step20-login-page-redesign.md` | LoginPage 左右双栏 |
| step21 | `step21-end-to-end-verification.md` | 烟雾测试 + 全量 lint/test/build |

## 全局约束

每一步都必须满足：

1. **lint 通过**：每次修改文件后立即执行项目 lint，全部通过才能进入下一 step
   - 后端：`cd backend && make check`
   - 前端：`cd frontend && pnpm lint:all`
2. **零 AI 痕迹**：commit message 不带 Co-Authored-By、不提 AI/Claude
3. **测试覆盖**：核心 service / 算法必须有单元测试，前端核心组件必须有渲染或交互测试
4. **commit 粒度**：每个 step 至少一次 commit，复杂 step 内部可分多次小 commit
5. **不擅自越界**：每个 step 只触及该 step 文件清单内的文件，发现需要改其他文件时停下来记录在该 step 的"实施备注"小节

## 与 PRD/TRD 的回溯链

| Phase | 主要 PRD 章节 | 主要 TRD 章节 |
|---|---|---|
| Phase 1 | §3.5 §8 | §1.3 §2 §3 §5.1 §5.4 |
| Phase 2 | §4.3 §7 §8 | §4 §5 §6 |
| Phase 3 | §3 §5 §6 §8 | §2.5 §4.5 §5.3 §6.1 |
| Phase 4 | §1 §9 | §7.1 §7.2 §6.1 §7.4 |
| Phase 5 | §2 | §6.1 |
| Phase 6 | §3 §5 | §2 §3 §7.3 |
| Phase 7 | §4 | §4 §7.3 |
| Phase 8 | §6 §7 §2.1 | §5 §6.2 §7 |
| Phase 9 | 全部 | 全部 |

## 非目标

以下内容明确不在本 Plan 范围（PRD §10 已声明）：

- AI 对话追问能力
- 历史决策胜率回测
- 多币种总资金
- 持仓 Excel/CSV 导入
- iOS / macOS / Android / Windows 原生客户端
- Slack / Telegram / 钉钉 推送渠道
- 支付集成与订阅升级
- Help 全文搜索

## 执行说明

每个 step 文件包含：

- 任务目标（做什么）
- 涉及文件（创建/修改清单）
- TRD/PRD 章节引用（设计依据）
- 验证标准（怎么确认做完了）
- 依赖说明（前置 step）
- 预估提交（commit 颗粒度建议）

执行 agent 根据 TRD 设计和实际代码情况动态决策具体实现，**不在 Plan 文件里写代码级细节**（方法签名、属性列表、SQL、算法步骤、常量值等都属于 TRD 职责，已在 TRD 中明确）。
