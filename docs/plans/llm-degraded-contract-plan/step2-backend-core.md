# Step 2: Backend Core (Repo + Resolver)

## 任务目标

在 Step 1 的 DB 和底层工具之上，实现 `LLMConfigRepo`（配置持久化）和 `llm.Resolver`（三级 fallback 链的编排层）。这两层实现后，上层 Synthesizer 就可以从 Provider 升级到 Resolver 依赖。

## 涉及文件

### 新增

- `backend/internal/repo/llm_config_repo.go`
- `backend/internal/repo/llm_config_repo_test.go`
- `backend/internal/llm/resolver.go`
- `backend/internal/llm/resolver_test.go`
- `backend/internal/llm/errors.go`（如果现有 llm 包没有集中的 err 定义）

### 修改

- `backend/internal/repo/user_repo.go`：新增 `GetUseSystemDefaultConsent` 和 `SetUseSystemDefaultConsent`
- `backend/internal/llm/provider.go`：如有必要，把 common errors 提到 `errors.go`

## 设计依据

- PRD "Fallback 链/完整解析顺序"：三级 fallback 状态图
- PRD "配置分层模型"：user 优先、system_default 次之、consent 控制
- PRD "状态空间完整枚举" 表格：7 行 valid target
- TRD "LLM Provider 接口/Resolver" 段落：Resolver 接口、ResolvedResponse、layer 枚举
- TRD "Repo 层/llm_config_repo.go"：方法签名
- TRD "Repo 层/user_repo.go 扩展"：consent 读写
- TRD "加密模块" + "SSRF 防护"：Resolver 构造时依赖

## 验证标准

### LLMConfigRepo 单元测试

- `GetActiveByUserID` 返回正确的 config（mock db / fixture）
- 用户没有 active 配置时返回 `ErrConfigNotFound`
- `Upsert` 创建第一条配置成功
- `Upsert` 更新已有配置（同 user_id）时保持 `config_id` 不变，仅更新内容
- `SoftDelete` 把 `is_deleted` 置 true，`GetActiveByUserID` 返回 `ErrConfigNotFound`
- partial unique index 能拦住手动 insert 第二条 active 配置的尝试
- `UpdateHealth` 只更新 `health_status / last_probe_at / last_probe_error`，不触碰 cipher/nonce

### Resolver 单元测试（覆盖状态空间 7 行）

按 PRD 状态表逐行写 test case：

| Test | user | sys_default | consent 组合 | 期望 layer | 期望 err |
|---|---|---|---|---|---|
| healthy-user-first | healthy | any | any | user | nil |
| user-fails-fallback-on | failing | available | fb_on_fail=true | system_default | nil |
| user-fails-fallback-off | failing | available | fb_on_fail=false | - | ErrAllLayersFailed or user err |
| user-fails-no-sys | failing | nil | any | - | ErrAllLayersFailed |
| unconfigured-sys-consent-on | absent | available | use_sd=true | system_default | nil |
| unconfigured-sys-consent-off | absent | available | use_sd=false | - | ErrConsentDenied |
| unconfigured-no-sys | absent | nil | any | - | ErrAllLayersFailed |

每个 case mock `LLMConfigRepo` 和 `UserRepo`，注入 stub provider / stub system_default。

### user_repo 扩展单元测试

- `GetUseSystemDefaultConsent` 读取新列
- `SetUseSystemDefaultConsent` 写入 + 更新 updated_at

### 集成校验

- `make check`（backend）全绿
- `golangci-lint run ./...` 0 issues
- `go test ./internal/llm/... ./internal/repo/...` 全绿

## 依赖

- Step 1 已完成（需要 migration 和 model）

## 偏差处理

- 如果现有 repo 风格用 sqlc，Step 2 也应该用 sqlc 生成查询代码；如果用手写 pgx，保持一致
- Resolver 的 UserRepo 依赖可以用一个更窄的 interface，避免拖进整个 user repo 的方法集
- `ErrConfigNotFound` vs `pgx.ErrNoRows`：在 Repo 层做一次转换，避免把 pgx 错误泄露到 Resolver
- 测试里构造 stub provider 时，允许复用 Step 1 之前的 synthesizer_test 的 stubProvider 风格

## 预期产出

- `LLMConfigRepo` 完整实现 + 测试
- `Resolver` 完整实现 + 覆盖 7 行状态表的测试
- `user_repo` consent 方法 + 测试
- commit: `feat(repo): add llm_config_repo`
- commit: `feat(repo): add user llm consent accessors`
- commit: `feat(llm): add resolver with three-level fallback chain`
