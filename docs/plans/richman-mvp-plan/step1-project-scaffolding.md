# Step 1: Project Scaffolding

## 任务目标

搭建 monorepo 骨架，初始化前后端项目，配置完善的 lint 工具链和本地开发环境。

## 涉及文件路径

### 创建

**前端：**
- `frontend/package.json` -- Next.js 项目配置（含 lint/format/type-check/lint:deps/lint:all scripts）
- `frontend/next.config.ts` -- Next.js 配置
- `frontend/tsconfig.json` -- TypeScript 配置（strict mode）
- `frontend/biome.json` -- Biome lint + format 配置（tab、100 char、双引号、始终分号、noRestrictedImports）
- `frontend/.dependency-cruiser.cjs` -- 前端架构边界检查规则
- `frontend/src/app/layout.tsx` -- Root layout
- `frontend/src/app/page.tsx` -- Root page (redirect)
- `frontend/src/ui-kit/eat/index.ts` -- Ant Design barrel 导出
- `frontend/vitest.config.ts` -- Vitest 测试配置

**后端：**
- `backend/go.mod` -- Go module
- `backend/cmd/server/main.go` -- Go 服务入口（空壳）
- `backend/.golangci.yml` -- golangci-lint 配置（govet/errcheck/staticcheck/unused/gosimple/gocritic/gofmt/goimports/misspell/revive）
- `backend/Makefile` -- 后端常用命令（lint/test/build/run）
- `backend/Dockerfile` -- Docker 构建文件

**根目录：**
- `docker-compose.yml` -- 本地开发环境（PostgreSQL）
- `.gitignore` -- 全局忽略规则

### 修改

- `CLAUDE.md` -- 更新 dev commands 为实际可用命令

## PRD/TRD 章节引用

- PRD 5.1 整体架构
- PRD 5.2 前端技术栈
- PRD 5.3 后端技术栈
- PRD 5.6 代码质量保障（Lint 系统）
- `docs/standards/naming.md` 文件命名规范
- `docs/standards/frontend.md` 目录结构、Biome 配置

## 验证标准

### 前端 lint 工具链
- [ ] `cd frontend && pnpm install` 成功
- [ ] `cd frontend && pnpm dev` 启动成功，浏览器可访问
- [ ] `cd frontend && pnpm lint` 通过（Biome lint）
- [ ] `cd frontend && pnpm format` 通过（Biome format）
- [ ] `cd frontend && pnpm type-check` 通过（TypeScript strict）
- [ ] `cd frontend && pnpm lint:deps` 通过（dependency-cruiser）
- [ ] `cd frontend && pnpm lint:all` 通过（以上全部合并）
- [ ] Biome noRestrictedImports 规则生效（直接 import antd 报错）
- [ ] ui-kit/eat barrel 能正确导出 Ant Design 组件

### 后端 lint 工具链
- [ ] `cd backend && go build ./...` 编译通过
- [ ] `cd backend && golangci-lint run ./...` 通过
- [ ] `cd backend && go vet ./...` 通过
- [ ] `.golangci.yml` 配置了全部必要 linter

### 基础设施
- [ ] `docker-compose up -d` PostgreSQL 容器启动成功
- [ ] PostgreSQL 可连接

## 依赖说明

- 无前置依赖，这是第一个 step
- 需要 Node.js 22+, pnpm, Go 1.22+, Docker, golangci-lint
