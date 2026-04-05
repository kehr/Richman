# Step 9: Integration Testing + Polish

## 任务目标

端到端集成测试、前后端联调验证、lint 全量检查、部署配置验证（Vercel + Docker）、风险声明确认、最终 polish。

## 涉及文件路径

### 创建

- `backend/internal/api/v1/integration_test.go` -- API 集成测试（完整流程）
- `frontend/src/features/decision-card/DecisionCardView.test.tsx` -- 决策卡组件测试
- `frontend/src/features/portfolio/usePortfolio.test.ts` -- 持仓 hooks 测试

### 修改

- `docker-compose.yml` -- 确认生产级配置
- `backend/Dockerfile` -- 确认多阶段构建
- `frontend/next.config.ts` -- 确认生产配置（API proxy 等）
- `README.md` -- 项目说明、本地开发指南、部署指南

## PRD/TRD 章节引用

- PRD 7.1 MVP 包含（逐项验证）
- PRD 7.2 MVP 不包含（确认未越界）
- `docs/standards/testing.md` 测试规范
- `docs/standards/frontend.md` 架构边界检查
- `docs/standards/logging.md` 日志输出验证

## 验证标准

### Lint 全量通过
- [ ] `cd frontend && pnpm lint` 零错误
- [ ] `cd frontend && pnpm type-check` 零错误
- [ ] `cd frontend && npx dependency-cruiser src --config .dependency-cruiser.cjs` 零违规
- [ ] `cd backend && golangci-lint run ./...` 零错误
- [ ] `cd backend && go vet ./...` 零警告

### 后端集成测试
- [ ] 完整流程：注册 -> 登录 -> 添加持仓 -> 触发分析 -> 查看决策卡
- [ ] 权限检查：未登录请求返回 401
- [ ] 持仓上限：第 6 个持仓被拒绝
- [ ] 降级场景：mock LLM 不可用，分析仍成功（催化剂降级）
- [ ] `go test ./...` 全部通过
- [ ] `go test -race ./...` 无竞态问题

### 前端验证
- [ ] `pnpm build` 构建成功无警告
- [ ] 所有页面可访问，无控制台错误
- [ ] 主题切换正常（亮色/暗色）
- [ ] 语言切换正常（中文/英文）
- [ ] 风险声明在所有含决策卡的页面展示

### 配置管理验证
- [ ] `frontend/.env.example` 和 `backend/.env.example` 变量完整且有说明
- [ ] 复制 `.env.example` 为 `.env` 后前后端均可启动
- [ ] `.gitignore` 正确忽略所有敏感 env 文件
- [ ] docker-compose.yml 通过 env_file 正确注入后端配置
- [ ] 生产环境配置（config.prod.yaml）不含敏感值

### 部署验证
- [ ] `docker-compose up` 后端服务正常启动（通过 env 配置）
- [ ] 后端健康检查端点 `GET /health` 返回 200
- [ ] 前端 `pnpm build` 输出可部署到 Vercel
- [ ] 前端环境变量 `NEXT_PUBLIC_API_BASE` 配置正确
- [ ] Vercel 部署通过 Dashboard Environment Variables 配置

### MVP 清单逐项确认
- [ ] 独立账户体系 + 邀请码注册
- [ ] 手动持仓录入（最多 5 个标的）
- [ ] 分类浏览 + 搜索选择标的
- [ ] 三维分析引擎（量化底座 + LLM 催化剂增强）
- [ ] 决策卡输出（简洁/详细切换，两层操作建议）
- [ ] 每日推送（微信公众号 / 飞书 / 邮件，三个时段）
- [ ] 权限骨架（plan + quota 模型）
- [ ] 亮色/暗色主题
- [ ] 中文 + 英文国际化
- [ ] 风险免责声明

## 依赖说明

- Step 8 完成（前端全部页面就绪）
- 所有后端 API 就绪
