# Step 14: Frontend Foundation

> Phase 4 | 并行组 R8 (单独执行) | 前置: Steps 1, 3 完成（API 契约在 TRD 中已定义）

## 任务目标

重构前端基础设施：HTTP client 拆分为 requestV1/requestV2/requestPublic 三函数、路由表重写（新增 /market, /market/:code, /settings/risk-preference，移除 /onboarding/*, /decision-cards/:id）、路由守卫简化（移除 OnboardingGuard）、导航栏更新、废弃代码清理、现有 feature 调用点迁移到 requestV1。

## 涉及文件

### 创建

无新建文件（本 step 为重构 + 删除）

### 修改

- `frontend/src/domain/http/client.ts` -- 拆分 requestV1 / requestV2 / requestPublic
- `frontend/src/config/routes.tsx` (或路由配置文件) -- 路由表重写
- `frontend/src/domain/auth/` -- AuthGuard 简化（移除 onboarding 检查）
- `frontend/src/layouts/` -- 导航栏更新（行情/持仓/投研简报 三项）
- 全部使用 request() 的 feature api.ts 文件 -- 迁移到 requestV1()
- `frontend/src/domain/storage/` -- localStorage key 前缀统一迁移 (richman_ 前缀)

### 删除

- `frontend/src/pages/onboarding/` -- 整个目录
- `frontend/src/domain/auth/onboarding-guard.tsx`
- `frontend/src/pages/decision-cards/` -- 整个目录（功能合并到标的详情页）

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| v2 路由表 | SS4/SS5/SS6 | frontend SS2.1 |
| 移除的路由 | SS1.7 零 onboarding | frontend SS2.2 |
| 导航栏 | SS4 Market Overview | frontend SS2.3 |
| 路由守卫简化 | SS1.7 | frontend SS2.4 |
| HTTP client 拆分 | - | frontend SS8.1 |
| 现有 feature 迁移 | - | frontend SS16.1 |
| localStorage 前缀 | - | frontend SS16.7 |
| dashboard-llm-status 处置 | - | frontend SS16.8 |
| react-helmet barrel 豁免 | - | frontend SS16.2 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G3.1 HTTP client 迁移 | request -> requestV1 全量替换 | frontend SS16.1 |
| G3.2 react-helmet barrel 豁免 | Biome 规则豁免配置 | frontend SS16.2 |
| G3.7 localStorage 前缀 | 统一迁移到 richman_ 前缀 | frontend SS16.7 |
| G3.8 dashboard-llm-status | 评估是否迁移或废弃 | frontend SS16.8 |

- requestV2 自动在 URL 前加 /api/v2/ 前缀
- requestPublic 不携带 JWT token
- 所有现有 feature 的 api.ts 调用点从 request() 改为 requestV1()
- 删除 onboarding 相关全部文件（pages + guard + components）
- 删除 decision-cards 页面（功能移至标的详情页 Step 16）
- dashboard-summary feature 暂保留（Step 17 重构为 research-briefing）
- 安装 @dr.pogodin/react-helmet 依赖（SEO 用）
- 路由配置中 `/` 重定向到 `/market`

## 验证标准

- [ ] `cd frontend && pnpm lint:all` 全部通过
- [ ] `pnpm build` 成功
- [ ] 无任何文件引用 onboarding-guard
- [ ] grep 确认无 request() 直接调用（全部为 requestV1/V2/Public）
- [ ] 路由表包含 /market, /market/:code, /settings/risk-preference
- [ ] 路由表不包含 /onboarding/*, /decision-cards/:id
- [ ] 导航栏未登录显示 [行情] [登录] [注册]，已登录显示 [行情] [持仓] [投研简报]
- [ ] pages/onboarding/ 和 pages/decision-cards/ 目录不存在

## 变更点清单覆盖

E9.1-E9.9 (9), E10.1-E10.3 (3), E13.1-E13.3 (3), G3.1 (1), G3.2 (1), G3.7 (1), G3.8 (1) = **19 项**
