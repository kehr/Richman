# Step 2: backend EventItem 字段扩展

## 目标

`internal/richson/types.go::EventItem` 增加三个指针字段，让 backend 透传 richson 新返回的 `sourceUrl` / `sourceName` / `releaseId`。无 handler / service / repo 改动。

## 涉及文件

修改：
- `backend/internal/richson/types.go`（仅 EventItem struct，加 3 个字段）

## 设计依据

- PRD §4.1（DTO 字段表，pointer 类型对齐 Pydantic `T | None`）
- TRD §3.1（types.go EventItem 改动 diff）
- TRD §3.2（不修改的范围）
- contract-drift.md（Go 必须用指针类型表达 nullable，绝不可用值类型 + 注释暗示）

## 验证标准

- `cd backend && go build ./internal/richson/...` 通过
- `cd backend && make check` 通过（含 golangci-lint v2 / vet / test）
- 启动 backend dev server (`make dev` 或 `go run ./cmd/api`) 后端口 8100 可访问
- 本地需 richson 已起（依赖 Step 1 完成）才能联调；本 step 内无需联调，只确认编译通过 + lint 通过即可

## 依赖

- 无前置依赖（与 Step 1、Step 3 完全独立）
- 联调验证延后到 Step 4

## Commit 拆分

单一改动，1 个 commit：
- `feat(backend): expose source url/name and release id on event item`
