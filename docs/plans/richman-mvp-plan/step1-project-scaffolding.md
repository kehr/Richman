# Step 1: Project Scaffolding

## 任务目标

搭建 monorepo 骨架，初始化前后端项目，配置开发工具链和本地开发环境。

## 涉及文件路径

### 创建

- `frontend/package.json` -- Next.js 项目配置
- `frontend/next.config.ts` -- Next.js 配置
- `frontend/tsconfig.json` -- TypeScript 配置
- `frontend/biome.json` -- Biome lint + format 配置
- `frontend/.dependency-cruiser.cjs` -- 前端架构边界检查
- `frontend/src/app/layout.tsx` -- Root layout
- `frontend/src/app/page.tsx` -- Root page (redirect)
- `frontend/src/ui-kit/eat/index.ts` -- Ant Design barrel 导出
- `backend/go.mod` -- Go module
- `backend/cmd/server/main.go` -- Go 服务入口（空壳）
- `backend/Dockerfile` -- Docker 构建文件
- `docker-compose.yml` -- 本地开发环境（PostgreSQL）
- `.gitignore` -- 全局忽略规则

### 修改

- `CLAUDE.md` -- 更新 dev commands 为实际可用命令

## PRD/TRD 章节引用

- PRD 5.1 整体架构
- PRD 5.2 前端技术栈
- PRD 5.3 后端技术栈
- `docs/standards/naming.md` 文件命名规范
- `docs/standards/frontend.md` 目录结构

## 验证标准

- [ ] `cd frontend && pnpm install` 成功
- [ ] `cd frontend && pnpm dev` 启动成功，浏览器可访问
- [ ] `cd frontend && pnpm lint` 通过
- [ ] `cd frontend && pnpm type-check` 通过
- [ ] `cd backend && go build ./...` 编译通过
- [ ] `docker-compose up -d` PostgreSQL 容器启动成功
- [ ] ui-kit/eat barrel 能正确导出 Ant Design 组件
- [ ] Biome noRestrictedImports 规则生效（直接 import antd 报错）

## 依赖说明

- 无前置依赖，这是第一个 step
- 需要 Node.js 22+, pnpm, Go 1.22+, Docker
